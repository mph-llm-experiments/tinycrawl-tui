package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

const (
	anthropicAPIURL = "https://api.anthropic.com/v1/messages"
	anthropicModel  = "claude-sonnet-4-20250514"
	anthropicMaxTok = 1024
)

type Client struct {
	baseURL      string
	passphrase   string
	anthropicKey string
	httpClient   *http.Client
	llmClient    *http.Client
}

func New(cfg Config) *Client {
	return &Client{
		baseURL:      cfg.Server,
		passphrase:   cfg.Passphrase,
		anthropicKey: cfg.AnthropicKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		llmClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) get(path string, auth bool) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	if auth {
		req.Header.Set("X-Player-ID", c.passphrase)
	}
	return c.httpClient.Do(req)
}

func (c *Client) postJSON(path string, body any) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Player-ID", c.passphrase)
	return c.httpClient.Do(req)
}

func decodeJSON[T any](resp *http.Response) (T, error) {
	var zero T
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return zero, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return zero, err
	}
	return result, nil
}

// Ping checks server health via GET /api/packs.
func (c *Client) Ping() bool {
	resp, err := c.get("/api/packs", false)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// FetchPacks returns available content packs.
func (c *Client) FetchPacks() ([]types.ContentPackMeta, error) {
	resp, err := c.get("/api/packs", false)
	if err != nil {
		return nil, err
	}
	return decodeJSON[[]types.ContentPackMeta](resp)
}

// FetchPack returns a full content pack by ID.
func (c *Client) FetchPack(id string) (*types.ContentPack, error) {
	resp, err := c.get("/api/pack/"+id, false)
	if err != nil {
		return nil, err
	}
	pack, err := decodeJSON[types.ContentPack](resp)
	if err != nil {
		return nil, err
	}
	return &pack, nil
}

// FetchDungeon fetches a generated dungeon for the given pack and game type.
func (c *Client) FetchDungeon(packID, gameType string) (json.RawMessage, error) {
	path := fmt.Sprintf("/api/dungeon?pack=%s&type=%s", packID, gameType)
	resp, err := c.get(path, true)
	if err != nil {
		return nil, err
	}
	return decodeJSON[json.RawMessage](resp)
}

// SendCreativeAction calls the GM for a creative action ruling.
// If AnthropicKey is configured, it calls the Anthropic API directly.
func (c *Client) SendCreativeAction(gmPrompt, userPrompt string) (map[string]any, error) {
	if c.anthropicKey != "" {
		return c.sendCreativeActionDirect(gmPrompt, userPrompt)
	}
	return c.sendCreativeActionProxy(gmPrompt, userPrompt)
}

func (c *Client) sendCreativeActionProxy(gmPrompt, userPrompt string) (map[string]any, error) {
	payload := map[string]string{
		"system": gmPrompt,
		"user":   userPrompt,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/gm", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.passphrase != "" {
		req.Header.Set("X-Player-ID", c.passphrase)
	}
	resp, err := c.llmClient.Do(req)
	if err != nil {
		return nil, err
	}
	return decodeJSON[map[string]any](resp)
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

func (c *Client) sendCreativeActionDirect(gmPrompt, userPrompt string) (map[string]any, error) {
	payload := anthropicRequest{
		Model:     anthropicModel,
		MaxTokens: anthropicMaxTok,
		System:    gmPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, anthropicAPIURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.anthropicKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.llmClient.Do(req)
	if err != nil {
		return nil, err
	}
	return decodeJSON[map[string]any](resp)
}

// RecordRun submits a completed run record to the server.
func (c *Client) RecordRun(run types.RunRecord) error {
	resp, err := c.postJSON("/api/runs", run)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// FetchOssuary returns global ossuary statistics.
func (c *Client) FetchOssuary() (*types.OssuaryStats, error) {
	resp, err := c.get("/api/ossuary", false)
	if err != nil {
		return nil, err
	}
	stats, err := decodeJSON[types.OssuaryStats](resp)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// FetchPlayerRuns returns run history for the authenticated player.
func (c *Client) FetchPlayerRuns() (*types.PlayerStats, error) {
	resp, err := c.get("/api/runs", true)
	if err != nil {
		return nil, err
	}
	stats, err := decodeJSON[types.PlayerStats](resp)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}
