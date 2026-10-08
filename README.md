# Apex Runner Go

使用 [MyGo](https://github.com/egoist/mygo) native UI 编写的 Windows 游戏环境切换工具。界面与系统服务均为 Go，无 WebView 或前端运行时。

## 功能

- 游戏/日常两套独立方案，分别设置 Win 键、Ctrl+Esc、Alt+Shift、输入法和声音输出。
- 当前状态手动控制，枚举并切换默认声音输出设备。
- 全局快捷键切换两套方案，支持录制和修改，默认 `Ctrl+Alt+K`。
- 启动加速器、语音软件、Steam，打开显示及应用音量设置。
- 浏览或拖入 exe、lnk 路径，解析快捷方式目标。
- 启动自动优化、退出自动恢复、单实例唤起已有窗口。
- 配置保存于 `%AppData%\Apex Runner\config.json`。

## 开发与构建

需要 Windows 10/11 和 Go 1.27.1 或更新版本。

```powershell
go mod download
go test ./...
go tool mygo dev
go tool mygo build
```

只需要可直接运行的 exe 时，可用：

```powershell
go build -ldflags "-H windowsgui" -o build/ApexRunner.exe .
```

`go tool mygo build` 会生成带图标、安装器和版本元数据的发布包，首次构建需要下载 NSIS。直接 `go build` 不依赖 NSIS。

## 自动发布

将 `mygo.json` 中的 `version` 改成新版本并提交，然后推送同名的 `v` 前缀标签：

```powershell
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions 会构建 Windows x64 生产版，并自动创建公开 Release，上传安装器、便携 ZIP 和 `SHA256SUMS.txt`。标签版本必须与 `mygo.json` 一致；重新运行工作流会替换同名附件。当前构建未进行代码签名。

当前环境如无法访问默认 Go 代理，可仅在当前终端设置 `GOPROXY=https://goproxy.cn`。

游戏以管理员权限运行时，键盘 Hook 可能也需要管理员权限。声音输出切换使用 Windows Core Audio 的 `IPolicyConfig` 接口，需在目标 Windows 版本和设备上验证。
