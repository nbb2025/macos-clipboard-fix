//go:build !server

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func singleInstance() *application.SingleInstanceOptions {
	return &application.SingleInstanceOptions{
		UniqueID: "com.nbb2025.clipboardfix",
		OnSecondInstanceLaunch: func(_ application.SecondInstanceData) {
			if window, ok := application.Get().Window.GetByName("main"); ok {
				window.Show()
				window.Focus()
			}
		},
	}
}
