// Claude Langfuse Monitor - Automatic Langfuse tracking for Claude Code
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
	"github.com/user/claude-langfuse-go/internal/config"
	"github.com/user/claude-langfuse-go/internal/monitor"
	"github.com/user/claude-langfuse-go/internal/service"
	"github.com/user/claude-langfuse-go/internal/watcher"
)

func main() {
	app := &cli.App{
		Name:    "claude-langfuse",
		Usage:   "Automatic Langfuse tracking for Claude Code activity",
		Version: "1.0.0",
		Commands: []*cli.Command{
			startCommand(),
			configCommand(),
			statusCommand(),
			installServiceCommand(),
			uninstallServiceCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}
}

func startCommand() *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "Start monitoring Claude Code activity",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "daemon",
				Aliases: []string{"d"},
				Usage:   "Run as background daemon",
			},
			&cli.IntFlag{
				Name:    "history",
				Aliases: []string{"H"},
				Value:   24,
				Usage:   "Process last N hours of history",
			},
			&cli.BoolFlag{
				Name:    "quiet",
				Aliases: []string{"q"},
				Usage:   "Quiet mode - only show summaries",
			},
		},
		Action: func(c *cli.Context) error {
			cyan := color.New(color.FgCyan)
			gray := color.New(color.FgHiBlack)
			yellow := color.New(color.FgYellow)
			green := color.New(color.FgGreen)

			cyan.Println("Claude Langfuse Monitor")
			cyan.Println(strings.Repeat("=", 50))

			// Create monitor
			opts := monitor.Options{
				HistoryHours: c.Int("history"),
				Daemon:       c.Bool("daemon"),
				DryRun:       false,
				Quiet:        c.Bool("quiet"),
			}

			mon, err := monitor.New(opts)
			if err != nil {
				return fmt.Errorf("failed to create monitor: %w", err)
			}

			// Get projects directory
			projectsDir, err := monitor.GetClaudeProjectsDir()
			if err != nil {
				return err
			}

			gray.Printf("Claude projects: %s\n", projectsDir)

			// Process existing history
			if opts.HistoryHours > 0 {
				if err := mon.ProcessExistingHistory(); err != nil {
					return fmt.Errorf("failed to process history: %w", err)
				}
				// Flush any pending events from history processing
				if err := mon.Flush(); err != nil {
					gray.Printf("Warning: failed to flush history: %v\n", err)
				}
			}

			// Start watching
			cyan.Println("\nWatching for new Claude Code activity...")
			gray.Printf("Langfuse UI: %s\n", mon.Config().Host)
			gray.Println("Press Ctrl+C to stop")

			// Create file watcher
			w, err := watcher.New(projectsDir, mon.ProcessConversationFile)
			if err != nil {
				return fmt.Errorf("failed to create watcher: %w", err)
			}

			if err := w.Start(); err != nil {
				return fmt.Errorf("failed to start watcher: %w", err)
			}

			// Start periodic flush ticker (every 5 seconds)
			flushTicker := time.NewTicker(5 * time.Second)
			done := make(chan struct{})
			go func() {
				for {
					select {
					case <-flushTicker.C:
						mon.Flush()
					case <-done:
						return
					}
				}
			}()

			// Handle graceful shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			<-sigChan
			close(done)
			flushTicker.Stop()

			yellow.Println("\n\nStopping monitor...")
			w.Close()
			mon.Shutdown()
			green.Println("Monitor stopped")

			return nil
		},
	}
}

func configCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Configure Langfuse connection and trace metadata",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "host",
				Usage: "Langfuse host URL",
			},
			&cli.StringFlag{
				Name:  "public-key",
				Usage: "Langfuse public key",
			},
			&cli.StringFlag{
				Name:  "secret-key",
				Usage: "Langfuse secret key",
			},
			&cli.StringFlag{
				Name:  "user-id",
				Usage: "User ID for traces (default: system username)",
			},
			&cli.StringFlag{
				Name:  "model",
				Usage: "Model name for generations (default: claude-code)",
			},
			&cli.StringFlag{
				Name:  "source",
				Usage: "Source identifier (default: claude_code_monitor)",
			},
			&cli.StringFlag{
				Name:  "user-trace-name",
				Usage: "Name for user message traces (default: claude_code_user)",
			},
			&cli.StringFlag{
				Name:  "assistant-trace-name",
				Usage: "Name for assistant response traces (default: claude_response)",
			},
			&cli.BoolFlag{
				Name:  "show",
				Usage: "Show current configuration",
			},
		},
		Action: func(c *cli.Context) error {
			cyan := color.New(color.FgCyan)
			gray := color.New(color.FgHiBlack)
			green := color.New(color.FgGreen)
			yellow := color.New(color.FgYellow)

			configFile := config.DefaultConfigFile()

			// Show current config
			if c.Bool("show") {
				cfg, err := config.LoadFromFile()
				if err != nil {
					if os.IsNotExist(err) {
						yellow.Println("No configuration file found")
						gray.Println("   Using environment variables or defaults")
						return nil
					}
					return err
				}

				cyan.Println("Current configuration:")
				gray.Printf("   File: %s\n\n", configFile)

				if cfg.Host != "" {
					gray.Printf("   host: %s\n", cfg.Host)
				}
				if cfg.PublicKey != "" {
					gray.Println("   publicKey: ***")
				}
				if cfg.SecretKey != "" {
					gray.Println("   secretKey: ***")
				}
				if cfg.UserID != "" {
					gray.Printf("   userId: %s\n", cfg.UserID)
				}
				if cfg.Model != "" {
					gray.Printf("   model: %s\n", cfg.Model)
				}
				if cfg.Source != "" {
					gray.Printf("   source: %s\n", cfg.Source)
				}
				if cfg.UserTraceName != "" {
					gray.Printf("   userTraceName: %s\n", cfg.UserTraceName)
				}
				if cfg.AssistantTraceName != "" {
					gray.Printf("   assistantTraceName: %s\n", cfg.AssistantTraceName)
				}

				return nil
			}

			// Load existing config
			cfg, err := config.LoadFromFile()
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			if cfg == nil {
				cfg = &config.Config{}
			}

			// Update with provided values
			if v := c.String("host"); v != "" {
				cfg.Host = v
			}
			if v := c.String("public-key"); v != "" {
				cfg.PublicKey = v
			}
			if v := c.String("secret-key"); v != "" {
				cfg.SecretKey = v
			}
			if v := c.String("user-id"); v != "" {
				cfg.UserID = v
			}
			if v := c.String("model"); v != "" {
				cfg.Model = v
			}
			if v := c.String("source"); v != "" {
				cfg.Source = v
			}
			if v := c.String("user-trace-name"); v != "" {
				cfg.UserTraceName = v
			}
			if v := c.String("assistant-trace-name"); v != "" {
				cfg.AssistantTraceName = v
			}

			// Save config
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			green.Println("Configuration saved")
			gray.Printf("   Config file: %s\n", configFile)

			return nil
		},
	}
}

func statusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Check monitor status and connection",
		Action: func(c *cli.Context) error {
			cyan := color.New(color.FgCyan)
			gray := color.New(color.FgHiBlack)
			green := color.New(color.FgGreen)
			red := color.New(color.FgRed)
			yellow := color.New(color.FgYellow)

			cyan.Println("Claude Langfuse Monitor Status")
			cyan.Println(strings.Repeat("=", 50))

			// Check Claude directory
			projectsDir, err := monitor.GetClaudeProjectsDir()
			if err != nil {
				red.Println("[ERR] Claude projects directory not found")
				return nil
			}
			green.Println("[OK] Claude projects directory found")
			gray.Printf("   %s\n", projectsDir)

			// Check configuration
			cfg, err := config.Load()
			if err != nil {
				red.Println("[ERR] Failed to load configuration")
				return nil
			}

			if cfg.PublicKey != "" && cfg.SecretKey != "" {
				green.Println("[OK] Langfuse credentials configured")
			} else {
				red.Println("[ERR] Langfuse credentials not configured")
				yellow.Println("   Run: claude-langfuse config --public-key <key> --secret-key <key>")
				return nil
			}

			gray.Printf("   Host: %s\n", cfg.Host)

			green.Println("\n[OK] Monitor ready to run")
			gray.Println("   Start with: claude-langfuse start")

			return nil
		},
	}
}

func installServiceCommand() *cli.Command {
	return &cli.Command{
		Name:  "install-service",
		Usage: "Install as system service (launchd on macOS, systemd on Linux)",
		Action: func(c *cli.Context) error {
			installer := service.NewInstaller()
			return installer.Install()
		},
	}
}

func uninstallServiceCommand() *cli.Command {
	return &cli.Command{
		Name:  "uninstall-service",
		Usage: "Uninstall system service",
		Action: func(c *cli.Context) error {
			installer := service.NewInstaller()
			return installer.Uninstall()
		},
	}
}
