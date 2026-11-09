# Clipboard Fix

用 **Go + Wails 3** 编写的 macOS 通用剪贴板诊断与修复工具。把「手机能复制到 Mac，但 Mac 的内容传不到手机」的排查过程整理成一个中文桌面应用。

[下载最新版本](https://github.com/nbb2025/macos-clipboard-fix/releases/latest) · [修复原理与操作说明](docs/repair-process.md)

## 可以做什么

| 页面 | 功能 |
| --- | --- |
| 连接概览 | 检查接力、Wi-Fi、蓝牙、附近连接、当前用户服务，以及发送/接收缓存的时间和大小 |
| 修复工具 | 刷新接力与剪贴板服务、设备发现服务、身份服务；重新启用接力；分别备份并重建发送/接收缓存 |
| 跨设备复测 | 生成新的纯文字测试码，通过 `pbcopy` 写入并用 `pbpaste` 核对，再记录 iPhone/iPad 的实际粘贴结果 |
| 活动记录 | 查看真实执行进度和错误，采样接力日志的统计摘要，导出本地 JSON 报告 |

还提供 Maccy / Snipaste 临时退出与重新打开、系统设置入口，以及手动代理对照指南和 90 秒恢复提醒。

## 使用

1. 从 Releases 下载 Universal DMG 或 ZIP，将 `Clipboard Fix.app` 放入「应用程序」。包含 Apple Silicon 和 Intel 两种架构，构建最低系统版本为 macOS 12。
2. 打开应用，先看「连接概览」，再到「跨设备复测」生成新文字。两端使用同一个 Apple 账户、接力已开启，并保持设备解锁、放在附近。
3. 如果手机可以发到 Mac，而 Mac 发不出去，在「修复工具」选择「重建发送缓存」。程序会先保留旧文件，再重新加载接力服务。**新缓存可能直到下一次实际共享才生成。**
4. 重新生成测试文字，在 iPhone/iPad 备忘录直接粘贴。确认文字完全一致后点击「已成功」。程序不会仅凭服务重启或缓存变化宣告恢复。

当前发布包采用本地 ad-hoc 签名，**尚无 Apple Developer ID 签名和公证**；从网络下载后，macOS 可能阻止直接打开。也可以从源码构建。原生窗口已在 macOS 27.3 / Apple Silicon 上检查，Intel 版本完成交叉编译，未做 Intel 实机验证。

### 备份与权限

- 修复只操作当前用户的固定系统服务和剪贴板缓存，不需要管理员密码。
- 旧缓存重命名保留在原目录，备份名称带时间和随机标识。恢复时如果原位置已有新文件，会先保留新文件，再恢复旧备份。
- 受 macOS 隐私保护的缓存，会通过访达完成重命名；系统可能请求允许本应用控制「访达」。权限入口为「系统设置 → 隐私与安全性 → 自动化」。未授权时会显示失败原因，不会要求默认授予完全磁盘访问权限。
- 「生成测试文字」会替换当前剪贴板；重启 `pboard` 会清空当前剪贴板。执行前界面会说明影响。
- 备份登记和最多 200 条活动记录保存在 `~/Library/Application Support/Clipboard Fix/state.json`。跨设备成功标记只保留在本次会话，重新启动后需重新复测。
- 手机端开关、设备重启和代理对照由使用者手动操作。程序读取 Clash Verge 本地控制器状态，不会修改代理、TUN、Wi-Fi 或蓝牙开关。

### 隐私

诊断只读取缓存的路径、大小、时间等元数据，不读取旧缓存正文；测试文字由本程序随机生成。核对时只判断当前剪贴板是否等于测试码，不记录或返回其他剪贴板正文。

应用没有遥测、云端上传、外部字体或远程脚本。日志采样只返回有限的计数摘要，不导出原始系统日志。JSON 报告包含诊断状态、缓存元数据、备份登记和活动记录，用户主目录会替换为 `~`。报告仍包含系统版本、硬件型号、缓存文件名和时间，分享前可以自行检查。

## 从源码构建

需要 macOS、Xcode Command Line Tools、Go 1.26.10、Node.js 24 和 npm。Go 可按 `go.mod` 自动选择工具链。

```sh
git clone https://github.com/nbb2025/macos-clipboard-fix.git
cd macos-clipboard-fix
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0

# 当前架构的 .app
wails3 task package

# Apple Silicon + Intel 通用 .app
wails3 task package:universal

# 生成 ZIP、DMG 和 SHA256SUMS
bash scripts/package-macos.sh
```

应用位于 `bin/Clipboard Fix.app`，安装包位于 `dist/`。Wails 可能将 `bin` 链接到本机 Application Support 下的构建目录，以避免 iCloud 扩展属性干扰代码签名。

开发模式：

```sh
wails3 task dev
```

只读命令行诊断：

```sh
./bin/macos-clipboard-fix --diagnose-json
```

该入口不创建应用状态目录、不写剪贴板、不重启服务。输出含本机缓存路径，分享前应自行脱敏。

## 验证与发布

```sh
go test -race ./...
go vet ./...
npm run build --prefix frontend
codesign --verify --deep --strict 'bin/Clipboard Fix.app'
lipo -archs 'bin/Clipboard Fix.app/Contents/MacOS/macos-clipboard-fix'
```

涉及文件变更的自动测试全部使用临时用户目录和替身命令执行器，覆盖备份恢复、新文件保留、登记写入失败、路径和符号链接限制、测试文字被覆盖、只读模式等情况，不会拿真实系统缓存做测试。

GitHub Actions 在每次推送和 PR 上测试并构建通用应用；推送与应用版本一致的 `v*` 标签后，会发布 ZIP、DMG 和校验文件。

核心实现位于 `internal/core/`；Wails 接口位于 `clipboardservice.go`；React、TypeScript、Tailwind CSS 与 Radix Dialog 界面位于 `frontend/`。
