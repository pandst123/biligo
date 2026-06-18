//go:build windows

package winutil

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	shell32          = syscall.NewLazyDLL("shell32.dll")
	procShellExecute = shell32.NewProc("ShellExecuteW")
)

func Open(target string) error {
	operation, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	targetPtr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	result, _, callErr := procShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(operation)),
		uintptr(unsafe.Pointer(targetPtr)),
		0,
		0,
		1,
	)
	if result <= 32 {
		if callErr != syscall.Errno(0) {
			return callErr
		}
		return fmt.Errorf("ShellExecute failed with code %d", result)
	}
	return nil
}
