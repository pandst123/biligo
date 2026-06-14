package biliauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
)

const (
	qrGenerateURL = "https://passport.bilibili.com/x/passport-login/web/qrcode/generate"
	qrPollURL     = "https://passport.bilibili.com/x/passport-login/web/qrcode/poll"
	navURL        = "https://api.bilibili.com/x/web-interface/nav"
	userAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)

type Client struct {
	httpClient *http.Client
}

type QRStartResult struct {
	OK               bool
	LoginURL         string
	QRCodeKey        string
	QRImageDataURL   string
	ExpiresInSeconds int
}

type QRPollResult struct {
	OK       bool
	Status   string
	Message  string
	Code     int
	Cookie   string
	Username string
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{httpClient: httpClient}
}

func (c *Client) StartQRLogin(ctx context.Context) (QRStartResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, qrGenerateURL, nil)
	if err != nil {
		return QRStartResult{}, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return QRStartResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return QRStartResult{}, fmt.Errorf("二维码生成接口返回状态码 %d", resp.StatusCode)
	}

	var payload qrGeneratePayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return QRStartResult{}, err
	}
	if payload.Code != 0 {
		return QRStartResult{}, errors.New(firstNonEmpty(payload.Message, "二维码生成失败"))
	}
	if payload.Data.URL == "" || payload.Data.QRCodeKey == "" {
		return QRStartResult{}, errors.New("二维码生成响应缺少必要字段")
	}

	png, err := qrcode.Encode(payload.Data.URL, qrcode.Medium, 256)
	if err != nil {
		return QRStartResult{}, err
	}

	return QRStartResult{
		OK:               true,
		LoginURL:         payload.Data.URL,
		QRCodeKey:        payload.Data.QRCodeKey,
		QRImageDataURL:   "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
		ExpiresInSeconds: 180,
	}, nil
}

func (c *Client) PollQRLogin(ctx context.Context, qrcodeKey string) (QRPollResult, error) {
	qrcodeKey = strings.TrimSpace(qrcodeKey)
	if qrcodeKey == "" {
		return QRPollResult{}, errors.New("qrcode_key 不能为空")
	}

	endpoint := qrPollURL + "?qrcode_key=" + url.QueryEscape(qrcodeKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return QRPollResult{}, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return QRPollResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return QRPollResult{}, fmt.Errorf("二维码轮询接口返回状态码 %d", resp.StatusCode)
	}

	var payload qrPollPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return QRPollResult{}, err
	}
	if payload.Code != 0 {
		return QRPollResult{
			OK:      false,
			Status:  "failed",
			Message: firstNonEmpty(payload.Message, "轮询登录失败"),
			Code:    payload.Code,
		}, nil
	}

	stateCode := payload.Data.Code
	message := firstNonEmpty(payload.Data.Message, "等待扫码")
	switch stateCode {
	case 0:
		cookie := CookiesToHeader(resp.Cookies())
		username, loggedIn, err := c.VerifyCookie(ctx, cookie)
		if err != nil {
			return QRPollResult{}, err
		}
		if !loggedIn {
			return QRPollResult{
				OK:      false,
				Status:  "failed",
				Message: "登录成功但 Cookie 验证失败",
				Code:    stateCode,
			}, nil
		}
		return QRPollResult{
			OK:       true,
			Status:   "confirmed",
			Message:  "登录成功",
			Code:     stateCode,
			Cookie:   cookie,
			Username: username,
		}, nil
	case 86101:
		return QRPollResult{OK: true, Status: "waiting_scan", Message: firstNonEmpty(message, "等待扫码"), Code: stateCode}, nil
	case 86090:
		return QRPollResult{OK: true, Status: "waiting_confirm", Message: firstNonEmpty(message, "已扫码，等待确认"), Code: stateCode}, nil
	case 86038:
		return QRPollResult{OK: false, Status: "expired", Message: firstNonEmpty(message, "二维码已过期"), Code: stateCode}, nil
	default:
		return QRPollResult{OK: false, Status: "failed", Message: message, Code: stateCode}, nil
	}
}

func (c *Client) VerifyCookie(ctx context.Context, cookie string) (string, bool, error) {
	cookie = NormalizeCookieHeader(cookie)
	if cookie == "" {
		return "", false, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, navURL, nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", "https://show.bilibili.com/")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Cookie", cookie)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false, fmt.Errorf("登录态验证接口返回状态码 %d", resp.StatusCode)
	}

	var payload navPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", false, err
	}
	if payload.Code != 0 {
		return "", false, nil
	}

	username := strings.TrimSpace(payload.Data.Uname)
	return username, payload.Data.IsLogin && username != "", nil
}

func CookiesToHeader(cookies []*http.Cookie) string {
	parts := make([]string, 0, len(cookies))
	seen := map[string]struct{}{}
	for _, cookie := range cookies {
		name := strings.TrimSpace(cookie.Name)
		if name == "" || cookie.Value == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		parts = append(parts, name+"="+cookie.Value)
	}
	return strings.Join(parts, "; ")
}

func NormalizeCookieHeader(cookie string) string {
	cookie = strings.ReplaceAll(cookie, "\r", ";")
	cookie = strings.ReplaceAll(cookie, "\n", ";")
	fields := strings.Split(cookie, ";")
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			parts = append(parts, field)
		}
	}
	return strings.Join(parts, "; ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

type qrGeneratePayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		URL       string `json:"url"`
		QRCodeKey string `json:"qrcode_key"`
	} `json:"data"`
}

type qrPollPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"data"`
}

type navPayload struct {
	Code int `json:"code"`
	Data struct {
		IsLogin bool   `json:"isLogin"`
		Uname   string `json:"uname"`
	} `json:"data"`
}
