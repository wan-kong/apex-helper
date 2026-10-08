//go:build windows

package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/wan-kong/apex-helper/internal/config"
	"github.com/wan-kong/apex-helper/internal/winapi/keyboard"
)

func TestMainView(t *testing.T) {
	for _, width := range []int{760, 1040} {
		a := &app{config: config.Default(), keyboard: keyboard.New(), editing: "optimize", inputLanguage: "unknown"}
		view := ui.NewTester(a.view, width, 760)
		for _, label := range []string{"APEX RUNNER", "一键优化 · 进入游戏", "一键恢复 · 返回日常", "当前状态", "一键方案", "快速启动", "声音输出", "快捷键与启动行为"} {
			if !view.HasText(label) {
				t.Errorf("width %d: missing %s", width, label)
			}
		}
		if dir := os.Getenv("APEX_SNAPSHOT_DIR"); dir != "" {
			f, err := os.Create(filepath.Join(dir, fmt.Sprintf("apex-%d.png", width)))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, view.Image()); err != nil {
				f.Close()
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
}
