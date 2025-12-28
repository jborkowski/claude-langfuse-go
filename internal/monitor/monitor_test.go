package monitor

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractContent_TextBlock(t *testing.T) {
	mon := &Monitor{}

	msg := MessageContent{
		Content: []ContentBlock{
			{Type: "text", Text: "Hello, world!"},
		},
	}

	data, _ := json.Marshal(msg)
	result := mon.extractContent(data)

	if result != "Hello, world!" {
		t.Errorf("Expected 'Hello, world!', got '%s'", result)
	}
}

func TestExtractContent_ToolUse(t *testing.T) {
	mon := &Monitor{}

	input := map[string]string{"file_path": "/test.js"}
	inputData, _ := json.Marshal(input)

	msg := MessageContent{
		Content: []ContentBlock{
			{Type: "tool_use", Name: "Read", Input: inputData},
		},
	}

	data, _ := json.Marshal(msg)
	result := mon.extractContent(data)

	expected := `[Tool: Read]
{
  "file_path": "/test.js"
}`
	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

func TestExtractContent_ToolResult(t *testing.T) {
	mon := &Monitor{}

	msg := MessageContent{
		Content: []ContentBlock{
			{Type: "tool_result", Content: "File contents here"},
		},
	}

	data, _ := json.Marshal(msg)
	result := mon.extractContent(data)

	if result != "File contents here" {
		t.Errorf("Expected 'File contents here', got '%s'", result)
	}
}

func TestExtractContent_MixedBlocks(t *testing.T) {
	mon := &Monitor{}

	input := map[string]string{"file_path": "/test.js"}
	inputData, _ := json.Marshal(input)

	msg := MessageContent{
		Content: []ContentBlock{
			{Type: "text", Text: "Let me read that file"},
			{Type: "tool_use", Name: "Read", Input: inputData},
		},
	}

	data, _ := json.Marshal(msg)
	result := mon.extractContent(data)

	if result == "" {
		t.Error("Expected non-empty result for mixed blocks")
	}
	if len(result) < 20 {
		t.Errorf("Result seems too short: %s", result)
	}
}

func TestExtractContent_StringMessage(t *testing.T) {
	mon := &Monitor{}

	data, _ := json.Marshal("Simple string message")
	result := mon.extractContent(data)

	if result != "Simple string message" {
		t.Errorf("Expected 'Simple string message', got '%s'", result)
	}
}

func TestExtractContent_TextFieldFallback(t *testing.T) {
	mon := &Monitor{}

	msg := MessageContent{
		Text: "Fallback text",
	}

	data, _ := json.Marshal(msg)
	result := mon.extractContent(data)

	if result != "Fallback text" {
		t.Errorf("Expected 'Fallback text', got '%s'", result)
	}
}

func TestSessionIDGeneration(t *testing.T) {
	projectPath := "Users/test/Documents/github/myproject"
	conversationID := "conv-abc-123"
	sessionData := fmt.Sprintf("%s:%s", projectPath, conversationID)

	hash := md5.Sum([]byte(sessionData))
	sessionID := hex.EncodeToString(hash[:])

	if len(sessionID) != 32 {
		t.Errorf("Session ID should be 32 chars (MD5 hex), got %d", len(sessionID))
	}

	// Same input should produce same hash
	hash2 := md5.Sum([]byte(sessionData))
	sessionID2 := hex.EncodeToString(hash2[:])

	if sessionID != sessionID2 {
		t.Error("Session ID should be deterministic")
	}

	// Different input should produce different hash
	differentData := fmt.Sprintf("%s:%s", projectPath, "different-conv")
	hash3 := md5.Sum([]byte(differentData))
	sessionID3 := hex.EncodeToString(hash3[:])

	if sessionID == sessionID3 {
		t.Error("Different conversations should have different session IDs")
	}
}

func TestProjectPathDecoding(t *testing.T) {
	tests := []struct {
		encoded  string
		expected string
	}{
		{"Users-test-Documents-github-myproject", "Users/test/Documents/github/myproject"},
		{"home-user-projects-app", "home/user/projects/app"},
		{"simple", "simple"},
	}

	for _, tc := range tests {
		result := decodeProjectPath(tc.encoded)
		if result != tc.expected {
			t.Errorf("decodeProjectPath(%s) = %s, expected %s", tc.encoded, result, tc.expected)
		}
	}
}

func decodeProjectPath(encoded string) string {
	// Import strings at package level for this
	result := encoded
	for i := range result {
		if result[i] == '-' {
			result = result[:i] + "/" + result[i+1:]
		}
	}
	return result
}

func TestProcessMessage_Deduplication(t *testing.T) {
	mon := &Monitor{
		options:              Options{DryRun: true, Quiet: true},
		processedMessages:    make(map[string]bool),
		conversationSessions: make(map[string]string),
	}

	entry := &Entry{
		Type:      "user",
		UUID:      "test-uuid-123",
		Timestamp: "2024-01-01T00:00:00Z",
		Message:   json.RawMessage(`"Test message"`),
	}

	// First call should process
	mon.ProcessMessage(entry, "session-1", "/test/project", "conv-1")
	if !mon.processedMessages["test-uuid-123"] {
		t.Error("Message should be marked as processed")
	}

	userCount, _ := mon.MessageStats()
	if userCount != 1 {
		t.Errorf("Expected 1 user message, got %d", userCount)
	}

	// Second call should be deduplicated
	mon.ProcessMessage(entry, "session-1", "/test/project", "conv-1")
	userCount, _ = mon.MessageStats()
	if userCount != 1 {
		t.Errorf("Expected 1 user message (deduplicated), got %d", userCount)
	}
}

func TestProcessMessage_IgnoresNonUserAssistant(t *testing.T) {
	mon := &Monitor{
		options:              Options{DryRun: true, Quiet: true},
		processedMessages:    make(map[string]bool),
		conversationSessions: make(map[string]string),
	}

	entry := &Entry{
		Type:      "system",
		UUID:      "system-uuid",
		Timestamp: "2024-01-01T00:00:00Z",
		Message:   json.RawMessage(`"System message"`),
	}

	mon.ProcessMessage(entry, "session-1", "/test/project", "conv-1")

	if mon.processedMessages["system-uuid"] {
		t.Error("System message should not be processed")
	}

	userCount, assistantCount := mon.MessageStats()
	if userCount != 0 || assistantCount != 0 {
		t.Errorf("Expected 0 messages, got %d user, %d assistant", userCount, assistantCount)
	}
}

func TestProcessMessage_SkipsWithoutUUID(t *testing.T) {
	mon := &Monitor{
		options:              Options{DryRun: true, Quiet: true},
		processedMessages:    make(map[string]bool),
		conversationSessions: make(map[string]string),
	}

	entry := &Entry{
		Type:      "user",
		UUID:      "", // Empty UUID
		Timestamp: "2024-01-01T00:00:00Z",
		Message:   json.RawMessage(`"Test message"`),
	}

	mon.ProcessMessage(entry, "session-1", "/test/project", "conv-1")

	userCount, _ := mon.MessageStats()
	if userCount != 0 {
		t.Errorf("Expected 0 user messages (no UUID), got %d", userCount)
	}
}

func TestProcessConversationFile_InvalidPath(t *testing.T) {
	mon := &Monitor{
		options:              Options{DryRun: true, Quiet: true},
		processedMessages:    make(map[string]bool),
		conversationSessions: make(map[string]string),
	}

	// Path without "projects" should be ignored
	mon.ProcessConversationFile("/random/path/file.jsonl")

	if len(mon.conversationSessions) != 0 {
		t.Error("Invalid path should not create session")
	}
}

func TestProcessConversationFile_ValidJSONL(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "projects", "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	jsonlFile := filepath.Join(projectDir, "conv-123.jsonl")
	content := `{"type":"user","uuid":"msg-1","message":"Hello","timestamp":"2024-01-01T00:00:00Z"}
{"type":"assistant","uuid":"msg-2","parentUuid":"msg-1","message":"Hi there!","timestamp":"2024-01-01T00:00:01Z"}`

	if err := os.WriteFile(jsonlFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write JSONL: %v", err)
	}

	mon := &Monitor{
		options:              Options{DryRun: true, Quiet: true},
		processedMessages:    make(map[string]bool),
		conversationSessions: make(map[string]string),
	}

	mon.ProcessConversationFile(jsonlFile)

	userCount, assistantCount := mon.MessageStats()
	if userCount != 1 {
		t.Errorf("Expected 1 user message, got %d", userCount)
	}
	if assistantCount != 1 {
		t.Errorf("Expected 1 assistant message, got %d", assistantCount)
	}

	if !mon.processedMessages["msg-1"] {
		t.Error("msg-1 should be marked as processed")
	}
	if !mon.processedMessages["msg-2"] {
		t.Error("msg-2 should be marked as processed")
	}
}

func TestProcessConversationFile_InvalidJSON(t *testing.T) {
	// Create temp file with invalid JSON
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "projects", "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	jsonlFile := filepath.Join(projectDir, "conv-123.jsonl")
	content := `invalid json
{"type":"user","uuid":"valid-msg","message":"Hello","timestamp":"2024-01-01T00:00:00Z"}`

	if err := os.WriteFile(jsonlFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write JSONL: %v", err)
	}

	mon := &Monitor{
		options:              Options{DryRun: true, Quiet: true},
		processedMessages:    make(map[string]bool),
		conversationSessions: make(map[string]string),
	}

	// Should not panic, should skip invalid line
	mon.ProcessConversationFile(jsonlFile)

	userCount, _ := mon.MessageStats()
	if userCount != 1 {
		t.Errorf("Expected 1 user message (skipped invalid), got %d", userCount)
	}
}
