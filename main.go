package main

import (
	"embed"
	"fmt"
	"os"

	"zimap/internal/cli"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if len(os.Args) > 1 {
		subcommand := os.Args[1]
		if subcommand == "download" || subcommand == "convert" || subcommand == "update" {
			err := cli.ProcessCommand(os.Args[1:])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error processing command\n%v\n", err)
				os.Exit(1)
			}
			return
		}
		if subcommand == "--help" || subcommand == "-h" || subcommand == "help" {
			fmt.Println("ZiMap Downloader - OSM to GPS .img format converter and GUI")
			fmt.Println("\nUsage:")
			fmt.Println("  zimap [command]")
			fmt.Println("\nAvailable Commands:")
			fmt.Println("  download   Download map region from Geofabrik")
			fmt.Println("  convert    Convert OSM PBF file to GPS .img format")
			fmt.Println("  update     Check and update required tools (mkgmap, splitter, java)")
			fmt.Println("\nRun without arguments to launch the Apple HIG Desktop GUI.")
			return
		}
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "ZiMap Downloader",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			BackdropType:         windows.Mica,
		},
	})
	if err != nil {
		fmt.Println("Error:", err.Error())
	}
}
