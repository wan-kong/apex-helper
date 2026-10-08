package keyboard

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

const (
	whKeyboardLL = 13
	wmKeyDown    = 0x100
	wmKeyUp      = 0x101
	wmSysKeyDown = 0x104
	wmSysKeyUp   = 0x105
	wmQuit       = 0x12
	pmNoRemove   = 0
	vkEscape     = 0x1b
	vkLWin       = 0x5b
	vkRWin       = 0x5c
	vkControl    = 0x11
	vkMenu       = 0x12
	vkShift      = 0x10
)

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	setHook           = user32.NewProc("SetWindowsHookExW")
	unhook            = user32.NewProc("UnhookWindowsHookEx")
	nextHook          = user32.NewProc("CallNextHookEx")
	getKeyState       = user32.NewProc("GetAsyncKeyState")
	getMessage        = user32.NewProc("GetMessageW")
	peekMessage       = user32.NewProc("PeekMessageW")
	translateMessage  = user32.NewProc("TranslateMessage")
	dispatchMessage   = user32.NewProc("DispatchMessageW")
	postThreadMessage = user32.NewProc("PostThreadMessageW")
	getThreadID       = kernel32.NewProc("GetCurrentThreadId")
	getModuleHandle   = kernel32.NewProc("GetModuleHandleW")
)

type message struct {
	Hwnd     uintptr
	Message  uint32
	_        uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	PtX, PtY int32
	Private  uint32
}
type keyEvent struct {
	VKCode   uint32
	ScanCode uint32
	Flags    uint32
	Time     uint32
	Extra    uintptr
}

type Service struct {
	winDisabled      atomic.Bool
	altShiftDisabled atomic.Bool
	mu               sync.Mutex
	stop             chan struct{}
	threadID         uint32
	callback         uintptr
	recorder         func(string)
}

func New() *Service { return &Service{} }

func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stop != nil {
		return nil
	}
	ready := make(chan error, 1)
	stop := make(chan struct{})
	s.stop = stop
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		id, _, _ := getThreadID.Call()
		s.threadID = uint32(id)
		var msg message
		peekMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, pmNoRemove)
		cb := syscall.NewCallback(s.handle)
		s.callback = cb
		module, _, _ := getModuleHandle.Call(0)
		hook, _, callErr := setHook.Call(whKeyboardLL, cb, module, 0)
		if hook == 0 {
			ready <- fmt.Errorf("安装键盘 Hook 失败: %w", callErr)
			return
		}
		ready <- nil
		for {
			r, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(r) <= 0 {
				break
			}
			translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
		}
		unhook.Call(hook)
		close(stop)
	}()
	if err := <-ready; err != nil {
		s.stop = nil
		return err
	}
	return nil
}

func (s *Service) Stop() {
	s.mu.Lock()
	if s.stop == nil {
		s.mu.Unlock()
		return
	}
	stop, id := s.stop, s.threadID
	s.mu.Unlock()
	postThreadMessage.Call(uintptr(id), wmQuit, 0, 0)
	<-stop
	s.mu.Lock()
	s.stop = nil
	s.mu.Unlock()
}

func (s *Service) SetWinDisabled(v bool)      { s.winDisabled.Store(v) }
func (s *Service) SetAltShiftDisabled(v bool) { s.altShiftDisabled.Store(v) }
func (s *Service) WinDisabled() bool          { return s.winDisabled.Load() }
func (s *Service) AltShiftDisabled() bool     { return s.altShiftDisabled.Load() }
func (s *Service) Running() bool              { s.mu.Lock(); defer s.mu.Unlock(); return s.stop != nil }
func (s *Service) RecordNext(callback func(string)) {
	s.mu.Lock()
	s.recorder = callback
	s.mu.Unlock()
}

func down(vk uintptr) bool { r, _, _ := getKeyState.Call(vk); return uint16(r)&0x8000 != 0 }

func (s *Service) handle(code, wParam, lParam uintptr) uintptr {
	if int32(code) >= 0 && (wParam == wmKeyDown || wParam == wmSysKeyDown || wParam == wmKeyUp || wParam == wmSysKeyUp) {
		key := (*keyEvent)(unsafe.Pointer(lParam)).VKCode
		if wParam == wmKeyDown || wParam == wmSysKeyDown {
			if chord := shortcutName(key); chord != "" {
				s.mu.Lock()
				recorder := s.recorder
				s.recorder = nil
				s.mu.Unlock()
				if recorder != nil {
					go recorder(chord)
					return 1
				}
			}
		}
		if s.winDisabled.Load() && (key == vkLWin || key == vkRWin || key == vkEscape && down(vkControl)) {
			return 1
		}
		if s.altShiftDisabled.Load() && (isAlt(key) && down(vkShift) || isShift(key) && down(vkMenu)) {
			return 1
		}
	}
	r, _, _ := nextHook.Call(0, uintptr(code), wParam, lParam)
	return r
}

func isAlt(key uint32) bool   { return key == vkMenu || key == 0xa4 || key == 0xa5 }
func isShift(key uint32) bool { return key == vkShift || key == 0xa0 || key == 0xa1 }

func shortcutName(key uint32) string {
	name := ""
	switch {
	case key >= 'A' && key <= 'Z', key >= '0' && key <= '9':
		name = string(rune(key))
	case key >= 0x70 && key <= 0x7b:
		name = fmt.Sprintf("F%d", key-0x6f)
	}
	if name == "" {
		return ""
	}
	mods := ""
	if down(vkControl) {
		mods += "Ctrl+"
	}
	if down(vkMenu) {
		mods += "Alt+"
	}
	if down(vkShift) {
		mods += "Shift+"
	}
	if mods == "" {
		return ""
	}
	return mods + name
}
