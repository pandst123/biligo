//go:build windows

package winutil

import (
	"syscall"
	"unsafe"
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procOpenClipboard  = user32.NewProc("OpenClipboard")
	procCloseClipboard = user32.NewProc("CloseClipboard")
	procEmptyClipboard = user32.NewProc("EmptyClipboard")
	procSetClipboard   = user32.NewProc("SetClipboardData")
	procGlobalAlloc    = kernel32.NewProc("GlobalAlloc")
	procGlobalLock     = kernel32.NewProc("GlobalLock")
	procGlobalUnlock   = kernel32.NewProc("GlobalUnlock")
	procGlobalFree     = kernel32.NewProc("GlobalFree")
)

func SetClipboardText(text string) error {
	data, err := syscall.UTF16FromString(text)
	if err != nil {
		return err
	}
	size := uintptr(len(data) * 2)
	handle, _, err := procGlobalAlloc.Call(gmemMoveable, size)
	if handle == 0 {
		return err
	}
	locked, _, err := procGlobalLock.Call(handle)
	if locked == 0 {
		_, _, _ = procGlobalFree.Call(handle)
		return err
	}
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(locked)), len(data)), data)
	_, _, _ = procGlobalUnlock.Call(handle)

	if ok, _, err := procOpenClipboard.Call(0); ok == 0 {
		_, _, _ = procGlobalFree.Call(handle)
		return err
	}
	defer procCloseClipboard.Call()

	if ok, _, err := procEmptyClipboard.Call(); ok == 0 {
		_, _, _ = procGlobalFree.Call(handle)
		return err
	}
	if ok, _, err := procSetClipboard.Call(cfUnicodeText, handle); ok == 0 {
		_, _, _ = procGlobalFree.Call(handle)
		return err
	}
	return nil
}
