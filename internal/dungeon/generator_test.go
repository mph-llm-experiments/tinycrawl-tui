package dungeon

import (
	"testing"

	"github.com/mph-llm-experiments/tinycrawl-tui/internal/rng"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// minimalContentData returns a types.ContentData with enough content to run generation.
func minimalContentData() types.ContentData {
	return types.ContentData{
		Monsters: []types.Monster{
			{ID: "rat", Name: "Rat", Tier: "weak", HP: 3, Str: 8, Dex: 10, Wil: 6, Attack: types.Attack{Name: "bite", Die: "d4"}},
			{ID: "goblin", Name: "Goblin", Tier: "moderate", HP: 5, Str: 10, Dex: 12, Wil: 8, Attack: types.Attack{Name: "stab", Die: "d6"}},
			{ID: "orc", Name: "Orc", Tier: "tough", HP: 8, Str: 14, Dex: 8, Wil: 8, Attack: types.Attack{Name: "axe", Die: "d8"}},
			{ID: "dragon", Name: "Dragon", Tier: "boss", HP: 20, Str: 18, Dex: 10, Wil: 16, Attack: types.Attack{Name: "bite", Die: "d12"}},
		},
		Items: []types.Item{
			{ID: "torch", Name: "Torch", Type: "gear", Slots: 1, UseEffect: &types.UseEffect{Type: "restore_light", Level: 3}},
			{ID: "sword", Name: "Sword", Type: "weapon", Slots: 1, Damage: "d6"},
		},
		Rooms: []types.RoomTemplate{
			{
				ID:         "corridor",
				Name:       "Corridor",
				Features:   []types.RoomFeature{},
				Encounters: []types.EncounterDef{{Type: "monster", Chance: 0.5}},
				Loot:       &types.LootDef{Tier: "minor", Chance: 0.3},
			},
			{
				ID:         "chamber",
				Name:       "Chamber",
				Features:   []types.RoomFeature{},
				Encounters: []types.EncounterDef{{Type: "trap", Chance: 0.3}},
				Loot:       &types.LootDef{Tier: "standard", Chance: 0.4},
			},
		},
		Fragments: types.FragmentPool{},
	}
}

func defaultConfig() types.ExpeditionConfig {
	return types.ExpeditionConfig{
		RoomBudget:           [2]int{8, 12},
		BranchProbability:    0.4,
		LoopProbability:      0.1,
		DeadEndRatio:         0.3,
		LightSourceFrequency: 0.2,
		GridSize:             11,
	}
}

func TestGenerateExpeditionGrid(t *testing.T) {
	r := rng.New(42)
	cd := minimalContentData()
	config := defaultConfig()

	result := GenerateExpeditionGrid(r, cd, config)

	// Entry room must exist
	entryRoom := GetRoom(result.Grid, result.Entry)
	if entryRoom == nil {
		t.Fatal("entry room is nil")
	}
	if entryRoom.Pos != result.Entry {
		t.Fatalf("entry room pos mismatch: got %v, want %v", entryRoom.Pos, result.Entry)
	}

	// Boss room must exist
	bossRoom := GetRoom(result.Grid, result.BossPos)
	if bossRoom == nil {
		t.Fatal("boss room is nil")
	}

	// Room count within budget
	minR, maxR := config.RoomBudget[0], config.RoomBudget[1]
	if result.RoomCount < minR || result.RoomCount > maxR {
		t.Fatalf("room count %d outside budget [%d, %d]", result.RoomCount, minR, maxR)
	}

	// All rooms reachable from entry via BFS over open walls
	distances := computeDistances(result.Grid, result.Entry)
	if len(distances) != result.RoomCount {
		t.Fatalf("BFS reached %d rooms, expected %d (not all rooms connected)", len(distances), result.RoomCount)
	}

	// Entry room should have no encounter (it's the starting room)
	if entryRoom.Encounter != nil {
		t.Fatal("entry room should have no encounter")
	}
}

func TestGetTierForDistance(t *testing.T) {
	tests := []struct {
		dist     int
		maxDist  int
		expected string
	}{
		{0, 0, "weak"},   // maxDist == 0 edge case
		{0, 10, "weak"},  // 0%
		{2, 10, "weak"},  // 20%
		{3, 10, "moderate"}, // 30%
		{5, 10, "moderate"}, // 50%
		{6, 10, "tough"}, // 60%
		{8, 10, "tough"}, // 80%
		{9, 10, "boss"},  // 90%
		{10, 10, "boss"}, // 100%
	}

	for _, tc := range tests {
		got := getTierForDistance(tc.dist, tc.maxDist)
		if got != tc.expected {
			t.Errorf("getTierForDistance(%d, %d) = %q, want %q", tc.dist, tc.maxDist, got, tc.expected)
		}
	}
}

func TestComputeDistances(t *testing.T) {
	// Use our 3x3 hand-crafted grid from expedition tests
	// Layout: r11(1,1) is center connected to r01(1,0), r10(0,1), r12(2,1), r21(1,2)
	grid := makeTestGrid()
	entry := types.Pos{X: 1, Y: 1}

	distances := computeDistances(grid, entry)

	// Should reach all 5 rooms
	if len(distances) != 5 {
		t.Fatalf("expected 5 distances, got %d: %v", len(distances), distances)
	}

	// Entry should be distance 0
	entryKey := PosKey(entry)
	if distances[entryKey] != 0 {
		t.Fatalf("entry distance should be 0, got %d", distances[entryKey])
	}

	// Direct neighbors should be distance 1
	neighbors := []types.Pos{
		{X: 1, Y: 0}, // r01
		{X: 0, Y: 1}, // r10
		{X: 2, Y: 1}, // r12
		{X: 1, Y: 2}, // r21
	}
	for _, n := range neighbors {
		key := PosKey(n)
		d, ok := distances[key]
		if !ok {
			t.Fatalf("neighbor %v not found in distances", n)
		}
		if d != 1 {
			t.Errorf("neighbor %v should have distance 1, got %d", n, d)
		}
	}
}
