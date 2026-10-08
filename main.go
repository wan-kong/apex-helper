//go:build windows

package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/wan-kong/apex-helper/internal/config"
	"github.com/wan-kong/apex-helper/internal/launcher"
	"github.com/wan-kong/apex-helper/internal/winapi/audio"
	"github.com/wan-kong/apex-helper/internal/winapi/inputmethod"
	"github.com/wan-kong/apex-helper/internal/winapi/keyboard"
)

type app struct {
	store         config.Store
	config        config.Config
	keyboard      *keyboard.Service
	window        *mygo.Window
	devices       []audio.Device
	defaultAudio  string
	activeMode    string
	inputLanguage string
	message       string
	busy          bool
	recording     bool
	editing       string
	shortcutDraft string
	selectedAudio string
	profileAudio  string
}

func newApp() (*app, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return nil, err
	}
	store := config.Store{Path: path}
	cfg, err := store.Load()
	if err != nil {
		return nil, fmt.Errorf("读取配置: %w", err)
	}
	return &app{store: store, config: cfg, keyboard: keyboard.New(), shortcutDraft: cfg.Shortcut, editing: "optimize", inputLanguage: "unknown"}, nil
}

func (a *app) report(err error) {
	if err != nil {
		a.message = "操作失败：" + err.Error()
	} else {
		a.message = "操作已完成"
	}
}

func (a *app) save() error { return a.store.Save(a.config) }

func (a *app) registerShortcut(value string) error {
	value = strings.ReplaceAll(strings.TrimSpace(value), " ", "")
	if value == "" {
		return fmt.Errorf("快捷键不能为空")
	}
	old := strings.ReplaceAll(a.config.Shortcut, " ", "")
	if value == old && mygo.GlobalShortcut.IsRegistered(old) {
		return nil
	}
	if old != "" {
		mygo.GlobalShortcut.Unregister(old)
	}
	if err := mygo.GlobalShortcut.Register(value, func() { a.toggle() }); err != nil {
		if old != "" {
			_ = mygo.GlobalShortcut.Register(old, func() { a.toggle() })
		}
		return fmt.Errorf("快捷键 %s 注册失败，可能已被占用: %w", value, err)
	}
	a.config.Shortcut = value
	if err := a.save(); err != nil {
		return err
	}
	a.shortcutDraft = value
	return nil
}

func (a *app) initialize() {
	if err := a.keyboard.Start(); err != nil {
		a.report(err)
	}
	if err := a.registerShortcut(a.config.Shortcut); err != nil {
		a.report(err)
	}
	a.refreshAudio()
	a.inputLanguage = inputmethod.Current()
	if a.config.StartOneClickOptimize {
		a.apply("optimize")
	}
}

func (a *app) refreshAudio() {
	devices, id, err := audio.Enumerate()
	if err != nil {
		a.report(fmt.Errorf("读取声音设备: %w", err))
		return
	}
	a.devices, a.defaultAudio = devices, id
	a.selectedAudio = id
}

func (a *app) selectedProfile() *config.Profile {
	if a.editing == "restore" {
		return &a.config.Profiles.Restore
	}
	return &a.config.Profiles.Optimize
}

func (a *app) apply(mode string) {
	var p config.Profile
	if mode == "optimize" {
		p = a.config.Profiles.Optimize
	} else {
		p = a.config.Profiles.Restore
	}
	var errs []error
	a.keyboard.SetWinDisabled(p.WinKeyDisabled)
	a.keyboard.SetAltShiftDisabled(p.AltShiftDisabled)
	if err := inputmethod.SetEnglish(p.InputEnglish); err != nil {
		errs = append(errs, err)
	}
	if p.AudioDeviceID != "" {
		found := false
		for _, d := range a.devices {
			if d.ID == p.AudioDeviceID {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Errorf("方案中的声音设备未连接"))
		} else if err := audio.SetDefault(p.AudioDeviceID); err != nil {
			errs = append(errs, err)
		} else {
			a.defaultAudio = p.AudioDeviceID
			a.selectedAudio = p.AudioDeviceID
		}
	}
	a.activeMode = mode
	a.inputLanguage = inputmethod.Current()
	if len(errs) > 0 {
		a.report(errors.Join(errs...))
	} else if mode == "optimize" {
		a.message = "游戏方案已应用"
	} else {
		a.message = "日常方案已恢复"
	}
}

func (a *app) toggle() {
	if a.activeMode == "optimize" {
		a.apply("restore")
	} else {
		a.apply("optimize")
	}
	if a.window != nil {
		a.window.Invalidate()
	}
}

func (a *app) shutdown() {
	a.keyboard.RecordNext(nil)
	if a.config.CloseOneClickRestore {
		a.apply("restore")
	}
	mygo.GlobalShortcut.UnregisterAll()
	a.keyboard.Stop()
}

func (a *app) changeAudio(id string) {
	if id == "" {
		return
	}
	if err := audio.SetDefault(id); err != nil {
		a.report(err)
		return
	}
	a.defaultAudio, a.selectedAudio = id, id
	a.report(nil)
}

func (a *app) launch(target string) { a.report(launcher.Open(target)) }

func main() {
	mygo.App.SetName("Apex Runner")
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	a, err := newApp()
	if err != nil {
		log.Fatal(err)
	}
	mygo.App.OnSecondInstance(func(_ []string, _ string) {
		if a.window != nil {
			a.window.Restore()
			a.window.Focus()
		}
	})
	mygo.App.WhenReady(func() {
		a.window = mygo.NewWindow(mygo.WindowOptions{Title: "Apex Runner", Width: 1040, Height: 760, MinWidth: 760, MinHeight: 580, StateKey: "main", Content: ui.View(a.view)})
		a.initialize()
		a.window.Invalidate()
	})
	mygo.App.OnWillQuit(func(_ *mygo.QuitEvent) { a.shutdown() })
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
