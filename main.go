package main

import (
	"context"
	"embed"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/nbb2025/macos-clipboard-fix/internal/core"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 && os.Args[1] == "--diagnose-json" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err = json.NewEncoder(os.Stdout).Encode(core.NewReadOnly(home).Diagnose(ctx)); err != nil {
			log.Fatal(err)
		}
		return
	}
	engine, err := core.New(home)
	if err != nil {
		log.Fatal(err)
	}
	service := &ClipboardService{engine: engine}
	app := application.New(application.Options{
		Icon: appIcon, SingleInstance: singleInstance(),
		Name: "Clipboard Fix", Description: "Mac 通用剪贴板诊断与修复",
		Services: []application.Service{application.NewService(service)},
		Assets:   application.AssetOptions{Handler: application.BundledAssetFileServer(assets)},
		Mac:      application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
		Server:   application.ServerOptions{Host: "127.0.0.1", Port: 9246},
	})
	engine.Emit = func(p core.Progress) { app.Event.Emit("clipboard:progress", p) }
	menu := app.NewMenu()
	menu.AddRole(application.AppMenu)
	menu.AddRole(application.EditMenu)
	menu.AddRole(application.WindowMenu)
	app.Menu.Set(menu)
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "main", Title: "Clipboard Fix", Width: 1160, Height: 800, MinWidth: 960, MinHeight: 680,
		Mac:              application.MacWindow{InvisibleTitleBarHeight: 44, TitleBar: application.MacTitleBarHiddenInset},
		BackgroundColour: application.NewRGB(247, 249, 247), URL: "/",
	})
	if err = app.Run(); err != nil {
		log.Fatal(err)
	}
}
