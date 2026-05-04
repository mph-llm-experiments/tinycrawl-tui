package client

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

const statsFile = "stats.json"

// statsPath returns the path to the local stats file.
func statsPath() string {
	return filepath.Join(ConfigDir(), statsFile)
}

// LoadLocalStats reads the local stats file, returning an empty PlayerStats if
// the file does not exist.
func LoadLocalStats() (*types.PlayerStats, error) {
	data, err := os.ReadFile(statsPath())
	if os.IsNotExist(err) {
		return &types.PlayerStats{}, nil
	}
	if err != nil {
		return nil, err
	}
	var stats types.PlayerStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// SaveRun appends a run record to local stats and recalculates aggregates.
func SaveRun(run types.RunRecord) error {
	stats, err := LoadLocalStats()
	if err != nil {
		return err
	}

	stats.Runs = append(stats.Runs, run)

	// Recalculate totals.
	stats.TotalRuns = len(stats.Runs)
	stats.Victories = 0
	stats.BestDepth = 0
	stats.TotalMonstersKilled = 0

	for _, r := range stats.Runs {
		if r.Result == "victory" {
			stats.Victories++
		}
		if r.Depth > stats.BestDepth {
			stats.BestDepth = r.Depth
		}
		stats.TotalMonstersKilled += r.MonstersKilled
	}

	if err := os.MkdirAll(ConfigDir(), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statsPath(), data, 0o644)
}
