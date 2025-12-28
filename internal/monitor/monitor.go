// Package monitor implements the core monitoring logic.
package monitor

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/user/claude-langfuse-go/internal/config"
	"github.com/user/claude-langfuse-go/internal/langfuse"
)

// Options configures the monitor behavior.
type Options struct {
	HistoryHours int
	Daemon       bool
	DryRun       bool
	Quiet        bool
}

// Monitor watches Claude Code conversation files and creates Langfuse traces.
type Monitor struct {
	options Options
	config  *config.Config
	client  *langfuse.Client

	mu                   sync.Mutex
	processedMessages    map[string]bool
	conversationSessions map[string]string
	messageCount         struct {
		user      int
		assistant int
	}
}

// Entry represents a JSONL conversation entry.
type Entry struct {
	Type       string          `json:"type"`
	UUID       string          `json:"uuid"`
	ParentUUID string          `json:"parentUuid"`
	Timestamp  string          `json:"timestamp"`
	Message    json.RawMessage `json:"message"`
	GitBranch  string          `json:"gitBranch"`
	Cwd        string          `json:"cwd"`
	RequestID  string          `json:"requestId"`
}

// MessageContent represents the message field structure.
type MessageContent struct {
	Text    string         `json:"text"`
	Content []ContentBlock `json:"content"`
	Model   string         `json:"model"` // Model used for this response (e.g., "claude-opus-4-5-20251101")
}

// ContentBlock represents a content block in a message.
type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	Content   string          `json:"content"`
	ToolUseID string          `json:"tool_use_id"`
}

// New creates a new Monitor.
func New(opts Options) (*Monitor, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	m := &Monitor{
		options:              opts,
		config:               cfg,
		processedMessages:    make(map[string]bool),
		conversationSessions: make(map[string]string),
	}

	if !opts.DryRun {
		m.client = langfuse.NewClient(cfg.Host, cfg.PublicKey, cfg.SecretKey)
	}

	return m, nil
}

// GetClaudeProjectsDir returns the Claude projects directory.
func GetClaudeProjectsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	projectsDir := filepath.Join(home, ".claude", "projects")
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		return "", fmt.Errorf("Claude projects directory not found: %s", projectsDir)
	}

	return projectsDir, nil
}

// ProcessExistingHistory processes recent conversation files.
func (m *Monitor) ProcessExistingHistory() error {
	cyan := color.New(color.FgCyan)
	gray := color.New(color.FgHiBlack)
	green := color.New(color.FgGreen)

	cyan.Printf("Processing last %d hours...\n", m.options.HistoryHours)

	projectsDir, err := GetClaudeProjectsDir()
	if err != nil {
		return err
	}

	cutoffTime := time.Now().Add(-time.Duration(m.options.HistoryHours) * time.Hour)

	var conversations []string
	err = filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() && strings.HasSuffix(path, ".jsonl") {
			if info.ModTime().After(cutoffTime) {
				conversations = append(conversations, path)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan conversations: %w", err)
	}

	gray.Printf("  Found %d recent conversations\n", len(conversations))

	for _, filepath := range conversations {
		m.ProcessConversationFile(filepath)
	}

	totalMessages := m.messageCount.user + m.messageCount.assistant
	green.Printf("[OK] Processed %d conversations (%d messages: %d user, %d assistant)\n",
		len(conversations), totalMessages, m.messageCount.user, m.messageCount.assistant)

	return nil
}

// ProcessConversationFile processes a single JSONL conversation file.
func (m *Monitor) ProcessConversationFile(filepath string) {
	// Extract project path from file location
	parts := strings.Split(filepath, string(os.PathSeparator))

	projectsIdx := -1
	for i, part := range parts {
		if part == "projects" {
			projectsIdx = i
			break
		}
	}

	if projectsIdx == -1 || projectsIdx >= len(parts)-2 {
		return
	}

	encodedProject := parts[projectsIdx+1]
	projectPath := strings.ReplaceAll(encodedProject, "-", "/")
	conversationID := strings.TrimSuffix(parts[len(parts)-1], ".jsonl")

	// Get or create session ID
	m.mu.Lock()
	sessionID, exists := m.conversationSessions[filepath]
	if !exists {
		sessionData := fmt.Sprintf("%s:%s", projectPath, conversationID)
		hash := md5.Sum([]byte(sessionData))
		sessionID = hex.EncodeToString(hash[:])
		m.conversationSessions[filepath] = sessionID
	}
	m.mu.Unlock()

	// Read and process messages
	file, err := os.Open(filepath)
	if err != nil {
		color.Red("Error reading %s: %v", filepath, err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max line size

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip invalid JSON lines
		}

		m.ProcessMessage(&entry, sessionID, projectPath, conversationID)
	}
}

// ProcessMessage processes a single message entry.
func (m *Monitor) ProcessMessage(entry *Entry, sessionID, projectPath, conversationID string) {
	msgType := entry.Type

	if msgType != "user" && msgType != "assistant" {
		return
	}

	uuid := entry.UUID
	if uuid == "" {
		return
	}

	// Check deduplication
	m.mu.Lock()
	if m.processedMessages[uuid] {
		m.mu.Unlock()
		return
	}
	m.processedMessages[uuid] = true
	m.mu.Unlock()

	// Extract message content
	text := m.extractContent(entry.Message)

	// Parse timestamp
	timestamp := time.Now()
	if entry.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, entry.Timestamp); err == nil {
			timestamp = t
		}
	}

	// Track message counts
	m.mu.Lock()
	if msgType == "user" {
		m.messageCount.user++
	} else {
		m.messageCount.assistant++
	}
	m.mu.Unlock()

	// Print activity (unless quiet mode)
	if !m.options.Quiet {
		projectName := filepath.Base(projectPath)
		preview := text
		if len(preview) > 60 {
			preview = preview[:60]
		}
		preview = strings.ReplaceAll(preview, "\n", " ")

		icon := "[user]"
		if msgType == "assistant" {
			icon = "[assistant]"
		}
		gray := color.New(color.FgHiBlack)
		gray.Printf("%s [%s] %s...\n", icon, projectName, preview)
	}

	if m.options.DryRun || m.client == nil {
		return
	}

	// Create trace in Langfuse
	if msgType == "user" {
		trace := &langfuse.Trace{
			ID:        uuid,
			Name:      m.config.UserTraceName,
			SessionID: sessionID,
			UserID:    m.config.UserID,
			Metadata: map[string]interface{}{
				"project":        projectPath,
				"conversationId": conversationID,
				"gitBranch":      entry.GitBranch,
				"cwd":            entry.Cwd,
				"messageType":    msgType,
				"source":         m.config.Source,
			},
			Input:     text,
			Timestamp: timestamp,
		}
		if err := m.client.CreateTrace(trace); err != nil {
			color.Red("Error creating trace: %v", err)
		}
	} else if msgType == "assistant" {
		// Extract model from message if available, fallback to config
		model := m.extractModel(entry.Message)
		if model == "" {
			model = m.config.Model
		}

		gen := &langfuse.Generation{
			ID:      uuid,
			TraceID: entry.ParentUUID,
			Name:    m.config.AssistantTraceName,
			Model:   model,
			Metadata: map[string]interface{}{
				"project":        projectPath,
				"conversationId": conversationID,
				"requestId":      entry.RequestID,
				"messageType":    msgType,
				"source":         m.config.Source,
			},
			Output:    text,
			StartTime: timestamp,
			EndTime:   timestamp,
		}
		if err := m.client.CreateGeneration(gen); err != nil {
			color.Red("Error creating generation: %v", err)
		}
	}
}

// extractModel extracts the model name from the message field.
func (m *Monitor) extractModel(rawMessage json.RawMessage) string {
	if len(rawMessage) == 0 {
		return ""
	}

	// Try as MessageContent object
	var msgContent MessageContent
	if err := json.Unmarshal(rawMessage, &msgContent); err != nil {
		return ""
	}

	// Return model if present and not synthetic
	if msgContent.Model != "" && msgContent.Model != "<synthetic>" {
		return msgContent.Model
	}

	return ""
}

// extractContent extracts text from the message field.
func (m *Monitor) extractContent(rawMessage json.RawMessage) string {
	if len(rawMessage) == 0 {
		return ""
	}

	// Try as string first
	var strMessage string
	if err := json.Unmarshal(rawMessage, &strMessage); err == nil {
		return strMessage
	}

	// Try as MessageContent object
	var msgContent MessageContent
	if err := json.Unmarshal(rawMessage, &msgContent); err != nil {
		return ""
	}

	// If content array exists, process it
	if len(msgContent.Content) > 0 {
		var parts []string
		for _, block := range msgContent.Content {
			switch block.Type {
			case "text":
				if block.Text != "" {
					parts = append(parts, block.Text)
				}
			case "tool_use":
				inputStr := ""
				if len(block.Input) > 0 {
					var prettyInput bytes.Buffer
					if json.Indent(&prettyInput, block.Input, "", "  ") == nil {
						inputStr = prettyInput.String()
					} else {
						inputStr = string(block.Input)
					}
				}
				parts = append(parts, fmt.Sprintf("[Tool: %s]\n%s", block.Name, inputStr))
			case "tool_result":
				if block.Content != "" {
					parts = append(parts, block.Content)
				}
			default:
				// Handle text field on block directly
				if block.Text != "" {
					parts = append(parts, block.Text)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n\n")
		}
	}

	// Fallback to text field
	if msgContent.Text != "" {
		return msgContent.Text
	}

	return ""
}

// Shutdown stops the monitor and flushes pending events.
func (m *Monitor) Shutdown() error {
	if m.client != nil {
		return m.client.Shutdown()
	}
	return nil
}

// Config returns the loaded configuration.
func (m *Monitor) Config() *config.Config {
	return m.config
}

// MessageStats returns the message counts.
func (m *Monitor) MessageStats() (user, assistant int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.messageCount.user, m.messageCount.assistant
}

