package content

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mph-llm-experiments/tinycrawl-tui/internal/rng"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

//go:embed packs/*.json
var embeddedPacks embed.FS

// LoadEmbeddedPacks loads all packs embedded in the binary.
func LoadEmbeddedPacks() ([]types.ContentPack, error) {
	entries, err := embeddedPacks.ReadDir("packs")
	if err != nil {
		return nil, fmt.Errorf("read embedded packs: %w", err)
	}
	var packs []types.ContentPack
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := embeddedPacks.ReadFile("packs/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("read pack %s: %w", e.Name(), err)
		}
		var pack types.ContentPack
		if err := json.Unmarshal(data, &pack); err != nil {
			return nil, fmt.Errorf("parse pack %s: %w", e.Name(), err)
		}
		packs = append(packs, pack)
	}
	return packs, nil
}

// LoadLocalPacks loads JSON packs from a directory.
func LoadLocalPacks(dir string) ([]types.ContentPack, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var packs []types.ContentPack
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var pack types.ContentPack
		if err := json.Unmarshal(data, &pack); err != nil {
			continue
		}
		packs = append(packs, pack)
	}
	return packs, nil
}

// ContentFromPack extracts ContentData from a ContentPack.
func ContentFromPack(pack types.ContentPack) types.ContentData {
	return types.ContentData{
		Monsters:     pack.Monsters,
		Items:        pack.Items,
		Rooms:        pack.Rooms,
		Fragments:    pack.Fragments,
		StartingGear: pack.StartingGear,
		Names:        pack.Names,
	}
}

// GetMonstersByTier filters monsters by tier.
func GetMonstersByTier(tier string, cd types.ContentData) []types.Monster {
	var result []types.Monster
	for _, m := range cd.Monsters {
		if m.Tier == tier {
			result = append(result, m)
		}
	}
	return result
}

// GetLootByTier returns items appropriate for a loot tier.
func GetLootByTier(tier string, cd types.ContentData) []types.Item {
	var result []types.Item
	for _, item := range cd.Items {
		switch tier {
		case "minor":
			if item.Type == "gear" || item.Type == "trinket" {
				result = append(result, item)
			}
		case "standard":
			if item.Type == "weapon" || item.Type == "gear" {
				result = append(result, item)
			}
		case "good":
			if item.Type == "weapon" || item.Type == "armor" || item.Type == "shield" || item.Type == "spellbook" {
				result = append(result, item)
			}
		case "boss":
			if item.Type == "spellbook" || item.Type == "weapon" || (item.Type == "armor" && item.Slots == 2) {
				result = append(result, item)
			}
		}
	}
	return result
}

// RenderDescription renders a room description from features and fragment pool.
func RenderDescription(features []types.RoomFeature, fragments types.FragmentPool, r *rng.Rng) string {
	var parts []string
	for _, f := range features {
		pool, ok := fragments[f.Type]
		if !ok {
			continue
		}
		subtypePool, ok := pool[f.Subtype]
		if !ok || len(subtypePool) == 0 {
			continue
		}
		parts = append(parts, rng.Pick(r, subtypePool))
	}
	return strings.Join(parts, " ")
}
