package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fdcs99/biligo/internal/model"
	"github.com/fdcs99/biligo/internal/store"
)

func TestDeleteTaskWritesLog(t *testing.T) {
	taskStore, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer taskStore.Close()

	task, err := taskStore.CreateTask(context.Background(), model.TaskInput{
		Name:               "待删除任务",
		AccountID:          1,
		ProjectID:          1001701,
		ScreenID:           2001,
		SKUID:              3001,
		TicketDisplay:      "晚场 - VIP",
		SaleStart:          "2026-06-13 20:00:00",
		BuyerInfo:          []model.TicketBuyer{{Name: "张三", PersonalID: "110101199001010000"}},
		Buyer:              "张三",
		Tel:                "13800000000",
		DeliverInfo:        &model.TicketAddress{ID: 9, Name: "张三", Phone: "13800000000"},
		PollIntervalMillis: 1000,
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	router := NewRouter(taskStore)
	req := httptest.NewRequest(http.MethodDelete, "/api/tasks/1", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}

	if _, err := taskStore.GetTask(context.Background(), task.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetTask err = %v, want sql.ErrNoRows", err)
	}
	logs, err := taskStore.ListTaskLogs(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListTaskLogs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("delete log was not created")
	}
	if logs[0].TaskID != task.ID || logs[0].Level != "warn" || logs[0].Message != "任务已删除：待删除任务。" {
		t.Fatalf("unexpected delete log: %#v", logs[0])
	}
}
