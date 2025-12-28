// Package langfuse provides a REST API client for Langfuse.
package langfuse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Client is a Langfuse REST API client.
type Client struct {
	baseURL    string
	publicKey  string
	secretKey  string
	httpClient *http.Client

	// Batching
	mu        sync.Mutex
	events    []Event
	batchSize int
}

// Event represents a Langfuse event (trace or generation).
type Event struct {
	Type string      `json:"type"`
	Body interface{} `json:"body"`
}

// Trace represents a Langfuse trace.
type Trace struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	SessionID string                 `json:"sessionId,omitempty"`
	UserID    string                 `json:"userId,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Input     interface{}            `json:"input,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Generation represents a Langfuse generation.
type Generation struct {
	ID        string                 `json:"id"`
	TraceID   string                 `json:"traceId,omitempty"`
	Name      string                 `json:"name"`
	Model     string                 `json:"model,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Output    interface{}            `json:"output,omitempty"`
	StartTime time.Time              `json:"startTime"`
	EndTime   time.Time              `json:"endTime"`
}

// NewClient creates a new Langfuse client.
func NewClient(baseURL, publicKey, secretKey string) *Client {
	return &Client{
		baseURL:   baseURL,
		publicKey: publicKey,
		secretKey: secretKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		events:    make([]Event, 0),
		batchSize: 10,
	}
}

// CreateTrace creates a trace in Langfuse.
func (c *Client) CreateTrace(trace *Trace) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.events = append(c.events, Event{
		Type: "trace-create",
		Body: trace,
	})

	// Auto-flush when batch size reached
	if len(c.events) >= c.batchSize {
		return c.flushLocked()
	}

	return nil
}

// CreateGeneration creates a generation in Langfuse.
func (c *Client) CreateGeneration(gen *Generation) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.events = append(c.events, Event{
		Type: "generation-create",
		Body: gen,
	})

	// Auto-flush when batch size reached
	if len(c.events) >= c.batchSize {
		return c.flushLocked()
	}

	return nil
}

// Flush sends all pending events to Langfuse.
func (c *Client) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.flushLocked()
}

// flushLocked sends events (must be called with lock held).
func (c *Client) flushLocked() error {
	if len(c.events) == 0 {
		return nil
	}

	// Create batch request
	payload := map[string]interface{}{
		"batch": c.events,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", c.baseURL+"/api/public/ingestion", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.publicKey, c.secretKey)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("langfuse API error: status %d", resp.StatusCode)
	}

	// Clear events on success
	c.events = c.events[:0]

	return nil
}

// Shutdown flushes remaining events and closes the client.
func (c *Client) Shutdown() error {
	return c.Flush()
}

// EventCount returns the number of pending events.
func (c *Client) EventCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.events)
}
