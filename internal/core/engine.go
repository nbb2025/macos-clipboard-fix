package core

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const activityDomain = "com.apple.coreservices.useractivityd"

var errReadOnly = errors.New("只读诊断模式不允许修改系统")

var serviceLabels = map[string]string{
	"useractivityd": "com.apple.coreservices.useractivityd",
	"pboard":        "com.apple.pboard", "sharingd": "com.apple.sharingd",
	"rapportd": "com.apple.rapportd", "identityservicesd": "com.apple.identityservicesd",
}
var serviceOrder = []string{"useractivityd", "pboard", "sharingd", "rapportd", "identityservicesd"}
var toolBundles = map[string]string{"Maccy": "org.p0deje.Maccy", "Snipaste": "com.Snipaste"}

type Runner interface {
	Run(context.Context, string, []string, string) (string, error)
}
type CommandRunner struct{}
type cappedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if left := b.limit - b.Len(); left > 0 {
		if len(p) > left {
			p = p[:left]
		}
		_, _ = b.Buffer.Write(p)
	}
	return n, nil
}
func (CommandRunner) Run(ctx context.Context, name string, args []string, input string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	out := &cappedBuffer{limit: 1024 * 1024}
	stderr := &cappedBuffer{limit: 4096}
	cmd.Stdout = out
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return out.String(), ctx.Err()
		}
		if strings.Contains(stderr.String(), "1743") || strings.Contains(stderr.String(), "not authorized") {
			return out.String(), errors.New("需要在系统设置的「隐私与安全性 → 自动化」中允许本程序控制访达")
		}
		return out.String(), fmt.Errorf("%s 执行失败：%w", filepath.Base(name), err)
	}
	return out.String(), nil
}

type Engine struct {
	readOnly      bool
	Home          string
	DataDir       string
	Runner        Runner
	Emit          func(Progress)
	mu            sync.Mutex
	op            sync.Mutex
	history       []Event
	backups       []Backup
	verifications []Verification
	probe         Probe
	lastReport    Report
}

func New(home string) (*Engine, error) {
	e := &Engine{Home: home, DataDir: filepath.Join(home, "Library", "Application Support", "Clipboard Fix"), Runner: CommandRunner{}, history: []Event{}, backups: []Backup{}, verifications: []Verification{}}
	if err := os.MkdirAll(e.DataDir, 0700); err != nil {
		return nil, err
	}
	var saved struct {
		History []Event  `json:"history"`
		Backups []Backup `json:"backups"`
	}
	if b, err := os.ReadFile(filepath.Join(e.DataDir, "state.json")); err == nil && json.Unmarshal(b, &saved) == nil {
		if saved.History != nil {
			e.history = saved.History
		}
		if saved.Backups != nil {
			e.backups = saved.Backups
		}
	}
	for i, b := range e.backups {
		if !b.Ready && !b.Restored && validateBackupPath(home, b.Original, b.Path) == nil {
			if _, err := regularFile(b.Path); err == nil {
				e.backups[i].Ready = true
			}
		}
	}
	return e, nil
}
func (e *Engine) command(ctx context.Context, name string, args ...string) (string, error) {
	c, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return e.Runner.Run(c, name, args, "")
}
func (e *Engine) pref(ctx context.Context, key string, host bool) (string, bool) {
	a := []string{}
	if host {
		a = append(a, "-currentHost")
	}
	a = append(a, "read", activityDomain, key)
	s, err := e.command(ctx, "/usr/bin/defaults", a...)
	return strings.TrimSpace(s), err == nil
}
func (e *Engine) persistLocked() error {
	if e.readOnly {
		return nil
	}
	return writeJSON(filepath.Join(e.DataDir, "state.json"), struct {
		History []Event  `json:"history"`
		Backups []Backup `json:"backups"`
	}{e.history, e.backups})
}
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".clipboard-fix-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}
func (e *Engine) log(level, message string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.history = append(e.history, Event{time.Now().Format(time.RFC3339), level, strings.ReplaceAll(message, e.Home, "~")})
	if len(e.history) > 200 {
		e.history = e.history[len(e.history)-200:]
	}
	_ = e.persistLocked()
}
func (e *Engine) progress(action string, percent int, message, state string) {
	e.log(state, message)
	if e.Emit != nil {
		e.Emit(Progress{action, percent, message, state})
	}
}
func (e *Engine) Session() Session {
	e.mu.Lock()
	defer e.mu.Unlock()
	return Session{append([]Event{}, e.history...), append([]Backup{}, e.backups...), append([]Verification{}, e.verifications...)}
}
func (e *Engine) pid(ctx context.Context, name string) int {
	s, _ := e.command(ctx, "/usr/bin/pgrep", "-u", strconv.Itoa(os.Getuid()), "-x", name)
	for _, x := range strings.Fields(s) {
		n, err := strconv.Atoi(x)
		if err == nil && n > 1 {
			return n
		}
	}
	return 0
}
func (e *Engine) restart(ctx context.Context, name string) error {
	label, ok := serviceLabels[name]
	if !ok {
		return errors.New("不支持的系统服务")
	}
	old := e.pid(ctx, name)
	if old > 1 {
		if _, err := e.command(ctx, "/bin/kill", "-TERM", strconv.Itoa(old)); err != nil {
			return err
		}
	}
	_, _ = e.command(ctx, "/bin/launchctl", "kickstart", fmt.Sprintf("gui/%d/%s", os.Getuid(), label))
	for i := 0; i < 35; i++ {
		n := e.pid(ctx, name)
		if n > 1 && n != old {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	return fmt.Errorf("%s 未确认重新启动，请重新诊断", name)
}
func (e *Engine) Repair(ctx context.Context, action string) (err error) {
	if e.readOnly {
		return errReadOnly
	}
	if runtime.GOOS != "darwin" {
		return errors.New("修复功能仅支持 macOS")
	}
	if !e.op.TryLock() {
		return errors.New("已有操作正在进行")
	}
	defer e.op.Unlock()
	allowed := map[string]bool{"quick": true, "discovery": true, "identity": true, "handoff": true, "cache-send": true, "cache-receive": true}
	if !allowed[action] {
		return errors.New("不支持的修复操作")
	}
	e.mu.Lock()
	e.verifications = []Verification{}
	e.probe = Probe{}
	e.mu.Unlock()
	e.progress(action, 5, "开始执行修复", "running")
	defer func() {
		if err != nil {
			e.progress(action, 100, "操作未完成："+err.Error(), "error")
		} else {
			e.progress(action, 100, "操作已完成，请使用新文字跨设备复测", "done")
		}
	}()
	switch action {
	case "quick":
		if _, err = e.command(ctx, "/usr/bin/defaults", "write", activityDomain, "ClipboardSharingEnabled", "-bool", "true"); err != nil {
			return err
		}
		e.progress(action, 25, "已启用剪贴板共享", "running")
		for i, n := range []string{"useractivityd", "pboard"} {
			if err = e.restart(ctx, n); err != nil {
				return err
			}
			e.progress(action, 45+i*30, "已刷新 "+n, "running")
		}
	case "discovery", "identity":
		names := []string{"rapportd", "sharingd"}
		if action == "identity" {
			names = []string{"identityservicesd", "pboard"}
		}
		for i, n := range names {
			if err = e.restart(ctx, n); err != nil {
				return err
			}
			e.progress(action, 35+i*40, "已刷新 "+n, "running")
		}
	case "handoff":
		defer func() {
			c, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			for _, key := range []string{"ActivityAdvertisingAllowed", "ActivityReceivingAllowed"} {
				if _, restoreErr := e.command(c, "/usr/bin/defaults", "-currentHost", "write", activityDomain, key, "-bool", "true"); restoreErr != nil {
					err = errors.Join(err, restoreErr)
				}
			}
		}()
		for _, key := range []string{"ActivityAdvertisingAllowed", "ActivityReceivingAllowed"} {
			if _, err = e.command(ctx, "/usr/bin/defaults", "-currentHost", "write", activityDomain, key, "-bool", "false"); err != nil {
				return err
			}
		}
		e.progress(action, 35, "已暂时关闭接力", "running")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
		for _, key := range []string{"ActivityAdvertisingAllowed", "ActivityReceivingAllowed"} {
			if _, err = e.command(ctx, "/usr/bin/defaults", "-currentHost", "write", activityDomain, key, "-bool", "true"); err != nil {
				return err
			}
		}
		if err = e.restart(ctx, "useractivityd"); err != nil {
			return err
		}
		e.progress(action, 85, "接力已重新启用", "running")
	case "cache-send", "cache-receive":
		kind := "send"
		if action == "cache-receive" {
			kind = "receive"
		}
		if _, err = e.BackupCache(ctx, kind); err != nil {
			return err
		}
		e.progress(action, 70, "旧缓存已保留为可恢复备份", "running")
		if err = e.restart(ctx, "useractivityd"); err != nil {
			return err
		}
		e.progress(action, 90, "接力服务已重新载入；新缓存将在后续共享时生成", "running")
	}
	return nil
}
func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func (e *Engine) CreateProbe(ctx context.Context) (Probe, error) {
	if e.readOnly {
		return Probe{}, errReadOnly
	}
	if !e.op.TryLock() {
		return Probe{}, errors.New("请等待当前修复完成")
	}
	defer e.op.Unlock()
	p := Probe{Text: "CLIP-" + strings.ToUpper(newID()[:8]), Created: time.Now().Format(time.RFC3339)}
	if _, err := e.Runner.Run(ctx, "/usr/bin/pbcopy", nil, p.Text); err != nil {
		return Probe{}, err
	}
	time.Sleep(200 * time.Millisecond)
	s, err := e.command(ctx, "/usr/bin/pbpaste")
	if err != nil {
		return Probe{}, err
	}
	p.Matches = s == p.Text
	if !p.Matches {
		return p, errors.New("测试文字已被其他内容替换，请重新生成")
	}
	e.mu.Lock()
	e.probe = p
	e.verifications = []Verification{}
	e.mu.Unlock()
	e.log("info", "已写入并核对新的测试文字；等待跨设备粘贴确认")
	return p, nil
}
func (e *Engine) CheckProbe(ctx context.Context) (Probe, error) {
	e.mu.Lock()
	p := e.probe
	e.mu.Unlock()
	if p.Text == "" {
		return p, nil
	}
	s, err := e.command(ctx, "/usr/bin/pbpaste")
	p.Matches = err == nil && s == p.Text
	return p, err
}
func (e *Engine) RecordVerification(direction string, success bool) error {
	if e.readOnly {
		return errReadOnly
	}
	if direction != "mac-iphone" && direction != "mac-ipad" && direction != "iphone-mac" {
		return errors.New("不支持的复测方向")
	}
	if success && direction != "iphone-mac" {
		p, err := e.CheckProbe(context.Background())
		if err != nil {
			return err
		}
		if p.Text == "" {
			return errors.New("请先生成测试文字")
		}
		if !p.Matches {
			return errors.New("Mac 测试文字已被替换，请重新生成后复测")
		}
	}
	e.mu.Lock()
	if direction != "iphone-mac" && e.probe.Text == "" {
		e.mu.Unlock()
		return errors.New("请先生成测试文字")
	}
	v := Verification{direction, success, time.Now().Format(time.RFC3339)}
	for i, x := range e.verifications {
		if x.Direction == direction {
			e.verifications = append(e.verifications[:i], e.verifications[i+1:]...)
			break
		}
	}
	e.verifications = append(e.verifications, v)
	e.mu.Unlock()
	status := "成功"
	if !success {
		status = "未成功"
	}
	e.log("info", "用户确认跨设备复测 "+direction+"："+status)
	return nil
}
func (e *Engine) Tool(ctx context.Context, name string, open bool) error {
	if e.readOnly {
		return errReadOnly
	}
	bundle, ok := toolBundles[name]
	if !ok {
		return errors.New("不支持的剪贴板工具")
	}
	if !e.op.TryLock() {
		return errors.New("已有操作正在进行")
	}
	defer e.op.Unlock()
	if open {
		_, err := e.command(ctx, "/usr/bin/open", "-g", "-b", bundle)
		return err
	}
	pid := e.pid(ctx, name)
	if pid == 0 {
		return nil
	}
	_, err := e.command(ctx, "/bin/kill", "-TERM", strconv.Itoa(pid))
	if err == nil {
		e.log("info", "已暂时退出 "+name)
	}
	return err
}
func (e *Engine) OpenSettings(ctx context.Context, section string) error {
	if e.readOnly {
		return errReadOnly
	}
	links := map[string]string{"handoff": "x-apple.systempreferences:com.apple.AirDrop-Handoff-Settings.extension", "bluetooth": "x-apple.systempreferences:com.apple.BluetoothSettings", "wifi": "x-apple.systempreferences:com.apple.wifi-settings-extension", "automation": "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation"}
	u, ok := links[section]
	if !ok {
		return errors.New("不支持的设置页面")
	}
	_, err := e.command(ctx, "/usr/bin/open", u)
	return err
}
func (e *Engine) Export() ([]byte, error) {
	e.mu.Lock()
	r := e.lastReport
	e.mu.Unlock()
	b, err := json.MarshalIndent(struct {
		Report  Report  `json:"report"`
		Session Session `json:"session"`
	}{r, e.Session()}, "", "  ")
	if err != nil {
		return nil, err
	}
	homeJSON, err := json.Marshal(e.Home)
	if err != nil {
		return nil, err
	}
	return []byte(strings.ReplaceAll(string(b), string(homeJSON[1:len(homeJSON)-1]), "~")), nil
}
