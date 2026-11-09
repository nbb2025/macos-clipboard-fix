package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

// The controller is read-only. Network comparisons remain explicit, guided
// actions in the user's proxy app, so this utility never changes their routes.
func controllerRead(ctx context.Context, path string, dst any) error {
	socket := fmt.Sprintf("/var/run/clash-verge-service/users/%d/verge-mihomo.sock", os.Getuid())
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost"+path, nil)
	if err != nil {
		return err
	}
	r, err := client.Do(req)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("本地控制器 HTTP %d", r.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(r.Body, 256*1024)).Decode(dst)
}
func (e *Engine) ProxyStatus(ctx context.Context) ProxyState {
	var config struct {
		Mode string `json:"mode"`
		Tun  struct {
			Enable bool   `json:"enable"`
			Stack  string `json:"stack"`
		} `json:"tun"`
	}
	if controllerRead(ctx, "/configs", &config) != nil {
		return ProxyState{}
	}
	r := ProxyState{Available: true, TUN: config.Tun.Enable, Mode: config.Mode, Stack: config.Tun.Stack}
	var apple struct {
		Now string `json:"now"`
	}
	if controllerRead(ctx, "/proxies/"+url.PathEscape("🍎 苹果服务"), &apple) == nil {
		r.AppleRoute = apple.Now
	}
	return r
}
