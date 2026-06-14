package store

import (
	"context"
	"testing"

	"github.com/fdcs99/biligo/internal/model"
)

func TestCreateTaskPersistsFullPurchaseConfig(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	task, err := store.CreateTask(context.Background(), model.TaskInput{
		Name:        "测试任务",
		AccountID:   1,
		ProjectID:   1001701,
		ProjectName: "测试项目",
		ScreenID:    2001,
		SKUID:       3001,
		SessionName: "晚场",
		TicketLevel: "VIP",
		TicketPrice: 68000,
		SaleStart:   "2026-06-13 20:00:00",
		OrderType:   1,
		BuyerInfo: []model.TicketBuyer{
			{Name: "张三", PersonalID: "110101199001010000"},
			{Name: "李四", PersonalID: "110101199001010001"},
		},
		Buyer: "张三",
		Tel:   "13800000000",
		DeliverInfo: &model.TicketAddress{
			ID:          9,
			Name:        "张三",
			Phone:       "13800000000",
			FullAddress: "上海市测试路 1 号",
		},
		PollIntervalSeconds: 2,
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if task.Quantity != 2 {
		t.Fatalf("Quantity = %d, want 2", task.Quantity)
	}
	if task.PayMoney != 136000 {
		t.Fatalf("PayMoney = %d, want 136000", task.PayMoney)
	}
	if len(task.BuyerInfo) != 2 {
		t.Fatalf("BuyerInfo len = %d, want 2", len(task.BuyerInfo))
	}
	if task.DeliverInfo == nil || task.DeliverInfo.ID != 9 {
		t.Fatalf("DeliverInfo = %#v, want address id 9", task.DeliverInfo)
	}
	if task.TimeSyncStrategy != model.TimeSyncStrategyBilibili {
		t.Fatalf("TimeSyncStrategy = %q, want %q", task.TimeSyncStrategy, model.TimeSyncStrategyBilibili)
	}

	task, log, err := store.SetTaskTimeSync(context.Background(), task.ID, model.TimeSyncStrategyBilibili, 88, "2026-06-14T10:00:00+08:00", "时间同步完成")
	if err != nil {
		t.Fatalf("SetTaskTimeSync: %v", err)
	}
	if log.ID == 0 {
		t.Fatal("time sync log was not created")
	}
	if task.TimeOffsetMillis != 88 || task.TimeSyncedAt == "" {
		t.Fatalf("unexpected time sync fields: %#v", task)
	}

	task, log, err = store.SetTaskPayMoney(context.Background(), task.ID, 140000, "金额已更新")
	if err != nil {
		t.Fatalf("SetTaskPayMoney: %v", err)
	}
	if log.ID == 0 {
		t.Fatal("pay money log was not created")
	}
	if task.PayMoney != 140000 {
		t.Fatalf("PayMoney = %d, want 140000", task.PayMoney)
	}
}
