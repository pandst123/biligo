package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fdcs99/biligo/internal/biliticket"
	"github.com/fdcs99/biligo/internal/events"
	"github.com/fdcs99/biligo/internal/model"
	"github.com/fdcs99/biligo/internal/store"
)

type Manager struct {
	store  *store.Store
	ticket *biliticket.Client
	hub    *events.Hub

	mu      sync.Mutex
	running map[int64]context.CancelFunc
}

const (
	saleStartWaitTick           = time.Microsecond * 50
	saleStartWaitReportInterval = time.Second
)

func NewManager(store *store.Store, ticket *biliticket.Client, hub *events.Hub) *Manager {
	return &Manager{
		store:   store,
		ticket:  ticket,
		hub:     hub,
		running: map[int64]context.CancelFunc{},
	}
}

func (m *Manager) Dispatch(ctx context.Context, taskID int64) (model.Task, error) {
	task, err := m.store.GetTask(ctx, taskID)
	if err != nil {
		return model.Task{}, err
	}
	if err := validateTask(task); err != nil {
		task, log, setErr := m.store.SetTaskRuntime(ctx, taskID, model.TaskRuntimeUpdate{
			Status:      "failed",
			LastMessage: err.Error(),
		}, "error")
		if setErr == nil {
			m.publishTaskAndLog(task, log)
		}
		return model.Task{}, err
	}
	_, cookie, err := m.store.GetAccountCookie(ctx, task.AccountID)
	if err != nil {
		return model.Task{}, err
	}
	if strings.TrimSpace(cookie) == "" {
		return model.Task{}, errors.New("账号未保存 Cookie")
	}

	m.mu.Lock()
	if _, ok := m.running[taskID]; ok {
		m.mu.Unlock()
		return task, nil
	}
	runCtx, cancel := context.WithCancel(context.Background())
	m.running[taskID] = cancel
	m.mu.Unlock()

	task, log, err := m.store.SetTaskRuntime(ctx, taskID, model.TaskRuntimeUpdate{
		Status:      "waiting_start",
		LastMessage: "任务已下发，等待票档起售时间。",
	}, "info")
	if err != nil {
		m.remove(taskID)
		cancel()
		return model.Task{}, err
	}
	m.publishTaskAndLog(task, log)

	go m.run(runCtx, taskID, cookie)
	return task, nil
}

func (m *Manager) Pause(ctx context.Context, taskID int64) (model.Task, error) {
	m.mu.Lock()
	cancel := m.running[taskID]
	if cancel != nil {
		cancel()
		delete(m.running, taskID)
	}
	m.mu.Unlock()

	task, log, err := m.store.SetTaskRuntime(ctx, taskID, model.TaskRuntimeUpdate{
		Status:      "paused",
		LastMessage: "任务已暂停。",
	}, "warn")
	if err != nil {
		return model.Task{}, err
	}
	m.publishTaskAndLog(task, log)
	return task, nil
}

func (m *Manager) Cancel(taskID int64) {
	m.mu.Lock()
	cancel := m.running[taskID]
	if cancel != nil {
		cancel()
		delete(m.running, taskID)
	}
	m.mu.Unlock()
}

func (m *Manager) run(ctx context.Context, taskID int64, cookie string) {
	defer m.remove(taskID)

	task, err := m.store.GetTask(context.Background(), taskID)
	if err != nil {
		return
	}
	saleStart, err := parseTaskTime(task.SaleStart)
	if err != nil {
		m.setRuntime(taskID, "failed", "无法解析票档起售时间："+err.Error(), "error")
		return
	}
	endAt := saleStart.Add(10 * time.Minute)
	if parsedEnd, err := parseTaskTime(task.EndAt); err == nil && !parsedEnd.IsZero() {
		endAt = parsedEnd
	}

	if !m.waitUntilSaleStart(ctx, taskID, saleStart) {
		return
	}

	m.setRuntime(taskID, "running", "已到起售时间，开始检测票档状态。", "info")
	interval := time.Duration(task.PollIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 3 * time.Second
	}

	for {
		if ctx.Err() != nil {
			return
		}
		if time.Now().After(endAt) {
			m.setRuntime(taskID, "failed", "已超过任务结束时间，停止检测。", "warn")
			return
		}

		latestTask, err := m.store.GetTask(context.Background(), taskID)
		if err != nil {
			return
		}
		option, available, err := m.ticket.CheckTicketStatus(ctx, latestTask, cookie)
		checkedAt := time.Now().Format(time.RFC3339)
		if err != nil {
			status := "running"
			level := "warn"
			if requiresUserAction(err.Error()) {
				status = "waiting_user"
				level = "error"
			}
			m.setRuntimeWithCheckedAt(taskID, status, "检测票档状态失败："+err.Error(), level, checkedAt)
			if status == "waiting_user" {
				return
			}
			if !m.wait(ctx, interval) {
				return
			}
			continue
		}
		if !available {
			m.setRuntimeWithCheckedAt(taskID, "running", fmt.Sprintf("票档状态：%s，继续等待。", firstNonEmpty(option.SaleStatus, "未知状态")), "info", checkedAt)
			if !m.wait(ctx, interval) {
				return
			}
			continue
		}

		m.setRuntimeWithCheckedAt(taskID, "running", "票档可购买，开始准备订单。", "info", checkedAt)
		prepared, err := m.ticket.PrepareOrder(ctx, latestTask, cookie)
		if err != nil {
			m.handleRunError(taskID, "订单准备失败："+err.Error())
			return
		}
		result, err := m.ticket.CreateOrder(ctx, latestTask, cookie, prepared)
		if err != nil {
			if result.Code == 100079 {
				m.setRuntime(taskID, "duplicate_order", "存在重复订单，已停止。", "warn")
				return
			}
			m.handleRunError(taskID, "创建订单失败："+err.Error())
			return
		}
		if result.Code == 100079 {
			m.setRuntime(taskID, "duplicate_order", "存在重复订单，已停止。", "warn")
			return
		}
		if result.OrderID == "" {
			m.setRuntime(taskID, "succeeded", "订单接口返回成功，但未返回订单 ID。", "warn")
			return
		}
		payParam, err := m.ticket.GetPayParam(ctx, result.OrderID, cookie)
		if err != nil {
			m.setRuntime(taskID, "succeeded", "订单创建成功，但获取支付二维码失败："+err.Error(), "warn")
			return
		}
		task, log, err := m.store.SetTaskRuntime(context.Background(), taskID, model.TaskRuntimeUpdate{
			Status:                "waiting_payment",
			LastMessage:           "订单创建成功，请尽快完成支付。",
			OrderID:               result.OrderID,
			PaymentURL:            payParam.CodeURL,
			PaymentQRImageDataURL: payParam.QRImageDataURL,
			LastCheckedAt:         time.Now().Format(time.RFC3339),
		}, "info")
		if err == nil {
			m.publishTaskAndLog(task, log)
		}
		return
	}
}

func (m *Manager) handleRunError(taskID int64, message string) {
	status := "failed"
	if requiresUserAction(message) {
		status = "waiting_user"
	}
	m.setRuntime(taskID, status, message, "error")
}

func (m *Manager) setRuntime(taskID int64, status string, message string, level string) {
	m.setRuntimeWithCheckedAt(taskID, status, message, level, "")
}

func (m *Manager) setRuntimeWithCheckedAt(taskID int64, status string, message string, level string, checkedAt string) {
	task, log, err := m.store.SetTaskRuntime(context.Background(), taskID, model.TaskRuntimeUpdate{
		Status:        status,
		LastMessage:   message,
		LastCheckedAt: checkedAt,
	}, level)
	if err == nil {
		m.publishTaskAndLog(task, log)
	}
}

func (m *Manager) publishTaskAndLog(task model.Task, log model.TaskLog) {
	if m.hub == nil {
		return
	}
	m.hub.Publish("task.updated", task)
	if log.ID > 0 {
		m.hub.Publish("log.created", log)
	}
}

func (m *Manager) wait(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (m *Manager) waitUntilSaleStart(ctx context.Context, taskID int64, saleStart time.Time) bool {
	nextReportAt := time.Now().Add(saleStartWaitReportInterval)
	for {
		remaining := time.Until(saleStart)
		if remaining <= 0 {
			return true
		}
		if ctx.Err() != nil {
			return false
		}
		now := time.Now()
		if !now.Before(nextReportAt) {
			m.setRuntime(taskID, "waiting_start", "等待起售中，距离起售还有 "+formatRemaining(remaining)+"。", "info")
			nextReportAt = now.Add(saleStartWaitReportInterval)
		}
		waitFor := remaining
		if waitFor > saleStartWaitTick {
			waitFor = saleStartWaitTick
		}
		if !m.wait(ctx, waitFor) {
			return false
		}
	}
}

func (m *Manager) remove(taskID int64) {
	m.mu.Lock()
	delete(m.running, taskID)
	m.mu.Unlock()
}

func formatRemaining(duration time.Duration) string {
	if duration < 0 {
		duration = 0
	}
	totalSeconds := int(duration.Round(time.Second).Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	if hours > 0 {
		return fmt.Sprintf("%d小时%d分%d秒", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%d分%d秒", minutes, seconds)
	}
	return fmt.Sprintf("%d秒", seconds)
}

func validateTask(task model.Task) error {
	if task.AccountID <= 0 {
		return errors.New("请先选择账号")
	}
	if task.ProjectID <= 0 || task.ScreenID <= 0 || task.SKUID <= 0 {
		return errors.New("请先获取票务信息并选择票信息")
	}
	if strings.TrimSpace(task.SaleStart) == "" {
		return errors.New("票档缺少起售时间")
	}
	if len(task.BuyerInfo) == 0 {
		return errors.New("请至少选择一位实名购票人")
	}
	if task.DeliverInfo == nil || task.DeliverInfo.ID <= 0 {
		return errors.New("请先选择收货地址")
	}
	if strings.TrimSpace(task.Buyer) == "" {
		return errors.New("请填写联系人姓名")
	}
	if strings.TrimSpace(task.Tel) == "" {
		return errors.New("请填写联系人电话")
	}
	return nil
}

func parseTaskTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("时间为空")
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04",
	}
	var lastErr error
	for _, format := range formats {
		parsed, err := time.ParseInLocation(format, value, time.Local)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

func requiresUserAction(message string) bool {
	message = strings.ToLower(message)
	keywords := []string{"验证码", "风控", "人机", "实名", "请先登录", "登录", "risk", "captcha", "verify", "defaultbbr"}
	for _, keyword := range keywords {
		if strings.Contains(message, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
