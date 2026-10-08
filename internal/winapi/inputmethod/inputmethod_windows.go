package inputmethod

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	u          = syscall.NewLazyDLL("user32.dll")
	load       = u.NewProc("LoadKeyboardLayoutW")
	activate   = u.NewProc("ActivateKeyboardLayout")
	getLayout  = u.NewProc("GetKeyboardLayout")
	foreground = u.NewProc("GetForegroundWindow")
	threadOf   = u.NewProc("GetWindowThreadProcessId")
	post       = u.NewProc("PostMessageW")
)

func SetEnglish(english bool) error {
	id := "00000804"
	if english {
		id = "00000409"
	}
	p, _ := syscall.UTF16PtrFromString(id)
	hkl, _, err := load.Call(uintptr(unsafe.Pointer(p)), 1)
	if hkl == 0 {
		return fmt.Errorf("加载输入法 %s 失败: %w", id, err)
	}
	if r, _, e := activate.Call(hkl, 0); r == 0 {
		return fmt.Errorf("切换输入法失败: %w", e)
	}
	if hwnd, _, _ := foreground.Call(); hwnd != 0 {
		if r, _, e := post.Call(hwnd, 0x50, 0, hkl); r == 0 {
			return fmt.Errorf("通知前台窗口切换输入法失败: %w", e)
		}
	}
	return nil
}

// Current returns "zh", "en", or "unknown" for the foreground window.
func Current() string {
	hwnd, _, _ := foreground.Call()
	thread, _, _ := threadOf.Call(hwnd, 0)
	hkl, _, _ := getLayout.Call(thread)
	switch hkl & 0xffff {
	case 0x0804:
		return "zh"
	case 0x0409:
		return "en"
	}
	return "unknown"
}
