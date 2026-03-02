package main

import (
	_ "embed"
	"log"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// assets is defined in embed.go
// iconFS is defined in app.go

// init registers custom events for strong typing in the binding generator
func init() {
	// Register events that will be emitted to the frontend
	// These provide strongly typed JS/TS APIs
	application.RegisterEvent[interface{}]("app:startup-complete")
	application.RegisterEvent[interface{}]("app:shutdown")
	application.RegisterEvent[string]("auth:error")
	application.RegisterEvent[map[string]interface{}]("auth:success")
	application.RegisterEvent[map[string]interface{}]("webauthn:success")
}

func main() {
	// Create the lifecycle coordinator that manages all 11 services
	lifecycle := NewAppLifecycle()

	// Load the application icon from embedded assets
	iconBytes, _ := iconFS.ReadFile("assets/howlerops-transparent.png")

	// Create a new Wails v3 application with all services registered
	app := application.New(application.Options{
		Name:        "HowlerOps",
		Description: "A powerful desktop SQL client",
		Icon:        iconBytes,
		Services:    lifecycle.GetServices(),
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Store the application reference for event emission across services
	lifecycle.SetApplication(app)

	// Create the main window
	mainWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "HowlerOps",
		Width:     1200,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 59),
		URL:              "/",
	})

	// Store the main window reference for dialogs across services
	lifecycle.SetMainWindow(mainWindow)

	// Workaround for Wails v3 dev mode: the webview may open before the page
	// is ready, resulting in a blank screen. Track successful navigation and
	// retry once if it hasn't completed after a short delay.
	var navDone atomic.Bool
	if runtime.GOOS == "darwin" {
		mainWindow.OnWindowEvent(events.Mac.WebViewDidFinishNavigation, func(_ *application.WindowEvent) {
			navDone.Store(true)
		})
	}
	go func() {
		time.Sleep(3 * time.Second)
		if !navDone.Load() {
			mainWindow.Reload()
		}
	}()

	// Set up menus (v3 menu API)
	setupMenu(app, mainWindow)

	// Handle application lifecycle via v3 Event manager
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(event *application.ApplicationEvent) {
		lifecycle.OnStartup()
	})

	// Use app.OnShutdown for cleanup across all services
	app.OnShutdown(func() {
		lifecycle.OnShutdown()
	})

	// Handle URL open events (for OAuth callbacks on macOS)
	if runtime.GOOS == "darwin" {
		app.Event.OnApplicationEvent(events.Common.ApplicationLaunchedWithUrl, func(event *application.ApplicationEvent) {
			if url := event.Context().URL(); url != "" {
				lifecycle.OnUrlOpen(url)
			}
		})
	}

	// Run the application
	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}

// setupMenu creates the application menu using v3 API
func setupMenu(app *application.App, window application.Window) {
	// Create the main menu
	menu := app.Menu.New()

	// macOS requires the AppMenu (About, Services, Hide, Quit) as the first menu
	if runtime.GOOS == "darwin" {
		menu.AddRole(application.AppMenu)
	}

	// File menu
	fileMenu := menu.AddSubmenu("File")
	fileMenu.Add("New Query").SetAccelerator("CmdOrCtrl+N")
	fileMenu.Add("Open File...").SetAccelerator("CmdOrCtrl+O")
	fileMenu.Add("Save").SetAccelerator("CmdOrCtrl+S")
	fileMenu.Add("Save As...").SetAccelerator("CmdOrCtrl+Shift+S")
	fileMenu.AddSeparator()
	fileMenu.Add("Close Tab").SetAccelerator("CmdOrCtrl+W")
	if runtime.GOOS != "darwin" {
		// On macOS, Quit lives in the AppMenu; on other platforms add it to File
		fileMenu.Add("Quit").SetAccelerator("CmdOrCtrl+Q").OnClick(func(ctx *application.Context) {
			app.Quit()
		})
	}

	// Edit menu — use role-based items so native macOS selectors (cut:, copy:, paste:, etc.) bind correctly
	editMenu := menu.AddSubmenu("Edit")
	editMenu.AddRole(application.Undo)
	editMenu.AddRole(application.Redo)
	editMenu.AddSeparator()
	editMenu.AddRole(application.Cut)
	editMenu.AddRole(application.Copy)
	editMenu.AddRole(application.Paste)
	if runtime.GOOS == "darwin" {
		editMenu.AddRole(application.PasteAndMatchStyle)
		editMenu.AddRole(application.Delete)
		editMenu.AddRole(application.SelectAll)
	} else {
		editMenu.AddRole(application.Delete)
		editMenu.AddSeparator()
		editMenu.AddRole(application.SelectAll)
		editMenu.AddSeparator()
		editMenu.Add("Find").SetAccelerator("CmdOrCtrl+F")
		editMenu.Add("Replace").SetAccelerator("CmdOrCtrl+H")
	}

	// Query menu
	queryMenu := menu.AddSubmenu("Query")
	queryMenu.Add("Run Query").SetAccelerator("CmdOrCtrl+Return")
	queryMenu.Add("Run Selection").SetAccelerator("CmdOrCtrl+Shift+Return")
	queryMenu.Add("Explain Query").SetAccelerator("CmdOrCtrl+E")
	queryMenu.AddSeparator()
	queryMenu.Add("Format Query").SetAccelerator("CmdOrCtrl+Shift+F")

	// Connection menu
	connMenu := menu.AddSubmenu("Connection")
	connMenu.Add("New Connection").SetAccelerator("CmdOrCtrl+Shift+N")
	connMenu.Add("Test Connection").SetAccelerator("CmdOrCtrl+T")
	connMenu.Add("Refresh").SetAccelerator("CmdOrCtrl+R")

	// View menu
	viewMenu := menu.AddSubmenu("View")
	viewMenu.Add("Toggle Sidebar").SetAccelerator("CmdOrCtrl+B")
	viewMenu.Add("Toggle Results Panel").SetAccelerator("CmdOrCtrl+Shift+R")
	viewMenu.AddSeparator()
	viewMenu.Add("Zoom In").SetAccelerator("CmdOrCtrl+=")
	viewMenu.Add("Zoom Out").SetAccelerator("CmdOrCtrl+-")
	viewMenu.Add("Reset Zoom").SetAccelerator("CmdOrCtrl+0")

	// Window menu
	windowMenu := menu.AddSubmenu("Window")
	windowMenu.Add("Minimize").SetAccelerator("CmdOrCtrl+M")
	windowMenu.Add("Toggle Fullscreen").SetAccelerator("CmdOrCtrl+Ctrl+F")

	// Help menu
	helpMenu := menu.AddSubmenu("Help")
	helpMenu.Add("About HowlerOps")
	helpMenu.Add("Documentation")
	helpMenu.Add("Keyboard Shortcuts").SetAccelerator("CmdOrCtrl+?")

	// Set as application menu
	app.Menu.SetApplicationMenu(menu)
}
