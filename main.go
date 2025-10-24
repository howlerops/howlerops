package main

import (
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"runtime"
)

// assets is defined in embed.go

// buildMenu creates the application menu
// Note: Menu callbacks will receive their own context from Wails
func buildMenu() *menu.Menu {
	AppMenu := menu.NewMenu()

	// File menu
	FileMenu := AppMenu.AddSubmenu("File")
	FileMenu.AddText("New Query", keys.CmdOrCtrl("n"), func(cd *menu.CallbackData) {
		// TODO: Implement new query functionality
	})
	FileMenu.AddText("Open File...", keys.CmdOrCtrl("o"), func(cd *menu.CallbackData) {
		// TODO: Implement file open functionality
	})
	FileMenu.AddText("Save", keys.CmdOrCtrl("s"), func(cd *menu.CallbackData) {
		// TODO: Implement save functionality
	})
	FileMenu.AddText("Save As...", keys.Combo("s", keys.CmdOrCtrlKey, keys.ShiftKey), func(cd *menu.CallbackData) {
		// TODO: Implement save as functionality
	})
	FileMenu.AddSeparator()
	FileMenu.AddText("Close Tab", keys.CmdOrCtrl("w"), func(cd *menu.CallbackData) {
		// TODO: Implement close tab functionality
	})
	FileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(cd *menu.CallbackData) {
		// TODO: Implement proper quit with context
	})

	// Edit menu - Use built-in EditMenu for proper clipboard support
	if runtime.GOOS == "darwin" {
		// On macOS, use the built-in EditMenu for proper system integration
		AppMenu.Append(menu.EditMenu())
	} else {
		// For other platforms, create custom Edit menu
		EditMenu := AppMenu.AddSubmenu("Edit")
		EditMenu.AddText("Undo", keys.CmdOrCtrl("z"), nil)
		EditMenu.AddText("Redo", keys.CmdOrCtrl("y"), nil)
		EditMenu.AddSeparator()
		EditMenu.AddText("Cut", keys.CmdOrCtrl("x"), nil)
		EditMenu.AddText("Copy", keys.CmdOrCtrl("c"), nil)
		EditMenu.AddText("Paste", keys.CmdOrCtrl("v"), nil)
		EditMenu.AddText("Select All", keys.CmdOrCtrl("a"), nil)
		EditMenu.AddSeparator()
		EditMenu.AddText("Find", keys.CmdOrCtrl("f"), func(cd *menu.CallbackData) {
			// TODO: Implement find functionality
		})
		EditMenu.AddText("Replace", keys.CmdOrCtrl("h"), func(cd *menu.CallbackData) {
			// TODO: Implement replace functionality
		})
	}

	// Query menu
	QueryMenu := AppMenu.AddSubmenu("Query")
	QueryMenu.AddText("Run Query", keys.CmdOrCtrl("return"), func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:run-query")
	})
	QueryMenu.AddText("Run Selection", keys.Combo("return", keys.CmdOrCtrlKey, keys.ShiftKey), func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:run-selection")
	})
	QueryMenu.AddText("Explain Query", keys.CmdOrCtrl("e"), func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:explain-query")
	})
	QueryMenu.AddSeparator()
	QueryMenu.AddText("Format Query", keys.Combo("f", keys.CmdOrCtrlKey, keys.ShiftKey), func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:format-query")
	})

	// Connection menu
	ConnectionMenu := AppMenu.AddSubmenu("Connection")
	ConnectionMenu.AddText("New Connection", keys.Combo("n", keys.CmdOrCtrlKey, keys.ShiftKey), func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:new-connection")
	})
	ConnectionMenu.AddText("Test Connection", keys.CmdOrCtrl("t"), func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:test-connection")
	})
	ConnectionMenu.AddText("Refresh", keys.CmdOrCtrl("r"), func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:refresh")
	})

	// View menu
	ViewMenu := AppMenu.AddSubmenu("View")
	ViewMenu.AddText("Toggle Sidebar", keys.CmdOrCtrl("b"), func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:toggle-sidebar")
	})
	ViewMenu.AddText("Toggle Results Panel", keys.Combo("r", keys.CmdOrCtrlKey, keys.ShiftKey), func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:toggle-results")
	})
	ViewMenu.AddSeparator()
	ViewMenu.AddText("Zoom In", keys.CmdOrCtrl("="), func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:zoom-in")
	})
	ViewMenu.AddText("Zoom Out", keys.CmdOrCtrl("-"), func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:zoom-out")
	})
	ViewMenu.AddText("Reset Zoom", keys.CmdOrCtrl("0"), func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:reset-zoom")
	})

	// Window menu
	WindowMenu := AppMenu.AddSubmenu("Window")
	WindowMenu.AddText("Minimize", keys.CmdOrCtrl("m"), func(cd *menu.CallbackData) {
		// TODO: Implement window minimize
	})
	WindowMenu.AddText("Toggle Fullscreen", keys.Combo("f", keys.CmdOrCtrlKey, keys.ControlKey), func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:toggle-fullscreen")
	})

	// Help menu
	HelpMenu := AppMenu.AddSubmenu("Help")
	HelpMenu.AddText("About HowlerOps", nil, func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:about")
	})
	HelpMenu.AddText("Documentation", nil, func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:documentation")
	})
	HelpMenu.AddText("Keyboard Shortcuts", keys.CmdOrCtrl("?"), func(cd *menu.CallbackData) {
		// TODO: Implement menu functionality "menu:shortcuts")
	})

	return AppMenu
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Get the app icon for platform-specific configurations
	appIcon, err := app.GetAppIcon()
	if err != nil {
		println("Warning: Could not load app icon:", err.Error())
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:     "HowlerOps",
		Width:     1200,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:  &options.RGBA{R: 27, G: 38, B: 59, A: 1},
		OnStartup:         app.OnStartup,
		OnShutdown:        app.OnShutdown,
		Menu:              buildMenu(),
		Frameless:         false,
		DisableResize:     false,
		Fullscreen:        false,
		HideWindowOnClose: false,
		CSSDragProperty:   "--wails-draggable",
		CSSDragValue:      "drag",
		WindowStartState:  options.Normal,
		Bind: []interface{}{
			app,
		},
		EnumBind: []interface{}{
			// Add enums here if needed
		},
		// Platform-specific configurations
		Mac: &mac.Options{
			About: &mac.AboutInfo{
				Title:   "HowlerOps",
				Message: "© 2025 HowlerOps Team\nA powerful desktop SQL client",
				Icon:    appIcon,
			},
		},
		Linux: &linux.Options{
			Icon:        appIcon,
			ProgramName: "howlerops",
		},
		Windows: &windows.Options{
			// Windows will use the icon from wails.json configuration
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
