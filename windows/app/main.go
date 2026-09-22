//go:build windows

package main

import (
	"context"
	"embed"
	"sync/atomic"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"k10webprotection/internal/config"
	"k10webprotection/internal/i18n"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Load() is the single place that sets the language; the title below needs it.
	opts := loadOptions()
	cfg, loadErr := config.LoadWith(opts)

	app := NewApp(cfg, loadErr, opts)

	err := wails.Run(&options.App{
		Title:         i18n.T("app.title"),
		Width:         960,
		Height:        660,
		MinWidth:      960,
		MinHeight:     660,
		MaxWidth:      960,
		MaxHeight:     660,
		DisableResize: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 10, G: 22, B: 40, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose: func(ctx context.Context) bool {
			// Allow immediate quit during Windows shutdown/logoff,
			// or when the user has already confirmed with their password.
			if systemShuttingDown() || atomic.LoadInt32(&app.quitAuth) == 1 {
				return false
			}
			// X button minimizes to tray; only the tray "Exit" item can trigger the quit modal.
			wailsruntime.WindowHide(ctx)
			return true
		},
		Bind: []interface{}{app},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			Theme:                windows.Dark,
			CustomTheme: &windows.ThemeSettings{
				DarkModeTitleBar:   windows.RGB(13, 50, 96),
				DarkModeTitleText:  windows.RGB(255, 255, 255),
				DarkModeBorder:     windows.RGB(20, 73, 133),
				LightModeTitleBar:  windows.RGB(13, 50, 96),
				LightModeTitleText: windows.RGB(255, 255, 255),
				LightModeBorder:    windows.RGB(20, 73, 133),
			},
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.k10webprotection.app",
			OnSecondInstanceLaunch: func(data options.SecondInstanceData) {
				wailsruntime.WindowShow(app.ctx)
				wailsruntime.WindowUnminimise(app.ctx)
			},
		},
	})
	if err != nil {
		panic(err)
	}
}
