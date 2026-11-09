package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/nbb2025/macos-clipboard-fix/internal/core"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type ClipboardService struct {
	engine *core.Engine
	ctx    context.Context
}

func (s *ClipboardService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx
	return nil
}
func (s *ClipboardService) Diagnose() core.Report            { return s.engine.Diagnose(s.ctx) }
func (s *ClipboardService) GetSession() core.Session         { return s.engine.Session() }
func (s *ClipboardService) Repair(action string) error       { return s.engine.Repair(s.ctx, action) }
func (s *ClipboardService) RestoreBackup(id string) error    { return s.engine.RestoreBackup(s.ctx, id) }
func (s *ClipboardService) CreateProbe() (core.Probe, error) { return s.engine.CreateProbe(s.ctx) }
func (s *ClipboardService) CheckProbe() (core.Probe, error)  { return s.engine.CheckProbe(s.ctx) }
func (s *ClipboardService) RecordVerification(direction string, success bool) error {
	return s.engine.RecordVerification(direction, success)
}
func (s *ClipboardService) CollectSignals() ([]core.Signal, error) {
	return s.engine.CollectSignals(s.ctx)
}
func (s *ClipboardService) SetToolRunning(name string, running bool) error {
	return s.engine.Tool(s.ctx, name, running)
}
func (s *ClipboardService) OpenSettings(section string) error {
	return s.engine.OpenSettings(s.ctx, section)
}

func (s *ClipboardService) ExportReport() (string, error) {
	path, err := application.Get().Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{Title: "导出诊断报告"}).SetFilename("Clipboard-Fix-"+time.Now().Format("20060102-150405")+".json").AddFilter("JSON 诊断报告", "*.json").PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	data, err := s.engine.Export()
	if err != nil {
		return "", err
	}
	// A native Save dialog obtains the user's chosen destination; do not accept
	// arbitrary write paths from the webview RPC surface.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	return path, errors.Join(writeErr, closeErr)
}
