package audio

import (
	"fmt"
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

type Device struct{ ID, Name string }

var ole = syscall.NewLazyDLL("ole32.dll")
var coInit = ole.NewProc("CoInitializeEx")
var coUninit = ole.NewProc("CoUninitialize")
var coCreate = ole.NewProc("CoCreateInstance")
var coFree = ole.NewProc("CoTaskMemFree")
var variantClear = ole.NewProc("PropVariantClear")

var clsEnumerator = syscall.GUID{Data1: 0xbcde0395, Data2: 0xe52f, Data3: 0x467c, Data4: [8]byte{0x8e, 0x3d, 0xc4, 0x57, 0x92, 0x91, 0x69, 0x2e}}
var iidEnumerator = syscall.GUID{Data1: 0xa95664d2, Data2: 0x9614, Data3: 0x4f35, Data4: [8]byte{0xa7, 0x46, 0xde, 0x8d, 0xb6, 0x36, 0x17, 0xe6}}
var clsPolicy = syscall.GUID{Data1: 0x870af99c, Data2: 0x171d, Data3: 0x4f9e, Data4: [8]byte{0xaf, 0x0d, 0xe6, 0x3d, 0xf4, 0x0c, 0x2b, 0xc9}}
var iidPolicy = syscall.GUID{Data1: 0xf8679f50, Data2: 0x850a, Data3: 0x41cf, Data4: [8]byte{0x9c, 0x72, 0x43, 0x0f, 0x29, 0x02, 0x90, 0xc8}}
var friendlyName = propertyKey{Fmtid: syscall.GUID{Data1: 0xa45c254e, Data2: 0xdf1c, Data3: 0x4efd, Data4: [8]byte{0x80, 0x20, 0x67, 0xd1, 0x46, 0xa8, 0x50, 0xe0}}, Pid: 14}

type propertyKey struct {
	Fmtid syscall.GUID
	Pid   uint32
}
type propVariant struct {
	VT      uint16
	_       [6]byte
	Pointer uintptr
	Extra   uintptr
}

func method(p uintptr, index uintptr) uintptr {
	return *(*uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(p)) + index*unsafe.Sizeof(uintptr(0))))
}
func call(p uintptr, index uintptr, args ...uintptr) uintptr {
	a := append([]uintptr{p}, args...)
	r, _, _ := syscall.SyscallN(method(p, index), a...)
	return r
}
func check(hr uintptr) error {
	if int32(hr) < 0 {
		return fmt.Errorf("COM HRESULT 0x%08X", uint32(hr))
	}
	return nil
}
func release(p uintptr) {
	if p != 0 {
		call(p, 2)
	}
}
func utf16String(p uintptr) string {
	if p == 0 {
		return ""
	}
	var units []uint16
	for offset := uintptr(0); ; offset += 2 {
		unit := *(*uint16)(unsafe.Pointer(p + offset))
		if unit == 0 {
			break
		}
		units = append(units, unit)
	}
	return string(utf16.Decode(units))
}

func withCOM(fn func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInit.Call(0, 2) // STA; a borrowed MTA thread returns RPC_E_CHANGED_MODE.
	if err := check(hr); err != nil {
		return err
	}
	defer coUninit.Call()
	return fn()
}

func create(clsid, iid *syscall.GUID) (uintptr, error) {
	var object uintptr
	hr, _, _ := coCreate.Call(uintptr(unsafe.Pointer(clsid)), 0, 1, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&object)))
	return object, check(hr)
}

func deviceID(device uintptr) (string, error) {
	var ptr uintptr
	if err := check(call(device, 5, uintptr(unsafe.Pointer(&ptr)))); err != nil {
		return "", err
	}
	defer coFree.Call(ptr)
	return utf16String(ptr), nil
}

func Enumerate() (devices []Device, defaultID string, err error) {
	err = withCOM(func() error {
		enum, e := create(&clsEnumerator, &iidEnumerator)
		if e != nil {
			return e
		}
		defer release(enum)
		var def uintptr
		if check(call(enum, 4, 0, 1, uintptr(unsafe.Pointer(&def)))) == nil {
			defaultID, _ = deviceID(def)
			release(def)
		}
		var collection uintptr
		if e = check(call(enum, 3, 0, 1, uintptr(unsafe.Pointer(&collection)))); e != nil {
			return e
		}
		defer release(collection)
		var count uint32
		if e = check(call(collection, 3, uintptr(unsafe.Pointer(&count)))); e != nil {
			return e
		}
		for i := uint32(0); i < count; i++ {
			var device uintptr
			if e = check(call(collection, 4, uintptr(i), uintptr(unsafe.Pointer(&device)))); e != nil {
				return e
			}
			id, e := deviceID(device)
			if e != nil {
				release(device)
				return e
			}
			var props uintptr
			e = check(call(device, 4, 0, uintptr(unsafe.Pointer(&props))))
			release(device)
			if e != nil {
				return e
			}
			var value propVariant
			e = check(call(props, 5, uintptr(unsafe.Pointer(&friendlyName)), uintptr(unsafe.Pointer(&value))))
			release(props)
			if e != nil {
				return e
			}
			name := id
			if value.VT == 31 && value.Pointer != 0 {
				name = utf16String(value.Pointer)
			}
			variantClear.Call(uintptr(unsafe.Pointer(&value)))
			devices = append(devices, Device{ID: id, Name: name})
		}
		return nil
	})
	return
}

func SetDefault(id string) error {
	if id == "" {
		return fmt.Errorf("未选择声音设备")
	}
	return withCOM(func() error {
		policy, err := create(&clsPolicy, &iidPolicy)
		if err != nil {
			return err
		}
		defer release(policy)
		ptr, err := syscall.UTF16PtrFromString(id)
		if err != nil {
			return err
		}
		for role := uintptr(0); role < 3; role++ {
			if err = check(call(policy, 13, uintptr(unsafe.Pointer(ptr)), role)); err != nil {
				return fmt.Errorf("设置声音角色 %d: %w", role, err)
			}
		}
		return nil
	})
}
