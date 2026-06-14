package timesync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"time"
)

const (
	DefaultEndpoint = "https://api.live.bilibili.com/xlive/open-interface/v1/rtc/getTimestamp"
	defaultSamples  = 5
)

type Client struct {
	httpClient *http.Client
	endpoint   string
}

type Result struct {
	OffsetMillis          int64
	AverageRTTMillis      int64
	TotalSampleCount      int
	AveragedSampleCount   int
	SyncedAt              time.Time
	Samples               []Sample
	AveragedSampleOffsets []int64
}

type Sample struct {
	ServerUnixMillis int64
	RTTMillis        int64
	OffsetMillis     int64
}

type timestampResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Timestamp int64 `json:"timestamp"`
		Microtime int64 `json:"microtime"`
	} `json:"data"`
}

func NewClient(httpClient *http.Client) *Client {
	return NewClientWithEndpoint(httpClient, DefaultEndpoint)
}

func NewClientWithEndpoint(httpClient *http.Client, endpoint string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	return &Client{
		httpClient: httpClient,
		endpoint:   endpoint,
	}
}

func (c *Client) Sync(ctx context.Context) (Result, error) {
	samples := make([]Sample, 0, defaultSamples)
	for i := 0; i < defaultSamples; i++ {
		sample, err := c.sample(ctx)
		if err != nil {
			return Result{}, err
		}
		samples = append(samples, sample)
	}
	return Aggregate(samples)
}

func (c *Client) sample(ctx context.Context) (Sample, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return Sample{}, err
	}

	startedAt := time.Now()
	resp, err := c.httpClient.Do(req)
	endedAt := time.Now()
	if err != nil {
		return Sample{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Sample{}, fmt.Errorf("时间同步接口返回 HTTP %d", resp.StatusCode)
	}

	var payload timestampResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Sample{}, err
	}
	if payload.Code != 0 {
		return Sample{}, fmt.Errorf("时间同步接口返回 code=%d message=%s", payload.Code, payload.Message)
	}

	serverMillis := payload.Data.Microtime
	if serverMillis <= 0 && payload.Data.Timestamp > 0 {
		serverMillis = payload.Data.Timestamp * 1000
	} // fall back到timestamp上
	if serverMillis <= 0 {
		return Sample{}, errors.New("时间同步接口未返回有效时间")
	}

	rtt := endedAt.Sub(startedAt)
	localMidpoint := startedAt.Add(rtt / 2) // t0 + (t1 - t0) / 2 估算服务器时间
	return Sample{
		ServerUnixMillis: serverMillis,
		RTTMillis:        int64(math.Round(float64(rtt) / float64(time.Millisecond))),
		OffsetMillis:     serverMillis - localMidpoint.UnixMilli(),
	}, nil
}

func Aggregate(samples []Sample) (Result, error) {
	if len(samples) != defaultSamples {
		return Result{}, fmt.Errorf("时间同步需要 %d 次采样，实际 %d 次", defaultSamples, len(samples))
	}

	ordered := append([]Sample(nil), samples...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].OffsetMillis == ordered[j].OffsetMillis {
			return ordered[i].RTTMillis < ordered[j].RTTMillis
		}
		return ordered[i].OffsetMillis < ordered[j].OffsetMillis
	})

	trimmed := ordered[1 : len(ordered)-1]
	var offsetTotal int64
	var rttTotal int64
	offsets := make([]int64, 0, len(trimmed))
	for _, sample := range trimmed {
		offsetTotal += sample.OffsetMillis
		rttTotal += sample.RTTMillis
		offsets = append(offsets, sample.OffsetMillis)
	}

	return Result{
		OffsetMillis:          roundDiv(offsetTotal, int64(len(trimmed))),
		AverageRTTMillis:      roundDiv(rttTotal, int64(len(trimmed))),
		TotalSampleCount:      len(samples),
		AveragedSampleCount:   len(trimmed),
		SyncedAt:              time.Now(),
		Samples:               samples,
		AveragedSampleOffsets: offsets,
	}, nil
}

func roundDiv(total int64, count int64) int64 {
	if count == 0 {
		return 0
	}
	if total >= 0 {
		return (total + count/2) / count
	}
	return (total - count/2) / count
}
