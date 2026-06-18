//go:build windows

package wintray

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fdcs99/biligo/assets"
	"github.com/fdcs99/biligo/internal/app"
	"github.com/fdcs99/biligo/internal/config"
	"github.com/fdcs99/biligo/internal/winconsole"
	"github.com/fdcs99/biligo/internal/winutil"
	"github.com/gogpu/systray"
)

type Tray struct {
	app     *app.App
	console *winconsole.Controller
	tray    *systray.SystemTray
	mu      sync.Mutex
}

func Run(configPath string) error {
	console := winconsole.New()
	console.Hide()

	service := app.New(configPath)
	service.SetConsoleWriter(io.Discard)

	tray := &Tray{
		app:     service,
		console: console,
		tray:    systray.New(),
	}
	tray.tray.SetIcon(assets.LogoPNG).
		SetTooltip("Biligo").
		OnClick(tray.openPanel).
		OnDoubleClick(tray.openPanel)
	tray.refreshMenu()
	tray.tray.Show()
	tray.startService(true)
	return tray.tray.Run()
}

func (t *Tray) refreshMenu() {
	t.mu.Lock()
	defer t.mu.Unlock()

	menu := systray.NewMenu()
	menu.Add(statusLabel(t.app.Status()), nil)
	if err := t.app.LastError(); err != nil && t.app.Status() == app.StatusFailed {
		menu.Add("错误："+trimMenuText(err.Error()), nil)
	}
	menu.AddSeparator()
	menu.Add("打开网页", t.openPanel)
	menu.Add("启动服务", func() { t.startService(false) })
	menu.Add("停止服务", func() { t.stopService(false) })
	menu.Add("重启服务", t.restartService)
	menu.Add("复制面板密码", t.copyPanelPassword)
	menu.AddSeparator()
	if t.console.Visible() {
		menu.Add("隐藏控制台", t.hideConsole)
	} else {
		menu.Add("显示控制台", t.showConsole)
	}
	menu.Add("打开配置目录", t.openConfigDir)
	menu.Add("打开日志目录", t.openLogDir)
	menu.AddSeparator()
	menu.Add("退出 Biligo", t.quit)
	t.tray.SetMenu(menu)
	t.tray.SetTooltip("Biligo - " + statusLabel(t.app.Status()))
}

func (t *Tray) startService(auto bool) {
	go func() {
		if t.app.Status() == app.StatusRunning || t.app.Status() == app.StatusStarting {
			if !auto {
				t.tray.ShowNotification("Biligo", "服务已经在运行。")
			}
			t.refreshMenu()
			return
		}
		if err := t.app.Start(context.Background()); err != nil {
			t.tray.ShowNotification("Biligo 启动失败", err.Error())
			t.refreshMenu()
			return
		}
		message := "服务已启动。"
		if t.app.PanelURL() != "" {
			message = "服务已启动：" + t.app.PanelURL()
		}
		t.tray.ShowNotification("Biligo", message)
		if password := t.app.GeneratedPanelPassword(); password != "" {
			t.tray.ShowNotification("Biligo 面板登录密码", password)
		}
		t.refreshMenu()
	}()
}

func (t *Tray) stopService(quiet bool) {
	go func() {
		if t.app.Status() == app.StatusStopped || t.app.Status() == app.StatusStopping {
			if !quiet {
				t.tray.ShowNotification("Biligo", "服务已经停止。")
			}
			t.refreshMenu()
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := t.app.Stop(ctx); err != nil {
			t.tray.ShowNotification("Biligo 停止失败", err.Error())
			t.refreshMenu()
			return
		}
		if !quiet {
			t.tray.ShowNotification("Biligo", "服务已停止。")
		}
		t.refreshMenu()
	}()
}

func (t *Tray) restartService() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := t.app.Stop(ctx); err != nil {
			t.tray.ShowNotification("Biligo 重启失败", err.Error())
			t.refreshMenu()
			return
		}
		if err := t.app.Start(context.Background()); err != nil {
			t.tray.ShowNotification("Biligo 重启失败", err.Error())
			t.refreshMenu()
			return
		}
		t.tray.ShowNotification("Biligo", "服务已重启："+t.app.PanelURL())
		t.refreshMenu()
	}()
}

func (t *Tray) openPanel() {
	url := t.app.PanelURL()
	if strings.TrimSpace(url) == "" || t.app.Status() != app.StatusRunning {
		t.tray.ShowNotification("Biligo", "服务未启动，请先启动服务。")
		return
	}
	if err := winutil.Open(url); err != nil {
		t.tray.ShowNotification("Biligo", "打开网页失败："+err.Error())
	}
}

func (t *Tray) showConsole() {
	writer, err := t.console.Show()
	if err != nil {
		t.tray.ShowNotification("Biligo", "显示控制台失败："+err.Error())
		return
	}
	t.app.SetConsoleWriter(writer)
	if logger := t.app.Logger(); logger != nil {
		logger.Infof("控制台已显示，中文输出测试。")
	}
	t.refreshMenu()
}

func (t *Tray) hideConsole() {
	t.app.SetConsoleWriter(io.Discard)
	t.console.Hide()
	t.refreshMenu()
}

func (t *Tray) openConfigDir() {
	if err := openDir(filepath.Dir(absPath(t.app.ConfigPath()))); err != nil {
		t.tray.ShowNotification("Biligo", "打开配置目录失败："+err.Error())
	}
}

func (t *Tray) openLogDir() {
	if err := openDir(filepath.Dir(absPath(t.app.LogPath()))); err != nil {
		t.tray.ShowNotification("Biligo", "打开日志目录失败："+err.Error())
	}
}

func (t *Tray) copyPanelPassword() {
	password := strings.TrimSpace(t.app.PanelPassword())
	if password == "" {
		if storedPassword, err := config.ReadPanelPassword(t.app.ConfigPath()); err == nil {
			password = strings.TrimSpace(storedPassword)
		}
	}
	if password == "" {
		t.tray.ShowNotification("Biligo", "未找到面板密码，请先启动服务。")
		return
	}
	if err := winutil.SetClipboardText(password); err != nil {
		t.tray.ShowNotification("Biligo", "复制面板密码失败："+err.Error())
		return
	}
	t.tray.ShowNotification("Biligo", "面板密码已复制。")
}

func (t *Tray) quit() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = t.app.Stop(ctx)
		t.app.SetConsoleWriter(io.Discard)
		t.console.Hide()
		t.tray.Remove()
		os.Exit(0)
	}()
}

func statusLabel(status app.Status) string {
	switch status {
	case app.StatusStarting:
		return "🟡 状态：启动中"
	case app.StatusRunning:
		return "🟢 状态：运行中"
	case app.StatusStopping:
		return "🟡 状态：停止中"
	case app.StatusFailed:
		return "🔴 状态：启动失败"
	default:
		return "⚪ 状态：已停止"
	}
}

func trimMenuText(text string) string {
	text = strings.TrimSpace(text)
	const maxLen = 60
	if len([]rune(text)) <= maxLen {
		return text
	}
	runes := []rune(text)
	return string(runes[:maxLen]) + "..."
}

func absPath(path string) string {
	if strings.TrimSpace(path) == "" {
		path = "."
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func openDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return winutil.Open(dir)
}
