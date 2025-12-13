package download

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ExecuteLauncher executes the Minecraft launcher
func ExecuteLauncher() error {
	// Use the executable path found during download
	launcherPath := GetExecutablePath()

	if launcherPath == "" {
		return fmt.Errorf("no executable found during download")
	}

	// Check if file exists
	if _, err := os.Stat(launcherPath); err != nil {
		return fmt.Errorf("launcher executable not found at %s: %w", launcherPath, err)
	}

	return executeLauncher(launcherPath)
}

func executeLauncher(launcherPath string) error {
	var cmd *exec.Cmd

	path, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
	}

	var args []string = []string{
		"--workDir",
		filepath.Join(path, "Minecraft", "minecraft"),
		"--tmpDir",
		filepath.Join(path, "Minecraft", "tmp"),
		"--user-data-dir",
		filepath.Join(path, "Minecraft", "data user"),
	}

	switch runtime.GOOS {
	case "darwin":
		if len(args) > 0 {
			openArgs := append([]string{launcherPath, "--args"}, args...)
			cmd = exec.Command("open", openArgs...)
		} else {
			cmd = exec.Command("open", launcherPath)
		}
	default:
		cmd = exec.Command(launcherPath, args...)
	}

	return cmd.Start()
}
