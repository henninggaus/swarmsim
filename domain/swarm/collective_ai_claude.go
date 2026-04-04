package swarm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// ClaudeAIBackend handles real LLM API calls for code generation.
type ClaudeAIBackend struct {
	APIKey      string
	Model       string // default "claude-sonnet-4-20250514"
	mu          sync.Mutex
	lastCall    time.Time
	minInterval time.Duration // 5 seconds between calls
	pending     map[int]string // issueID -> generated code (from goroutine)
	errors      map[int]string // issueID -> error message
}

// NewClaudeAIBackend creates a new Claude API backend with the given API key.
func NewClaudeAIBackend(apiKey string) *ClaudeAIBackend {
	return &ClaudeAIBackend{
		APIKey:      apiKey,
		Model:       "claude-sonnet-4-20250514",
		minInterval: 5 * time.Second,
		pending:     make(map[int]string),
		errors:      make(map[int]string),
	}
}

// RequestCodeGeneration starts an async API call for the given issue.
// Returns immediately. Results are collected via CollectResults().
func (c *ClaudeAIBackend) RequestCodeGeneration(issue *SwarmIssue) {
	c.mu.Lock()
	// Rate limit: skip if called too soon
	if time.Since(c.lastCall) < c.minInterval {
		c.mu.Unlock()
		return // too soon, will retry next tick
	}
	c.lastCall = time.Now()
	c.mu.Unlock()

	// Build prompt from issue context
	prompt := buildSwarmScriptPrompt(issue)

	// Fire async API call
	go func() {
		code, err := callClaudeAPI(c.APIKey, c.Model, prompt)
		c.mu.Lock()
		defer c.mu.Unlock()
		if err != nil {
			c.errors[issue.ID] = err.Error()
		} else {
			c.pending[issue.ID] = code
		}
	}()
}

// CollectResults returns any completed code generations and clears the buffer.
// Caller should check periodically (e.g. every tick).
func (c *ClaudeAIBackend) CollectResults() map[int]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.pending) == 0 {
		return nil
	}
	results := c.pending
	c.pending = make(map[int]string)
	return results
}

// CollectErrors returns any failed generations and clears the buffer.
func (c *ClaudeAIBackend) CollectErrors() map[int]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.errors) == 0 {
		return nil
	}
	errs := c.errors
	c.errors = make(map[int]string)
	return errs
}

// buildSwarmScriptPrompt constructs the LLM prompt for a given issue.
func buildSwarmScriptPrompt(issue *SwarmIssue) string {
	return fmt.Sprintf(`You are programming an autonomous swarm robot using SwarmScript.

The robot has detected a problem: %s
Current sensor readings: %s

Generate 1-3 SwarmScript rules to solve this problem.
Each rule must be on its own line in this exact format:
IF <sensor> <operator> <value> THEN <action>

Available sensors (use exact names):
- carry (0=empty, 1+=carrying package)
- p_dist (distance to nearest pickup, pixels)
- d_dist (distance to nearest dropoff, pixels)
- match (1=carrying matches nearest dropoff)
- near_dist (distance to nearest bot, pixels)
- neighbors (count of nearby bots)
- obs_ahead (1=obstacle ahead, 0=clear)
- wall_right (1=wall on right side)
- wall_left (1=wall on left side)
- light (light intensity 0-100)
- energy (energy level 0-100)
- rnd (random 0-100)
- state (internal state 0-9)
- heading (direction 0-359 degrees)

Available operators: > < ==

Available actions (use exact names):
- FWD (move forward)
- STOP (stop moving)
- TURN_LEFT N (turn N degrees left)
- TURN_RIGHT N (turn N degrees right)
- TURN_RANDOM (random direction)
- TURN_TO_NEAREST (face nearest bot)
- TURN_FROM_NEAREST (face away from nearest bot)
- TURN_TO_LIGHT (face light source)
- PICKUP (pick up package)
- DROP (drop package)
- GOTO_DROPOFF (navigate to matching dropoff)
- GOTO_PICKUP (navigate to nearest pickup)
- AVOID_OBSTACLE (dodge obstacle)
- WALL_FOLLOW_RIGHT (follow right wall)
- WALL_FOLLOW_LEFT (follow left wall)
- SEND_MESSAGE N (broadcast message type N)
- SET_STATE N (set internal state)

IMPORTANT: Output ONLY the SwarmScript rules, nothing else. No explanations, no markdown.
Example output:
IF obs_ahead == 1 THEN TURN_RIGHT 90
IF carry == 0 AND p_dist < 30 THEN PICKUP
IF true THEN FWD`, issue.Problem, issue.SensorSnap)
}

// claudeAPIRequest is the request body for the Anthropic Messages API.
type claudeAPIRequest struct {
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens"`
	Messages  []claudeAPIMessage  `json:"messages"`
}

// claudeAPIMessage is a single message in the API request.
type claudeAPIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeAPIResponse is the response body from the Anthropic Messages API.
type claudeAPIResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// callClaudeAPI makes the actual HTTP request to the Anthropic Messages API.
func callClaudeAPI(apiKey, model, prompt string) (string, error) {
	reqBody := claudeAPIRequest{
		Model:     model,
		MaxTokens: 200,
		Messages: []claudeAPIMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBytes))
	}

	// Parse response
	var result claudeAPIResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return result.Content[0].Text, nil
}

// EnableClaudeAPI activates the Claude API backend for code generation.
func EnableClaudeAPI(ss *SwarmState, apiKey string) {
	ss.IssueBoard.UseClaudeAPI = true
	ss.IssueBoard.ClaudeBackend = NewClaudeAIBackend(apiKey)
}

// DisableClaudeAPI deactivates the Claude API backend, reverting to templates.
func DisableClaudeAPI(ss *SwarmState) {
	ss.IssueBoard.UseClaudeAPI = false
	ss.IssueBoard.ClaudeBackend = nil
}
