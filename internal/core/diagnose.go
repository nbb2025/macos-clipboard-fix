package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

func parsePower(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if strings.HasSuffix(s, ": on") {
		return "ok"
	}
	if strings.HasSuffix(s, ": off") {
		return "warning"
	}
	return "unknown"
}
func wifiInterface(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.Contains(line, "Hardware Port: Wi-Fi") || strings.Contains(line, "Hardware Port: AirPort") {
			for _, n := range lines[i+1 : min(i+4, len(lines))] {
				if strings.HasPrefix(n, "Device: ") {
					name := strings.TrimPrefix(n, "Device: ")
					if regexp.MustCompile(`^en[0-9]+$`).MatchString(name) {
						return name
					}
				}
			}
		}
	}
	return ""
}
func (e *Engine) cacheState(ctx context.Context, kind string) CacheState {
	r := CacheState{Kind: kind}
	key := "kLocalPasteboardBlobName"
	if kind == "receive" {
		key = "kRemotePasteboardBlobName"
	}
	p, ok := e.pref(ctx, key, false)
	if !ok || p == "" {
		r.Detail = "尚无缓存记录"
		return r
	}
	if err := validateCachePath(e.Home, p); err != nil {
		r.Detail = "缓存路径不符合系统目录，已停止访问"
		return r
	}
	r.Path = p
	s, err := os.Lstat(p)
	if err != nil {
		r.Detail = "缓存暂不存在或系统限制了访问"
		return r
	}
	if !s.Mode().IsRegular() {
		r.Detail = "不是常规缓存文件"
		return r
	}
	r.Exists = true
	r.Bytes = s.Size()
	r.Modified = s.ModTime().Format(time.RFC3339)
	r.AgeSeconds = max(0, int64(time.Since(s.ModTime()).Seconds()))
	r.Detail = "仅检查文件大小与更新时间"
	return r
}
func (e *Engine) Diagnose(ctx context.Context) Report {
	r := Report{At: time.Now().Format(time.RFC3339), Checks: make([]Check, 4), Services: make([]ServiceState, len(serviceOrder)), Caches: make([]CacheState, 2), Tools: make([]ToolState, 2), Findings: []string{}, SuggestedAction: "quick"}
	if runtime.GOOS != "darwin" {
		r.Findings = []string{"诊断与修复仅支持 macOS"}
		return r
	}
	var wg sync.WaitGroup
	run := func(fn func()) { wg.Add(1); go func() { defer wg.Done(); fn() }() }
	run(func() {
		r.Version, _ = e.command(ctx, "/usr/bin/sw_vers", "-productVersion")
		r.Version = strings.TrimSpace(r.Version)
		r.Model, _ = e.command(ctx, "/usr/sbin/sysctl", "-n", "hw.model")
		r.Model = strings.TrimSpace(r.Model)
	})
	run(func() {
		a, hasA := e.pref(ctx, "ActivityAdvertisingAllowed", true)
		b, hasB := e.pref(ctx, "ActivityReceivingAllowed", true)
		c := Check{"handoff", "接力", "ok", "发送与接收均已开启"}
		if (hasA && a == "0") || (hasB && b == "0") {
			c.Status = "warning"
			c.Detail = "接力的发送或接收开关已关闭"
		} else if !hasA || !hasB {
			c.Status = "unknown"
			c.Detail = "跟随系统默认，请在系统设置中核对"
		}
		r.Checks[0] = c
	})
	run(func() {
		hardware, _ := e.command(ctx, "/usr/sbin/networksetup", "-listallhardwareports")
		dev := wifiInterface(hardware)
		c := Check{"wifi", "Wi-Fi", "unknown", "未能识别无线网卡"}
		if dev != "" {
			s, _ := e.command(ctx, "/usr/sbin/networksetup", "-getairportpower", dev)
			c.Status = parsePower(s)
			if c.Status == "ok" {
				c.Detail = "无线网卡已开启"
			} else if c.Status == "warning" {
				c.Detail = "请开启 Wi-Fi，以便设备间传输"
			}
		}
		r.Checks[1] = c
	})
	run(func() {
		s, err := e.command(ctx, "/usr/sbin/system_profiler", "SPBluetoothDataType", "-json")
		var p struct {
			Items []struct {
				Controller struct {
					State string `json:"controller_state"`
				} `json:"controller_properties"`
			} `json:"SPBluetoothDataType"`
		}
		c := Check{"bluetooth", "蓝牙", "unknown", "未能读取蓝牙状态"}
		if err == nil && json.Unmarshal([]byte(s), &p) == nil && len(p.Items) > 0 {
			if p.Items[0].Controller.State == "attrib_on" {
				c.Status = "ok"
				c.Detail = "蓝牙已开启，可发现附近设备"
			} else {
				c.Status = "warning"
				c.Detail = "请开启蓝牙，以便设备发现"
			}
		}
		r.Checks[2] = c
	})
	run(func() {
		s, err := e.command(ctx, "/sbin/ifconfig", "awdl0")
		c := Check{"awdl", "附近连接", "unknown", "未能读取点对点网络状态"}
		if err == nil {
			if strings.Contains(s, "UP,") || strings.Contains(s, ",UP,") {
				c.Status = "ok"
				c.Detail = "Apple 点对点网络已就绪"
			} else {
				c.Status = "warning"
				c.Detail = "点对点接口尚未启动"
			}
		}
		r.Checks[3] = c
	})
	for i, name := range serviceOrder {
		i, name := i, name
		run(func() { p := e.pid(ctx, name); r.Services[i] = ServiceState{name, serviceLabels[name], p, p > 0} })
	}
	for i, kind := range []string{"send", "receive"} {
		i, kind := i, kind
		run(func() { r.Caches[i] = e.cacheState(ctx, kind) })
	}
	for i, name := range []string{"Maccy", "Snipaste"} {
		i, name := i, name
		run(func() { r.Tools[i] = ToolState{name, e.pid(ctx, name) > 0} })
	}
	run(func() { r.Proxy = e.ProxyStatus(ctx) })
	wg.Wait()
	for _, c := range r.Checks {
		if c.Status == "warning" {
			r.Findings = append(r.Findings, c.Detail)
		}
	}
	if r.Caches[0].Exists && r.Caches[1].Exists && r.Caches[0].AgeSeconds > r.Caches[1].AgeSeconds+3600 && r.Caches[1].AgeSeconds < 1800 {
		r.Findings = append(r.Findings, "接收缓存近期更新，发送缓存较旧。若仅 Mac 发出失败，可先备份并重建发送缓存；时间差本身不代表故障。")
		r.SuggestedAction = "cache-send"
	}
	if len(r.Findings) == 0 {
		r.Findings = append(r.Findings, "基础条件已检查，跨设备复制仍需要在 iPhone 或 iPad 上实际粘贴确认。")
	}
	e.mu.Lock()
	e.lastReport = r
	e.mu.Unlock()
	e.log("info", "已完成只读诊断；跨设备结果待复测")
	return r
}
func (e *Engine) CollectSignals(ctx context.Context) ([]Signal, error) {
	predicate := `(process == "useractivityd" OR process == "sharingd") AND (eventMessage CONTAINS[c] "wrapping key" OR eventMessage CONTAINS[c] "DATA PROVIDER" OR eventMessage CONTAINS[c] "DATA REQUESTER" OR eventMessage CONTAINS[c] "Pasteboard data fetch")`
	s, err := e.command(ctx, "/usr/bin/log", "show", "--last", "3m", "--style", "compact", "--info", "--predicate", predicate)
	if err != nil {
		return nil, err
	}
	metrics := []Signal{{"接力密钥重试", strings.Count(s, "wrapping key is unavailable"), "这是认证状态的线索，单条日志不足以确定原因。"}, {"发送端数据请求", strings.Count(s, "[DATA PROVIDER]"), "出现请求表示其他设备已进入取数据阶段。"}, {"接收端数据请求", strings.Count(s, "[DATA REQUESTER]"), "用于对照单向复制故障，不读取实际剪贴板内容。"}}
	e.log("info", fmt.Sprintf("已采样最近 3 分钟日志：密钥重试 %d，发送请求 %d，接收请求 %d", metrics[0].Count, metrics[1].Count, metrics[2].Count))
	return metrics, nil
}
func cacheDirectory(home string) string {
	return filepath.Join(home, "Library", "Group Containers", "group.com.apple.coreservices.useractivityd", "shared-pasteboard")
}
