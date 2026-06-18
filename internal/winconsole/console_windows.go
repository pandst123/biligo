//go:build windows

package winconsole

import (
	"io"
	"os"
	"sync"
	"syscall"
)

const (
	swHide = 0
	swShow = 5
)

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	user32            = syscall.NewLazyDLL("user32.dll")
	procAllocConsole  = kernel32.NewProc("AllocConsole")
	procFreeConsole   = kernel32.NewProc("FreeConsole")
	procGetConsoleWnd = kernel32.NewProc("GetConsoleWindow")
	procSetConsoleCP  = kernel32.NewProc("SetConsoleCP")
	procSetOutputCP   = kernel32.NewProc("SetConsoleOutputCP")
	procShowWindow    = user32.NewProc("ShowWindow")
)

type Controller struct {
	mu        sync.Mutex
	file      *os.File
	allocated bool
	visible   bool
}

func New() *Controller {
	return &Controller{visible: getConsoleWindow() != 0}
}

func (c *Controller) Show() (io.Writer, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if getConsoleWindow() == 0 {
		if result, _, err := procAllocConsole.Call(); result == 0 {
			return nil, err
		}
		c.allocated = true
	}
	setConsoleCodePageUTF8()
	showConsole(swShow)
	if c.file == nil {
		file, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
		if err != nil {
			return nil, err
		}
		c.file = file
		os.Stdout = file
		os.Stderr = file
	}
	c.visible = true
	return c.file, nil
}

func setConsoleCodePageUTF8() {
	_, _, _ = procSetConsoleCP.Call(65001)
	_, _, _ = procSetOutputCP.Call(65001)
}

func (c *Controller) Hide() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.file != nil {
		_ = c.file.Close()
		c.file = nil
	}
	if c.allocated {
		_, _, _ = procFreeConsole.Call()
		c.allocated = false
	} else {
		showConsole(swHide)
	}
	c.visible = false
}

func (c *Controller) Visible() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.visible
}

func showConsole(cmdShow uintptr) {
	if hwnd := getConsoleWindow(); hwnd != 0 {
		_, _, _ = procShowWindow.Call(hwnd, cmdShow)
	}
}

func getConsoleWindow() uintptr {
	hwnd, _, _ := procGetConsoleWnd.Call()
	return hwnd
}
