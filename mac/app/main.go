package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "K9 Web Protection",
		Width:     900,
		Height:    640,
		MinWidth:  800,
		MinHeight: 560,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 10, G: 22, B: 40, A: 255},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		OnBeforeClose: func(ctx context.Context) bool {
			wailsruntime.WindowMinimise(ctx)
			return true
		},
		Bind:             []interface{}{app},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  true,
				HideTitleBar:               false,
				FullSizeContent:            true,
				UseToolbar:                 false,
			},
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.k9webprotection.app",
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
