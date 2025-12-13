package main

import (
	"fmt"
	"minecraft/config"
	"minecraft/download"
	"runtime"
)

func main() {
	var launcherURL string

	switch runtime.GOOS {
	case "darwin":
		launcherURL = config.Darwin
	case "windows":
		launcherURL = config.Windows
	case "linux":
		launcherURL = config.Linux
	default:
		fmt.Printf("Unsupported OS: %s\n", runtime.GOOS)
		return
	}

	// Download the launcher
	if err := download.DownloadLauncher(launcherURL); err != nil {
		fmt.Printf("Error downloading launcher: %v\n", err)
		return
	}

	// Execute the launcher
	if err := download.ExecuteLauncher(); err != nil {
		fmt.Printf("Error executing launcher: %v\n", err)
	}
}
