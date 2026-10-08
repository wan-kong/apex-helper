//go:build windows

package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/wan-kong/apex-helper/internal/launcher"
	"github.com/wan-kong/apex-helper/internal/winapi/inputmethod"
)

var (
	ink    = ui.Hex("#162432")
	muted  = ui.Hex("#637383")
	line   = ui.Hex("#D8E1E8")
	panel  = ui.Hex("#FFFFFF")
	canvas = ui.Hex("#F3F6F8")
	accent = ui.Hex("#157A72")
)

func (a *app) view(c *ui.Context) {
	t := *c.Theme()
	t.Accent = accent
	t.Radius = 9
	c.SetTheme(&t)
	ui.Column(c).Fill().Background(canvas).Children(func() {
		ui.Row(c).FillWidth().Padding(20, 28).Gap(12).Background(ink).Children(func() {
			ui.Column(c).Grow(1).Gap(3).Children(func() {
				ui.Text(c, "APEX RUNNER").FontSize(21).Bold().TextColor(ui.Hex("#FFFFFF"))
				ui.Text(c, "游戏前安静切换，结束后快速恢复").FontSize(12).TextColor(ui.Hex("#B6C7D2"))
			})
			mode := "待命"
			if a.activeMode == "optimize" {
				mode = "游戏模式"
			} else if a.activeMode == "restore" {
				mode = "日常模式"
			}
			ui.Text(c, mode).FontSize(13).Bold().TextColor(ui.Hex("#FFFFFF"))
		})
		ui.Scroll(c).Grow(1).Padding(24, 28).Gap(18).Children(func() {
			a.actions(c)
			a.systemStatus(c)
			ui.Row(c).Gap(18).AlignItems(ui.Start).Children(func() {
				ui.Column(c).Grow(1).Gap(18).Children(func() { a.profilePanel(c); a.launchPanel(c) })
				ui.Column(c).Grow(1).Gap(18).Children(func() { a.audioPanel(c); a.settingsPanel(c) })
			})
		})
		ui.Row(c).FillWidth().Padding(12, 28).Background(panel).BorderWidth(1, 0, 0, 0).BorderColor(line).Children(func() {
			msg := a.message
			if msg == "" {
				msg = "就绪。建议先检查两套方案和声音设备。"
			}
			ui.Text(c, msg).FontSize(12).TextColor(muted)
		})
	})
}

func section(c *ui.Context, title, desc string, body func()) {
	ui.Column(c).FillWidth().Padding(18).Gap(12).Radius(12).Background(panel).Border(1, line).Children(func() {
		ui.Text(c, title).FontSize(16).Bold().TextColor(ink)
		if desc != "" {
			ui.Text(c, desc).FontSize(12).TextColor(muted)
		}
		body()
	})
}

func (a *app) actions(c *ui.Context) {
	ui.Row(c).FillWidth().Gap(12).Children(func() {
		if ui.PrimaryButton(c, "一键优化 · 进入游戏").Grow(1).Padding(14, 18).Clicked() {
			a.apply("optimize")
		}
		if ui.Button(c, "一键恢复 · 返回日常").Grow(1).Padding(14, 18).Clicked() {
			a.apply("restore")
		}
	})
}

func (a *app) systemStatus(c *ui.Context) {
	section(c, "当前状态", "手动控制立即生效；Ctrl+Esc 随 Win 键一起拦截。", func() {
		ui.Row(c).Gap(14).Children(func() {
			win := a.keyboard.WinDisabled()
			if ui.Checkbox(c, &win, "禁用 Win 键").Changed() {
				a.keyboard.SetWinDisabled(win)
				a.report(nil)
			}
			alt := a.keyboard.AltShiftDisabled()
			if ui.Checkbox(c, &alt, "禁用 Alt+Shift").Changed() {
				a.keyboard.SetAltShiftDisabled(alt)
				a.report(nil)
			}
			ui.Text(c, "前台输入法："+language(a.inputLanguage)).TextColor(muted)
			if ui.Button(c, "切中文").Clicked() {
				a.setLanguage(false)
			}
			if ui.Button(c, "切英文").Clicked() {
				a.setLanguage(true)
			}
		})
	})
}

func language(code string) string {
	if code == "zh" {
		return "中文"
	}
	if code == "en" {
		return "英文"
	}
	return "未知"
}

func (a *app) setLanguage(english bool) {
	a.report(inputmethod.SetEnglish(english))
	a.inputLanguage = inputmethod.Current()
}

func (a *app) profilePanel(c *ui.Context) {
	section(c, "一键方案", "分别设置游戏与日常状态，声音设备可留空。", func() {
		index := 0
		if a.editing == "restore" {
			index = 1
		}
		if ui.Segmented(c, &index, "优化方案", "恢复方案").Changed() {
			if index == 0 {
				a.editing = "optimize"
			} else {
				a.editing = "restore"
			}
		}
		p := a.selectedProfile()
		ui.Checkbox(c, &p.WinKeyDisabled, "禁用 Win 键与 Ctrl+Esc")
		ui.Checkbox(c, &p.AltShiftDisabled, "禁用 Alt+Shift")
		ui.Checkbox(c, &p.InputEnglish, "切换为英文输入法")
		options := append([]string{"不切换声音设备"}, a.audioNames()...)
		selected := a.nameForID(p.AudioDeviceID)
		if selected == "" {
			selected = options[0]
		}
		if ui.Select(c, &selected, options).Label("方案声音设备").FillWidth().Changed() {
			p.AudioDeviceID = a.idForName(selected)
		}
		if p.AudioDeviceID != "" && !a.hasAudio(p.AudioDeviceID) {
			ui.Text(c, "已配置的设备未连接，执行方案时会提示。").FontSize(12).TextColor(ui.Hex("#AC5C1D"))
		}
		if ui.Button(c, "保存两套方案").Clicked() {
			a.report(a.save())
		}
	})
}

func (a *app) audioPanel(c *ui.Context) {
	section(c, "声音输出", "选择后立即设为 Windows 默认输出设备。", func() {
		if len(a.devices) == 0 {
			ui.Text(c, "当前没有可用的输出设备").TextColor(muted)
		} else {
			selected := a.nameForID(a.selectedAudio)
			if selected == "" {
				selected = a.audioNames()[0]
			}
			if ui.Select(c, &selected, a.audioNames()).Label("默认输出").FillWidth().Changed() {
				a.changeAudio(a.idForName(selected))
			}
		}
		if ui.Button(c, "刷新设备").Clicked() {
			a.refreshAudio()
		}
	})
}

func (a *app) hasAudio(id string) bool {
	for _, d := range a.devices {
		if d.ID == id {
			return true
		}
	}
	return false
}
func (a *app) audioNames() []string {
	res := make([]string, 0, len(a.devices))
	seen := map[string]int{}
	for _, d := range a.devices {
		seen[d.Name]++
	}
	for _, d := range a.devices {
		if seen[d.Name] > 1 {
			res = append(res, fmt.Sprintf("%s · %s", d.Name, shortID(d.ID)))
		} else {
			res = append(res, d.Name)
		}
	}
	return res
}
func shortID(id string) string {
	if len(id) > 8 {
		return id[len(id)-8:]
	}
	return id
}
func (a *app) nameForID(id string) string {
	for i, d := range a.devices {
		if d.ID == id {
			return a.audioNames()[i]
		}
	}
	return ""
}
func (a *app) idForName(name string) string {
	for i, n := range a.audioNames() {
		if n == name {
			return a.devices[i].ID
		}
	}
	return ""
}

func (a *app) launchPanel(c *ui.Context) {
	section(c, "快速启动", "路径支持 exe 和 .lnk；也可打开 Windows 系统设置。", func() {
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "加速器").Clicked() {
				a.launch(a.config.Paths.Accelerator)
			}
			if ui.Button(c, "语音软件").Clicked() {
				a.launch(a.config.Paths.Voice)
			}
			if ui.Button(c, "Steam").Clicked() {
				a.launch(a.config.Paths.Steam)
			}
		})
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "显示设置").Clicked() {
				a.launch("ms-settings:display")
			}
			if ui.Button(c, "应用音量").Clicked() {
				a.launch("ms-settings:apps-volume")
			}
		})
		pathField(c, "加速器路径", &a.config.Paths.Accelerator, a)
		pathField(c, "语音路径", &a.config.Paths.Voice, a)
		pathField(c, "Steam 路径", &a.config.Paths.Steam, a)
		if ui.Button(c, "保存启动路径").Clicked() {
			a.report(a.save())
		}
	})
}

func pathField(c *ui.Context, label string, value *string, a *app) {
	ui.Text(c, label).FontSize(12).TextColor(muted)
	row := ui.Row(c).Gap(6)
	row.Children(func() {
		ui.TextInput(c, value).Placeholder("选择 exe 或 .lnk").Label(label).Grow(1)
		if ui.Button(c, "浏览").Clicked() {
			paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.window, Title: "选择" + label, Filters: []mygo.FileFilter{{Name: "程序或快捷方式", Extensions: []string{"exe", "lnk"}}}})
			if err != nil {
				a.report(err)
			} else if len(paths) > 0 {
				resolved, e := launcher.Resolve(paths[0])
				if e != nil {
					a.report(e)
				} else {
					*value = resolved
				}
			}
		}
	})
	if files := row.DroppedFiles(); len(files) > 0 {
		resolved, err := launcher.Resolve(files[0])
		if err == nil {
			*value = resolved
			err = a.save()
		}
		a.report(err)
	}
	if row.FileDragOver() {
		row.Border(2, accent).Radius(9)
	}
}

func (a *app) settingsPanel(c *ui.Context) {
	section(c, "快捷键与启动行为", "快捷键使用 MyGo 格式，例如 Ctrl+Alt+K。", func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.TextInput(c, &a.shortcutDraft).Label("全局快捷键").Grow(1)
			caption := "录制"
			if a.recording {
				caption = "按下组合键…"
			}
			if ui.Button(c, caption).Disabled(a.recording).Clicked() {
				a.recording = true
				a.keyboard.RecordNext(func(chord string) {
					if a.window != nil {
						a.window.Update(func() {
							a.shortcutDraft = chord
							a.recording = false
							a.message = "已录制 " + chord + "，点击应用快捷键保存。"
						})
					}
				})
			}
			if ui.Button(c, "应用快捷键").Clicked() {
				a.report(a.registerShortcut(a.shortcutDraft))
			}
		})
		ui.Checkbox(c, &a.config.StartOneClickOptimize, "启动时自动执行优化")
		ui.Checkbox(c, &a.config.CloseOneClickRestore, "关闭时自动执行恢复")
		if ui.Button(c, "保存启动行为").Clicked() {
			a.report(a.save())
		}
		ui.Text(c, "权限提示：若游戏以管理员身份运行，拦截系统按键也可能需要管理员权限。").FontSize(12).TextColor(muted)
		ui.Text(c, strings.TrimSpace(a.store.Path)).FontSize(11).TextColor(muted)
	})
}
