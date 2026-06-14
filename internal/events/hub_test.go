package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHubPublishesToSubscriber(t *testing.T) {
	hub := NewHub()
	id, ch := hub.Subscribe()
	defer hub.Unsubscribe(id)

	hub.Publish("task.updated", map[string]any{"id": 1})

	select {
	case event := <-ch:
		if event.Name != "task.updated" {
			t.Fatalf("event.Name = %q, want task.updated", event.Name)
		}
		var payload map[string]any
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if payload["id"].(float64) != 1 {
			t.Fatalf("payload id = %v, want 1", payload["id"])
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}
