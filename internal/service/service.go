// Package service provides system service installation for macOS and Linux.
package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/user/claude-langfuse-go/internal/config"
)

// Installer provides service installation functionality.
type Installer struct {
	serviceName string
}

// NewInstaller creates a new service installer.
func NewInstaller() *Installer {
	return &Installer{
		serviceName: config.ServiceName(),
	}
}

// Install installs the system service.
func (i *Installer) Install() error {
	switch runtime.GOOS {
	case "darwin":
		return i.installMacOS()
	case "linux":
		return i.installLinux()
	default:
		return fmt.Errorf("unsupported platform: %s (only macOS and Linux are supported)", runtime.GOOS)
	}
}

// Uninstall removes the system service.
func (i *Installer) Uninstall() error {
	switch runtime.GOOS {
	case "darwin":
		return i.uninstallMacOS()
	case "linux":
		return i.uninstallLinux()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// GetLogDir returns the log directory for the current platform.
func (i *Installer) GetLogDir() string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Logs")
	}
	return filepath.Join(home, ".local", "share", i.serviceName, "logs")
}

// GetLogFile returns the log file path.
func (i *Installer) GetLogFile() string {
	return filepath.Join(i.GetLogDir(), i.serviceName+".log")
}

// executablePath returns the path to the current executable.
func executablePath() (string, error) {
	return os.Executable()
}

// ensureDir creates a directory if it doesn't exist.
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// runCommand runs a command and returns any error.
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// runCommandIgnoreError runs a command and ignores errors.
func runCommandIgnoreError(name string, args ...string) {
	cmd := exec.Command(name, args...)
	_ = cmd.Run()
}
