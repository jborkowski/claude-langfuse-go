//go:build darwin

package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/user/claude-langfuse-go/internal/config"
)

func (i *Installer) getPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", i.serviceName+".plist")
}

func (i *Installer) installMacOS() error {
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

	// Create LaunchAgents directory
	home, _ := os.UserHomeDir()
	launchAgentsDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := ensureDir(launchAgentsDir); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
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

	// Generate plist content
	plistContent := i.generatePlist(execPath, logDir)

	// Write plist
	plistPath := i.getPlistPath()
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist: %w", err)
	}

	green.Println("[OK] Service configuration created")
	gray.Printf("   %s\n", plistPath)

	// Unload existing service (ignore errors)
	runCommandIgnoreError("launchctl", "unload", plistPath)

	// Load service
	if err := runCommand("launchctl", "load", plistPath); err != nil {
		red.Println("[ERR] Failed to load service")
		return fmt.Errorf("failed to load service: %w", err)
	}

	green.Println("[OK] Service loaded and started")

	// Show status
	cyan.Println("\nService Status:")
	gray.Println("   The monitor will now start automatically on login")
	gray.Printf("   Logs: %s\n", i.GetLogFile())

	cyan.Println("\nUseful Commands:")
	gray.Printf("   View logs:    tail -f %s\n", i.GetLogFile())
	gray.Printf("   Stop service: launchctl stop %s\n", i.serviceName)
	gray.Println("   Uninstall:    claude-langfuse uninstall-service")

	return nil
}

func (i *Installer) uninstallMacOS() error {
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen)
	gray := color.New(color.FgHiBlack)
	yellow := color.New(color.FgYellow)

	cyan.Println("Uninstalling Claude Langfuse Monitor Service")
	cyan.Println(string(make([]byte, 60)))

	plistPath := i.getPlistPath()

	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		yellow.Println("[WARN] Service not installed")
		return nil
	}

	// Unload service
	if err := runCommand("launchctl", "unload", plistPath); err != nil {
		yellow.Println("[WARN] Service was not running")
	} else {
		green.Println("[OK] Service stopped")
	}

	// Remove plist
	if err := os.Remove(plistPath); err != nil {
		return fmt.Errorf("failed to remove plist: %w", err)
	}

	green.Println("[OK] Service removed")

	gray.Println("\nConfiguration and logs are preserved")
	gray.Println("   To remove completely:")
	gray.Println("   rm -rf ~/.claude-langfuse")
	gray.Printf("   rm %s*.log\n", filepath.Join(i.GetLogDir(), i.serviceName))

	return nil
}

func (i *Installer) generatePlist(execPath, logDir string) string {
	home, _ := os.UserHomeDir()
	logFile := filepath.Join(logDir, i.serviceName+".log")
	errorLogFile := filepath.Join(logDir, i.serviceName+"-error.log")

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>

    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>start</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>

    <key>StandardOutPath</key>
    <string>%s</string>

    <key>StandardErrorPath</key>
    <string>%s</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin</string>
    </dict>

    <key>WorkingDirectory</key>
    <string>%s</string>

    <key>ThrottleInterval</key>
    <integer>60</integer>
</dict>
</plist>`, i.serviceName, execPath, logFile, errorLogFile, home)
}
