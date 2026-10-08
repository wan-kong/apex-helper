package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var shellExecute = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")

func Open(target string) error {
	target = strings.TrimSpace(strings.Trim(target, `"`))
	if target == "" {
		return fmt.Errorf("尚未配置程序路径")
	}
	if !strings.Contains(target, "://") && !strings.HasPrefix(target, "ms-settings:") {
		if _, err := os.Stat(target); err != nil {
			return fmt.Errorf("路径不可用 %s: %w", target, err)
		}
	}
	p, _ := syscall.UTF16PtrFromString(target)
	verb, _ := syscall.UTF16PtrFromString("open")
	r, _, err := shellExecute.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(p)), 0, 0, 1)
	if r <= 32 {
		return fmt.Errorf("打开 %s 失败，ShellExecute 返回 %d: %w", target, r, err)
	}
	return nil
}

// Resolve accepts executables and Windows shortcuts. For .lnk files it asks
// the Windows Script Host COM object for the actual target.
func Resolve(path string) (string, error) {
	path = strings.TrimSpace(strings.Trim(path, `"`))
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(path), ".lnk") {
		if !strings.EqualFold(filepath.Ext(path), ".exe") {
			return "", fmt.Errorf("请选择 exe 或 .lnk 文件")
		}
		return filepath.Abs(path)
	}
	script := `$s=(New-Object -ComObject WScript.Shell).CreateShortcut($env:APEX_SHORTCUT_PATH); [Console]::Write($s.TargetPath)`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(), "APEX_SHORTCUT_PATH="+path)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("解析快捷方式失败: %w", err)
	}
	target := strings.TrimSpace(string(out))
	if target == "" {
		return "", fmt.Errorf("快捷方式没有目标路径")
	}
	if _, err = os.Stat(target); err != nil {
		return "", fmt.Errorf("快捷方式目标不可用: %w", err)
	}
	return target, nil
}
