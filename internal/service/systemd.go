//go:build linux

package service

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/user/claude-langfuse-go/internal/config"
)

func (i *Installer) getSystemdServicePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", i.serviceName+".service")
}

func (i *Installer) installLinux() error {
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen)
	gray := color.New(color.FgHiBlack)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)

	cyan.Println("Installing Claude Langfuse Monitor as System Service")
	cyan.Println(string(make([]byte, 60)))

	// Check configuration
	configFile := config.DefaultConfigFile()
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		red.Println("[ERR] Configuration not found")
		yellow.Println("   Run: claude-langfuse config --public-key <key> --secret-key <key>")
		return fmt.Errorf("configuration not found")
	}

	// Create systemd user directory
	home, _ := os.UserHomeDir()
	systemdDir := filepath.Join(home, ".config", "systemd", "user")
	if err := ensureDir(systemdDir); err != nil {
		return fmt.Errorf("failed to create systemd directory: %w", err)
	}

	// Create log directory
	logDir := i.GetLogDir()
	if err := ensureDir(logDir); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Get executable path
	execPath, err := executablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Generate service content
	serviceContent := i.generateSystemdService(execPath, logDir)

	// Write service file
	servicePath := i.getSystemdServicePath()
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	green.Println("[OK] Service configuration created")
	gray.Printf("   %s\n", servicePath)

	// Reload systemd
	if err := runCommand("systemctl", "--user", "daemon-reload"); err != nil {
		red.Println("[ERR] Failed to reload systemd")
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable service
	if err := runCommand("systemctl", "--user", "enable", i.serviceName+".service"); err != nil {
		red.Println("[ERR] Failed to enable service")
		return fmt.Errorf("failed to enable service: %w", err)
	}

	// Start service
	if err := runCommand("systemctl", "--user", "start", i.serviceName+".service"); err != nil {
		red.Println("[ERR] Failed to start service")
		yellow.Println("\n[WARN] You may need to enable lingering for user services:")
		currentUser, _ := user.Current()
		gray.Printf("   sudo loginctl enable-linger %s\n", currentUser.Username)
		return fmt.Errorf("failed to start service: %w", err)
	}

	green.Println("[OK] Service enabled and started")

	// Show status
	cyan.Println("\nService Status:")
	gray.Println("   The monitor will now start automatically on login")
	gray.Printf("   Logs: %s\n", i.GetLogFile())

	cyan.Println("\nUseful Commands:")
	gray.Printf("   View logs:    tail -f %s\n", i.GetLogFile())
	gray.Printf("   Status:       systemctl --user status %s.service\n", i.serviceName)
	gray.Printf("   Stop service: systemctl --user stop %s.service\n", i.serviceName)
	gray.Println("   Uninstall:    claude-langfuse uninstall-service")

	return nil
}

func (i *Installer) uninstallLinux() error {
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen)
	gray := color.New(color.FgHiBlack)
	yellow := color.New(color.FgYellow)

	cyan.Println("Uninstalling Claude Langfuse Monitor Service")
	cyan.Println(string(make([]byte, 60)))

	servicePath := i.getSystemdServicePath()

	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		yellow.Println("[WARN] Service not installed")
		return nil
	}

	// Stop service
	if err := runCommand("systemctl", "--user", "stop", i.serviceName+".service"); err != nil {
		yellow.Println("[WARN] Service was not running")
	} else {
		green.Println("[OK] Service stopped")
	}

	// Disable service
	runCommandIgnoreError("systemctl", "--user", "disable", i.serviceName+".service")
	green.Println("[OK] Service disabled")

	// Remove service file
	if err := os.Remove(servicePath); err != nil {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// Reload systemd
	runCommand("systemctl", "--user", "daemon-reload")

	green.Println("[OK] Service removed")

	gray.Println("\nConfiguration and logs are preserved")
	gray.Println("   To remove completely:")
	gray.Println("   rm -rf ~/.claude-langfuse")
	gray.Printf("   rm -rf %s\n", i.GetLogDir())

	return nil
}

func (i *Installer) generateSystemdService(execPath, logDir string) string {
	home, _ := os.UserHomeDir()
	logFile := filepath.Join(logDir, i.serviceName+".log")
	errorLogFile := filepath.Join(logDir, i.serviceName+"-error.log")

	return fmt.Sprintf(`[Unit]
Description=Claude Langfuse Monitor - Automatic observability for Claude Code
After=network.target

[Service]
Type=simple
ExecStart=%s start
Restart=on-failure
RestartSec=60
WorkingDirectory=%s
Environment=PATH=/usr/local/bin:/usr/bin:/bin
StandardOutput=append:%s
StandardError=append:%s

[Install]
WantedBy=default.target
`, execPath, home, logFile, errorLogFile)
}
