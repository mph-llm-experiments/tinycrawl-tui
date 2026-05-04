package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// --- Config tests ---

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Server != "https://tinycrawl.puddingtime.net" {
		t.Errorf("unexpected default server: %s", cfg.Server)
	}
	if cfg.DefaultPack != "the-dark-below" {
		t.Errorf("unexpected default pack: %s", cfg.DefaultPack)
	}
	if cfg.DefaultType != "sprint" {
		t.Errorf("unexpected default type: %s", cfg.DefaultType)
	}
	if cfg.ImageMode != "auto" {
		t.Errorf("unexpected default image mode: %s", cfg.ImageMode)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	dir := t.TempDir()
	origOverride := configDirOverride
	configDirOverride = dir
	defer func() { configDirOverride = origOverride }()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error for missing file: %v", err)
	}
	// Defaults should be intact.
	if cfg.Server != "https://tinycrawl.puddingtime.net" {
		t.Errorf("got server %q, want default", cfg.Server)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	origOverride := configDirOverride
	configDirOverride = dir
	defer func() { configDirOverride = origOverride }()

	content := `
passphrase = "secret"
server = "https://example.com"
anthropic_key = "sk-test"
default_pack = "test-pack"
default_type = "expedition"
image_mode = "off"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Passphrase != "secret" {
		t.Errorf("passphrase = %q, want %q", cfg.Passphrase, "secret")
	}
	if cfg.Server != "https://example.com" {
		t.Errorf("server = %q, want %q", cfg.Server, "https://example.com")
	}
	if cfg.AnthropicKey != "sk-test" {
		t.Errorf("anthropic_key = %q, want %q", cfg.AnthropicKey, "sk-test")
	}
	if cfg.DefaultPack != "test-pack" {
		t.Errorf("default_pack = %q, want %q", cfg.DefaultPack, "test-pack")
	}
	if cfg.DefaultType != "expedition" {
		t.Errorf("default_type = %q, want %q", cfg.DefaultType, "expedition")
	}
	if cfg.ImageMode != "off" {
		t.Errorf("image_mode = %q, want %q", cfg.ImageMode, "off")
	}
}

func TestDecodeTomlHelper(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`server = "https://helper.example.com"`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	if _, err := decodeToml(path, &cfg); err != nil {
		t.Fatalf("decodeToml: %v", err)
	}
	if cfg.Server != "https://helper.example.com" {
		t.Errorf("server = %q, want %q", cfg.Server, "https://helper.example.com")
	}
}

func TestTomlRoundTrip(t *testing.T) {
	// Ensure toml tags match expected keys by encoding and decoding.
	dir := t.TempDir()
	path := filepath.Join(dir, "rt.toml")

	content := `
passphrase = "p"
server = "s"
anthropic_key = "a"
default_pack = "dp"
default_type = "dt"
image_mode = "im"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Passphrase != "p" || cfg.Server != "s" || cfg.AnthropicKey != "a" ||
		cfg.DefaultPack != "dp" || cfg.DefaultType != "dt" || cfg.ImageMode != "im" {
		t.Errorf("round-trip mismatch: %+v", cfg)
	}
}

// --- Client constructor tests ---

func TestNew(t *testing.T) {
	cfg := Config{
		Server:     "https://test.example.com",
		Passphrase: "hunter2",
	}
	c := New(cfg)
	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.baseURL != "https://test.example.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://test.example.com")
	}
	if c.passphrase != "hunter2" {
		t.Errorf("passphrase = %q, want %q", c.passphrase, "hunter2")
	}
	if c.httpClient == nil {
		t.Error("httpClient is nil")
	}
	if c.httpClient.Timeout != 5*time.Second {
		t.Errorf("httpClient.Timeout = %v, want 5s", c.httpClient.Timeout)
	}
}

func TestNewWithAnthropicKey(t *testing.T) {
	cfg := Config{
		Server:       "https://test.example.com",
		AnthropicKey: "sk-ant-test",
	}
	c := New(cfg)
	if c.anthropicKey != "sk-ant-test" {
		t.Errorf("anthropicKey = %q, want %q", c.anthropicKey, "sk-ant-test")
	}
}

func TestNewDefaultsFromConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Passphrase = "mypass"
	c := New(cfg)
	if c.baseURL != "https://tinycrawl.puddingtime.net" {
		t.Errorf("baseURL = %q, want default", c.baseURL)
	}
	if c.passphrase != "mypass" {
		t.Errorf("passphrase = %q, want %q", c.passphrase, "mypass")
	}
}

// --- Stats tests ---

func TestSaveAndLoadStats(t *testing.T) {
	dir := t.TempDir()
	origOverride := configDirOverride
	configDirOverride = dir
	defer func() { configDirOverride = origOverride }()

	// Initially empty.
	stats, err := LoadLocalStats()
	if err != nil {
		t.Fatalf("LoadLocalStats: %v", err)
	}
	if stats.TotalRuns != 0 {
		t.Errorf("expected 0 runs, got %d", stats.TotalRuns)
	}
	if len(stats.Runs) != 0 {
		t.Errorf("expected empty runs slice, got %d entries", len(stats.Runs))
	}

	// Save first run (a death).
	run1 := types.RunRecord{
		Name:           "Theron",
		Depth:          3,
		TotalRooms:     5,
		MonstersKilled: 2,
		Result:         "death",
		PackName:       "the-dark-below",
		Timestamp:      1000,
	}
	if err := SaveRun(run1); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	stats, err = LoadLocalStats()
	if err != nil {
		t.Fatalf("LoadLocalStats: %v", err)
	}
	if stats.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, want 1", stats.TotalRuns)
	}
	if stats.Victories != 0 {
		t.Errorf("Victories = %d, want 0", stats.Victories)
	}
	if stats.BestDepth != 3 {
		t.Errorf("BestDepth = %d, want 3", stats.BestDepth)
	}
	if stats.TotalMonstersKilled != 2 {
		t.Errorf("TotalMonstersKilled = %d, want 2", stats.TotalMonstersKilled)
	}

	// Save second run (a victory at greater depth).
	run2 := types.RunRecord{
		Name:           "Mira",
		Depth:          7,
		TotalRooms:     10,
		MonstersKilled: 5,
		Result:         "victory",
		PackName:       "the-dark-below",
		Timestamp:      2000,
	}
	if err := SaveRun(run2); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	stats, err = LoadLocalStats()
	if err != nil {
		t.Fatalf("LoadLocalStats: %v", err)
	}
	if stats.TotalRuns != 2 {
		t.Errorf("TotalRuns = %d, want 2", stats.TotalRuns)
	}
	if stats.Victories != 1 {
		t.Errorf("Victories = %d, want 1", stats.Victories)
	}
	if stats.BestDepth != 7 {
		t.Errorf("BestDepth = %d, want 7", stats.BestDepth)
	}
	if stats.TotalMonstersKilled != 7 {
		t.Errorf("TotalMonstersKilled = %d, want 7", stats.TotalMonstersKilled)
	}
}

func TestMultipleVictories(t *testing.T) {
	dir := t.TempDir()
	origOverride := configDirOverride
	configDirOverride = dir
	defer func() { configDirOverride = origOverride }()

	for i := 0; i < 3; i++ {
		run := types.RunRecord{
			Name:      "Hero",
			Depth:     5,
			Result:    "victory",
			PackName:  "test",
			Timestamp: int64(i + 1),
		}
		if err := SaveRun(run); err != nil {
			t.Fatalf("SaveRun %d: %v", i, err)
		}
	}

	stats, err := LoadLocalStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Victories != 3 {
		t.Errorf("Victories = %d, want 3", stats.Victories)
	}
	if stats.TotalRuns != 3 {
		t.Errorf("TotalRuns = %d, want 3", stats.TotalRuns)
	}
}

func TestLoadLocalStatsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	origOverride := configDirOverride
	configDirOverride = dir
	defer func() { configDirOverride = origOverride }()

	if err := os.WriteFile(filepath.Join(dir, statsFile), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadLocalStats()
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestStatsFileContentsAreValidJSON(t *testing.T) {
	dir := t.TempDir()
	origOverride := configDirOverride
	configDirOverride = dir
	defer func() { configDirOverride = origOverride }()

	run := types.RunRecord{
		Name:      "Test",
		Depth:     1,
		Result:    "death",
		PackName:  "test",
		Timestamp: 999,
	}
	if err := SaveRun(run); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, statsFile))
	if err != nil {
		t.Fatal(err)
	}

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Errorf("stats file is not valid JSON: %v", err)
	}
}
