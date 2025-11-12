package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sql-studio/backend-go/pkg/updater"
	"github.com/sql-studio/backend-go/pkg/version"
)

// Build-time variables (set via ldflags)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func main() {
	// Set version info in package
	version.Version = Version
	version.Commit = Commit
	version.BuildDate = BuildDate

	// Parse command
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "version", "--version", "-v":
		handleVersion()
	case "update":
		handleUpdate(os.Args[2:])
	case "serve", "server":
		// Run the server (not implemented in this file - delegates to cmd/server)
		fmt.Println("Server command should be run via cmd/server/main.go")
		fmt.Println("Use: make dev")
		os.Exit(1)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleVersion() {
	info := version.GetInfo()
	fmt.Println(info.String())
}

func handleUpdate(args []string) {
	// Parse flags
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	checkOnly := fs.Bool("check", false, "Only check for updates without installing")
	force := fs.Bool("force", false, "Force update even if already up to date")
	if err := fs.Parse(args); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Get config directory
	configDir, err := getConfigDir()
	if err != nil {
		fmt.Printf("Error: Failed to get config directory: %v\n", err)
		return
	}

	// Create updater
	u := updater.NewUpdater(configDir)

	fmt.Println("Checking for updates...")
	fmt.Printf("Current version: %s\n\n", version.Version)

	// Check for updates
	updateInfo, err := u.CheckForUpdate(ctx)
	if err != nil {
		fmt.Printf("Error: Failed to check for updates: %v\n", err)
		fmt.Println("\nPossible causes:")
		fmt.Println("  - No internet connection")
		fmt.Println("  - GitHub API rate limit exceeded")
		fmt.Println("  - Network firewall blocking requests")
		return
	}

	if !updateInfo.Available {
		fmt.Printf("You are already running the latest version (%s)\n", version.Version)
		return
	}

	fmt.Printf("New version available: %s\n", updateInfo.LatestVersion)
	fmt.Printf("Current version: %s\n\n", updateInfo.CurrentVersion)

	if updateInfo.ReleaseNotes != "" {
		fmt.Println("Release Notes:")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Println(updateInfo.ReleaseNotes)
		fmt.Println(strings.Repeat("-", 60))
		fmt.Println()
	}

	if *checkOnly {
		fmt.Println("To install the update, run: sqlstudio update")
		return
	}

	// Prompt for confirmation
	if !*force {
		fmt.Print("Do you want to install this update? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			return
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Update canceled.")
			return
		}
	}

	// Download and install update
	fmt.Println("\nDownloading update...")
	if err := u.DownloadUpdate(ctx, updateInfo); err != nil {
		fmt.Printf("Error: Failed to install update: %v\n", err)
		fmt.Println("\nPossible causes:")
		fmt.Println("  - Insufficient permissions (try running with sudo)")
		fmt.Println("  - Binary is currently running (stop it first)")
		fmt.Println("  - Installation method doesn't support auto-update (e.g., brew)")
		fmt.Println("\nManual update:")
		fmt.Printf("  Download from: %s\n", updateInfo.DownloadURL)
		return
	}

	fmt.Println("\n✓ Update installed successfully!")
	fmt.Printf("Updated from %s to %s\n", updateInfo.CurrentVersion, updateInfo.LatestVersion)
	fmt.Println("\nPlease restart the application for the changes to take effect.")
}

func getConfigDir() (string, error) {
	// Try to get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".sqlstudio")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", err
	}

	return configDir, nil
}

func printUsage() {
	fmt.Println("SQL Studio - Modern database management tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  sqlstudio <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  version              Show version information")
	fmt.Println("  update               Check for and install updates")
	fmt.Println("  help                 Show this help message")
	fmt.Println()
	fmt.Println("Update Options:")
	fmt.Println("  --check              Only check for updates without installing")
	fmt.Println("  --force              Force update without confirmation")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  sqlstudio version")
	fmt.Println("  sqlstudio update --check")
	fmt.Println("  sqlstudio update")
	fmt.Println()
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("For more information, visit: https://github.com/sql-studio/sql-studio\n")
}
