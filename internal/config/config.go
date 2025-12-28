// Package config handles configuration loading from file and environment variables.
package config

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
)

// Config holds all configuration for the monitor.
type Config struct {
	Host               string `json:"host"`
	PublicKey          string `json:"publicKey"`
	SecretKey          string `json:"secretKey"`
	UserID             string `json:"userId"`
	Model              string `json:"model"`
	Source             string `json:"source"`
	UserTraceName      string `json:"userTraceName"`
	AssistantTraceName string `json:"assistantTraceName"`
}

// DefaultConfigDir returns the default configuration directory.
func DefaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude-langfuse")
}

// DefaultConfigFile returns the default configuration file path.
func DefaultConfigFile() string {
	return filepath.Join(DefaultConfigDir(), "config.json")
}

// getEnvOrDefault returns the environment variable value or the default.
func getEnvOrDefault(envVar, defaultVal string) string {
	if val := os.Getenv(envVar); val != "" {
		return val
	}
	return defaultVal
}

// getCurrentUsername returns the current user's username.
func getCurrentUsername() string {
	u, err := user.Current()
	if err != nil {
		return "unknown"
	}
	return u.Username
}

// Load reads configuration from file and merges with environment variables.
// Environment variables take precedence over file configuration.
func Load() (*Config, error) {
	// Start with defaults
	cfg := &Config{
		Host:               "http://localhost:3001",
		PublicKey:          "",
		SecretKey:          "",
		UserID:             getCurrentUsername(),
		Model:              "claude-code",
		Source:             "claude_code_monitor",
		UserTraceName:      "claude_code_user",
		AssistantTraceName: "claude_response",
	}

	// Try to load from config file
	configFile := DefaultConfigFile()
	if data, err := os.ReadFile(configFile); err == nil {
		var fileCfg Config
		if err := json.Unmarshal(data, &fileCfg); err == nil {
			// Merge file config (only non-empty values)
			if fileCfg.Host != "" {
				cfg.Host = fileCfg.Host
			}
			if fileCfg.PublicKey != "" {
				cfg.PublicKey = fileCfg.PublicKey
			}
			if fileCfg.SecretKey != "" {
				cfg.SecretKey = fileCfg.SecretKey
			}
			if fileCfg.UserID != "" {
				cfg.UserID = fileCfg.UserID
			}
			if fileCfg.Model != "" {
				cfg.Model = fileCfg.Model
			}
			if fileCfg.Source != "" {
				cfg.Source = fileCfg.Source
			}
			if fileCfg.UserTraceName != "" {
				cfg.UserTraceName = fileCfg.UserTraceName
			}
			if fileCfg.AssistantTraceName != "" {
				cfg.AssistantTraceName = fileCfg.AssistantTraceName
			}
		}
	}

	// Environment variables take precedence
	cfg.Host = getEnvOrDefault("LANGFUSE_HOST", cfg.Host)
	cfg.PublicKey = getEnvOrDefault("LANGFUSE_PUBLIC_KEY", cfg.PublicKey)
	cfg.SecretKey = getEnvOrDefault("LANGFUSE_SECRET_KEY", cfg.SecretKey)
	cfg.UserID = getEnvOrDefault("CLAUDE_LANGFUSE_USER_ID", cfg.UserID)
	cfg.Model = getEnvOrDefault("CLAUDE_LANGFUSE_MODEL", cfg.Model)
	cfg.Source = getEnvOrDefault("CLAUDE_LANGFUSE_SOURCE", cfg.Source)
	cfg.UserTraceName = getEnvOrDefault("CLAUDE_LANGFUSE_USER_TRACE_NAME", cfg.UserTraceName)
	cfg.AssistantTraceName = getEnvOrDefault("CLAUDE_LANGFUSE_ASSISTANT_TRACE_NAME", cfg.AssistantTraceName)

	return cfg, nil
}

// Save writes the configuration to the config file.
func Save(cfg *Config) error {
	configDir := DefaultConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(DefaultConfigFile(), data, 0600)
}

// LoadFromFile loads existing config from file (for merging).
func LoadFromFile() (*Config, error) {
	configFile := DefaultConfigFile()
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// ServiceName returns the service name from environment or default.
func ServiceName() string {
	return getEnvOrDefault("CLAUDE_LANGFUSE_SERVICE_NAME", "claude-langfuse-monitor")
}
