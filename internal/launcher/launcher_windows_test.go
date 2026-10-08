package launcher

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestResolveShortcut(t *testing.T) {
	target, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "test.lnk")
	script := `$s=(New-Object -ComObject WScript.Shell).CreateShortcut($env:APEX_TEST_LINK); $s.TargetPath=$env:APEX_TEST_TARGET; $s.Save()`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(), "APEX_TEST_LINK="+link, "APEX_TEST_TARGET="+target)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create link: %v, %s", err, out)
	}
	got, err := Resolve(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Fatalf("want %q, got %q", target, got)
	}
}
