package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fdcs99/biligo/internal/model"
	xproxy "golang.org/x/net/proxy"
)

const (
	latencyTestURL    = "https://show.bilibili.com/"
	ipLocationTestURL = "http://myip.ipip.net"
)

type TestResult struct {
	LatencyMillis int64
	IPLocation    string
	IPLocationErr string
}

func NewHTTPClient(node model.ProxyNode) (*http.Client, error) {
	node.Protocol = model.NormalizeProxyProtocol(node.Protocol)
	if strings.TrimSpace(node.Host) == "" || node.Port <= 0 {
		return nil, errors.New("代理节点地址不完整")
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = false
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 20
	transport.IdleConnTimeout = 90 * time.Second

	switch node.Protocol {
	case model.ProxyProtocolSOCKS5:
		auth := &xproxy.Auth{}
		if node.Username != "" || node.Password != "" {
			auth.User = node.Username
			auth.Password = node.Password
		} else {
			auth = nil
		}
		dialer, err := xproxy.SOCKS5("tcp", net.JoinHostPort(node.Host, fmt.Sprint(node.Port)), auth, xproxy.Direct)
		if err != nil {
			return nil, err
		}
		transport.Proxy = nil
		transport.DialContext = func(ctx context.Context, network string, address string) (net.Conn, error) {
			type dialResult struct {
				conn net.Conn
				err  error
			}
			ch := make(chan dialResult, 1)
			go func() {
				conn, err := dialer.Dial(network, address)
				ch <- dialResult{conn: conn, err: err}
			}()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case result := <-ch:
				return result.conn, result.err
			}
		}
	default:
		proxyURL := &url.URL{
			Scheme: node.Protocol,
			Host:   net.JoinHostPort(node.Host, fmt.Sprint(node.Port)),
		}
		if node.Username != "" || node.Password != "" {
			proxyURL.User = url.UserPassword(node.Username, node.Password)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: transport,
	}, nil
}

func TestNode(ctx context.Context, node model.ProxyNode) (TestResult, error) {
	var result TestResult
	client, err := NewHTTPClient(node)
	if err != nil {
		return result, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, latencyTestURL, nil)
	if err != nil {
		return result, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 Biligo Proxy Check")
	startedAt := time.Now()
	resp, err := client.Do(req)
	result.LatencyMillis = elapsedMillis(startedAt)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return result, fmt.Errorf("检测请求返回状态码 %d", resp.StatusCode)
	}

	locationCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	location, locationErr := fetchIPLocation(locationCtx, client)
	if locationErr != nil {
		result.IPLocationErr = locationErr.Error()
	} else {
		result.IPLocation = location
	}
	return result, nil
}

func fetchIPLocation(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ipLocationTestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 Biligo Proxy Check")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", fmt.Errorf("IP 归属地请求返回状态码 %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}
	location := strings.TrimSpace(string(body))
	if location == "" {
		return "", errors.New("IP 归属地响应为空")
	}
	return location, nil
}

func elapsedMillis(startedAt time.Time) int64 {
	elapsed := time.Since(startedAt).Milliseconds()
	if elapsed <= 0 {
		return 1
	}
	return elapsed
}

func IsRequestError(err error) bool {
	if err == nil {
		return false
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return strings.Contains(err.Error(), "proxyconnect") ||
		strings.Contains(err.Error(), "socks") ||
		strings.Contains(err.Error(), "connection refused")
}
