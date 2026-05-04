package content

import (
	"testing"

	"github.com/mph-llm-experiments/tinycrawl-tui/internal/rng"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

func TestLoadEmbeddedPacks(t *testing.T) {
	packs, err := LoadEmbeddedPacks()
	if err != nil {
		t.Fatalf("LoadEmbeddedPacks: %v", err)
	}
	if len(packs) < 5 {
		t.Fatalf("expected at least 5 packs, got %d", len(packs))
	}
	found := false
	for _, p := range packs {
		if p.Meta.Name == "The Dark Below" {
			found = true
			if len(p.Monsters) == 0 {
				t.Fatal("default pack has no monsters")
			}
			if len(p.Items) == 0 {
				t.Fatal("default pack has no items")
			}
			if len(p.Rooms) == 0 {
				t.Fatal("default pack has no rooms")
			}
			if len(p.Names) == 0 {
				t.Fatal("default pack has no names")
			}
		}
	}
	if !found {
		t.Fatal("default pack 'The Dark Below' not found")
	}
}

func TestContentFromPack(t *testing.T) {
	packs, _ := LoadEmbeddedPacks()
	cd := ContentFromPack(packs[0])
	if len(cd.Monsters) == 0 {
		t.Fatal("no monsters")
	}
	if len(cd.Items) == 0 {
		t.Fatal("no items")
	}
}

func TestGetMonstersByTier(t *testing.T) {
	packs, _ := LoadEmbeddedPacks()
	cd := ContentFromPack(packs[0])
	weak := GetMonstersByTier("weak", cd)
	if len(weak) == 0 {
		t.Fatal("no weak monsters found")
	}
	for _, m := range weak {
		if m.Tier != "weak" {
			t.Fatalf("expected tier 'weak', got %q", m.Tier)
		}
	}
}

func TestGetLootByTier(t *testing.T) {
	packs, _ := LoadEmbeddedPacks()
	cd := ContentFromPack(packs[0])
	minor := GetLootByTier("minor", cd)
	if len(minor) == 0 {
		t.Fatal("no minor loot found")
	}
	for _, item := range minor {
		if item.Type != "gear" && item.Type != "trinket" {
			t.Fatalf("minor loot should be gear/trinket, got %q", item.Type)
		}
	}
}

func TestRenderDescription(t *testing.T) {
	r := rng.New(42)
	fragments := types.FragmentPool{
		"architecture": {
			"stone": []string{"Cold stone walls drip with moisture."},
		},
	}
	features := []types.RoomFeature{{Type: "architecture", Subtype: "stone"}}
	desc := RenderDescription(features, fragments, r)
	if desc != "Cold stone walls drip with moisture." {
		t.Fatalf("unexpected description: %q", desc)
	}
}
