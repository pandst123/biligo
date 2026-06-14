package timesync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAggregateDropsMinAndMaxOffset(t *testing.T) {
	result, err := Aggregate([]Sample{
		{OffsetMillis: 100, RTTMillis: 10},
		{OffsetMillis: 2, RTTMillis: 20},
		{OffsetMillis: 4, RTTMillis: 30},
		{OffsetMillis: 1, RTTMillis: 40},
		{OffsetMillis: 3, RTTMillis: 50},
	})
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	if result.OffsetMillis != 3 {
		t.Fatalf("OffsetMillis = %d, want 3", result.OffsetMillis)
	}
	if result.AverageRTTMillis != 33 {
		t.Fatalf("AverageRTTMillis = %d, want 33", result.AverageRTTMillis)
	}
	if result.AveragedSampleCount != 3 || result.TotalSampleCount != 5 {
		t.Fatalf("sample counts = %d/%d, want 3/5", result.AveragedSampleCount, result.TotalSampleCount)
	}
}

func TestClientSyncRequestsFiveSamples(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    0,
			"message": "0",
			"ttl":     1,
			"data": map[string]any{
				"timestamp": time.Now().Unix(),
				"microtime": time.Now().UnixMilli(),
			},
		})
	}))
	defer server.Close()

	client := NewClientWithEndpoint(server.Client(), server.URL)
	result, err := client.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if calls != 5 {
		t.Fatalf("calls = %d, want 5", calls)
	}
	if result.AveragedSampleCount != 3 {
		t.Fatalf("AveragedSampleCount = %d, want 3", result.AveragedSampleCount)
	}
}
