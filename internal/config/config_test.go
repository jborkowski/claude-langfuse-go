package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetEnvOrDefault(t *testing.T) {
	// Test with env var set
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	result := getEnvOrDefault("TEST_VAR", "default")
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", result)
	}

	// Test with env var not set
	result = getEnvOrDefault("NONEXISTENT_VAR", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got '%s'", result)
	}
}

func TestGetCurrentUsername(t *testing.T) {
	username := getCurrentUsername()
	if username == "" || username == "unknown" {
		t.Log("Could not get username, got:", username)
	}
}

func TestLoadDefaults(t *testing.T) {
	// Clear env vars that might interfere
	envVars := []string{
		"LANGFUSE_HOST", "LANGFUSE_PUBLIC_KEY", "LANGFUSE_SECRET_KEY",
		"CLAUDE_LANGFUSE_USER_ID", "CLAUDE_LANGFUSE_MODEL", "CLAUDE_LANGFUSE_SOURCE",
		"CLAUDE_LANGFUSE_USER_TRACE_NAME", "CLAUDE_LANGFUSE_ASSISTANT_TRACE_NAME",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check defaults
	if cfg.Host != "http://localhost:3001" {
		t.Errorf("Expected default host 'http://localhost:3001', got '%s'", cfg.Host)
	}
	if cfg.Model != "claude-code" {
		t.Errorf("Expected default model 'claude-code', got '%s'", cfg.Model)
	}
	if cfg.Source != "claude_code_monitor" {
		t.Errorf("Expected default source 'claude_code_monitor', got '%s'", cfg.Source)
	}
	if cfg.UserTraceName != "claude_code_user" {
		t.Errorf("Expected default userTraceName 'claude_code_user', got '%s'", cfg.UserTraceName)
	}
	if cfg.AssistantTraceName != "claude_response" {
		t.Errorf("Expected default assistantTraceName 'claude_response', got '%s'", cfg.AssistantTraceName)
	}
}

func TestLoadFromEnvVars(t *testing.T) {
	// Set env vars
	os.Setenv("LANGFUSE_HOST", "http://test:3000")
	os.Setenv("LANGFUSE_PUBLIC_KEY", "pk-test")
	os.Setenv("LANGFUSE_SECRET_KEY", "sk-test")
	os.Setenv("CLAUDE_LANGFUSE_USER_ID", "testuser")
	os.Setenv("CLAUDE_LANGFUSE_MODEL", "claude-opus-4")
	os.Setenv("CLAUDE_LANGFUSE_SOURCE", "test_source")

	defer func() {
		os.Unsetenv("LANGFUSE_HOST")
		os.Unsetenv("LANGFUSE_PUBLIC_KEY")
		os.Unsetenv("LANGFUSE_SECRET_KEY")
		os.Unsetenv("CLAUDE_LANGFUSE_USER_ID")
		os.Unsetenv("CLAUDE_LANGFUSE_MODEL")
		os.Unsetenv("CLAUDE_LANGFUSE_SOURCE")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Host != "http://test:3000" {
		t.Errorf("Expected host 'http://test:3000', got '%s'", cfg.Host)
	}
	if cfg.PublicKey != "pk-test" {
		t.Errorf("Expected publicKey 'pk-test', got '%s'", cfg.PublicKey)
	}
	if cfg.SecretKey != "sk-test" {
		t.Errorf("Expected secretKey 'sk-test', got '%s'", cfg.SecretKey)
	}
	if cfg.UserID != "testuser" {
		t.Errorf("Expected userId 'testuser', got '%s'", cfg.UserID)
	}
	if cfg.Model != "claude-opus-4" {
		t.Errorf("Expected model 'claude-opus-4', got '%s'", cfg.Model)
	}
	if cfg.Source != "test_source" {
		t.Errorf("Expected source 'test_source', got '%s'", cfg.Source)
	}
}

func TestSaveAndLoadFromFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".claude-langfuse")
	configFile := filepath.Join(configDir, "config.json")

	// Override DefaultConfigFile for testing
	originalFunc := DefaultConfigFile
	defer func() { _ = originalFunc }()

	// Create config manually
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	cfg := &Config{
		Host:               "http://file:3001",
		PublicKey:          "pk-file",
		SecretKey:          "sk-file",
		UserID:             "fileuser",
		Model:              "claude-file",
		Source:             "file_source",
		UserTraceName:      "file_user_trace",
		AssistantTraceName: "file_assistant_trace",
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configFile, data, 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Read it back
	readData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var readCfg Config
	if err := json.Unmarshal(readData, &readCfg); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if readCfg.Host != cfg.Host {
		t.Errorf("Host mismatch: expected '%s', got '%s'", cfg.Host, readCfg.Host)
	}
	if readCfg.UserID != cfg.UserID {
		t.Errorf("UserID mismatch: expected '%s', got '%s'", cfg.UserID, readCfg.UserID)
	}
}

func TestServiceName(t *testing.T) {
	// Test default
	os.Unsetenv("CLAUDE_LANGFUSE_SERVICE_NAME")
	name := ServiceName()
	if name != "claude-langfuse-monitor" {
		t.Errorf("Expected default service name 'claude-langfuse-monitor', got '%s'", name)
	}

	// Test with env var
	os.Setenv("CLAUDE_LANGFUSE_SERVICE_NAME", "custom-service")
	defer os.Unsetenv("CLAUDE_LANGFUSE_SERVICE_NAME")

	name = ServiceName()
	if name != "custom-service" {
		t.Errorf("Expected service name 'custom-service', got '%s'", name)
	}
}
