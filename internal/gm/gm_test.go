package gm

import (
	"strings"
	"testing"

	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// --- BuildGmSystemPrompt tests ---

func TestBuildGmSystemPrompt_ContainsSchema(t *testing.T) {
	prompt := BuildGmSystemPrompt("", nil)
	if !strings.Contains(prompt, GmJSONSchema) {
		t.Error("expected system prompt to contain GmJSONSchema")
	}
}

func TestBuildGmSystemPrompt_DefaultPersonality(t *testing.T) {
	prompt := BuildGmSystemPrompt("", nil)
	if !strings.Contains(prompt, DefaultGMPersonality) {
		t.Error("expected system prompt to contain DefaultGMPersonality when none provided")
	}
}

func TestBuildGmSystemPrompt_CustomPersonality(t *testing.T) {
	custom := "You are a strict and unforgiving GM."
	prompt := BuildGmSystemPrompt(custom, nil)
	if !strings.Contains(prompt, custom) {
		t.Error("expected system prompt to contain custom personality")
	}
	if strings.Contains(prompt, DefaultGMPersonality) {
		t.Error("expected system prompt NOT to contain default personality when custom provided")
	}
}

func TestBuildGmSystemPrompt_NilExpeditionOk(t *testing.T) {
	// Should not panic when expedition is nil
	prompt := BuildGmSystemPrompt("", nil)
	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestBuildGmSystemPrompt_WithExpedition_ExploredNeighbor(t *testing.T) {
	// Build a simple 1x3 grid: rooms at (0,0), (1,0), (2,0)
	// Player at (0,0), east wall open, (1,0) visited
	roomA := &types.ExpeditionRoom{
		DungeonRoom: types.DungeonRoom{Name: "Dark Corridor"},
		Pos:         types.Pos{X: 0, Y: 0},
		Walls:       types.Walls{E: true},
	}
	roomB := &types.ExpeditionRoom{
		DungeonRoom: types.DungeonRoom{Name: "Torch Chamber"},
		Pos:         types.Pos{X: 1, Y: 0},
		Walls:       types.Walls{W: true},
	}
	grid := [][]*types.ExpeditionRoom{
		{roomA, roomB},
	}
	exp := &types.ExpeditionDungeon{
		Grid:      grid,
		PlayerPos: types.Pos{X: 0, Y: 0},
		Visited:   []string{"0,0", "1,0"},
	}
	prompt := BuildGmSystemPrompt("", exp)
	if !strings.Contains(prompt, "Torch Chamber") {
		t.Errorf("expected prompt to mention explored neighbor room name, got: %s", prompt)
	}
	if !strings.Contains(prompt, "east") {
		t.Errorf("expected prompt to mention direction 'east', got: %s", prompt)
	}
}

func TestBuildGmSystemPrompt_WithExpedition_UnexploredNeighbor(t *testing.T) {
	roomA := &types.ExpeditionRoom{
		DungeonRoom: types.DungeonRoom{Name: "Entry Hall"},
		Pos:         types.Pos{X: 0, Y: 0},
		Walls:       types.Walls{E: true},
	}
	roomB := &types.ExpeditionRoom{
		DungeonRoom: types.DungeonRoom{Name: "Hidden Chamber"},
		Pos:         types.Pos{X: 1, Y: 0},
		Walls:       types.Walls{W: true},
	}
	grid := [][]*types.ExpeditionRoom{
		{roomA, roomB},
	}
	exp := &types.ExpeditionDungeon{
		Grid:      grid,
		PlayerPos: types.Pos{X: 0, Y: 0},
		Visited:   []string{"0,0"}, // roomB not visited
	}
	prompt := BuildGmSystemPrompt("", exp)
	if strings.Contains(prompt, "Hidden Chamber") {
		t.Error("expected prompt NOT to reveal unexplored room name")
	}
	if !strings.Contains(prompt, "unexplored passage") {
		t.Errorf("expected prompt to mention 'unexplored passage', got: %s", prompt)
	}
}

// --- BuildCreativeActionPrompt tests ---

func makeTestMonster() types.MonsterInstance {
	return types.MonsterInstance{
		Base: types.Monster{
			Name:       "Goblin Shaman",
			Str:        8,
			Dex:        12,
			Wil:        14,
			HP:         6,
			Armor:      0,
			Weaknesses: []string{"fire", "holy"},
			Attack:     types.Attack{Name: "cursed bolt", Die: "d6"},
		},
		CurrentHP: 6,
	}
}

func makeTestCharacter() types.Character {
	return types.Character{
		Name: "Aldric",
		Inventory: []*types.Item{
			{ID: "torch", Name: "Torch", Traits: []string{"fire", "light"}},
			{ID: "sword", Name: "Iron Sword", Damage: "d6", Traits: []string{"sharp"}},
			nil, // test nil handling
		},
	}
}

func makeTestRoom() types.DungeonRoom {
	return types.DungeonRoom{
		Name:        "Fetid Crypt",
		Description: "Bones litter the floor.",
	}
}

func TestBuildCreativeActionPrompt_ContainsMonsterName(t *testing.T) {
	prompt := BuildCreativeActionPrompt(makeTestMonster(), makeTestCharacter(), makeTestRoom(), "I throw my torch at it!")
	if !strings.Contains(prompt, "Goblin Shaman") {
		t.Error("expected prompt to contain monster name")
	}
}

func TestBuildCreativeActionPrompt_ContainsWeaknesses(t *testing.T) {
	prompt := BuildCreativeActionPrompt(makeTestMonster(), makeTestCharacter(), makeTestRoom(), "attack")
	if !strings.Contains(prompt, "fire") || !strings.Contains(prompt, "holy") {
		t.Error("expected prompt to contain monster weaknesses")
	}
}

func TestBuildCreativeActionPrompt_NoWeaknesses(t *testing.T) {
	m := makeTestMonster()
	m.Base.Weaknesses = nil
	prompt := BuildCreativeActionPrompt(m, makeTestCharacter(), makeTestRoom(), "attack")
	if !strings.Contains(prompt, "none") {
		t.Error("expected 'none' for no weaknesses")
	}
}

func TestBuildCreativeActionPrompt_ContainsRoom(t *testing.T) {
	prompt := BuildCreativeActionPrompt(makeTestMonster(), makeTestCharacter(), makeTestRoom(), "attack")
	if !strings.Contains(prompt, "Fetid Crypt") {
		t.Error("expected prompt to contain room name")
	}
	if !strings.Contains(prompt, "Bones litter the floor.") {
		t.Error("expected prompt to contain room description")
	}
}

func TestBuildCreativeActionPrompt_ContainsInventory(t *testing.T) {
	prompt := BuildCreativeActionPrompt(makeTestMonster(), makeTestCharacter(), makeTestRoom(), "attack")
	if !strings.Contains(prompt, "Torch") {
		t.Error("expected prompt to contain item name 'Torch'")
	}
	if !strings.Contains(prompt, "Iron Sword") {
		t.Error("expected prompt to contain item name 'Iron Sword'")
	}
	if !strings.Contains(prompt, "d6") {
		t.Error("expected prompt to contain item damage 'd6'")
	}
	if !strings.Contains(prompt, "fire") {
		t.Error("expected prompt to contain item trait 'fire'")
	}
}

func TestBuildCreativeActionPrompt_ContainsActionText(t *testing.T) {
	action := "I smash the torch into its face!"
	prompt := BuildCreativeActionPrompt(makeTestMonster(), makeTestCharacter(), makeTestRoom(), action)
	if !strings.Contains(prompt, action) {
		t.Error("expected prompt to contain player action text")
	}
}

func TestBuildCreativeActionPrompt_DeduplicatesItems(t *testing.T) {
	char := types.Character{
		Inventory: []*types.Item{
			{ID: "torch", Name: "Torch", Traits: []string{"fire"}},
			{ID: "torch", Name: "Torch", Traits: []string{"fire"}}, // duplicate
		},
	}
	prompt := BuildCreativeActionPrompt(makeTestMonster(), char, makeTestRoom(), "attack")
	count := strings.Count(prompt, "- Torch")
	if count != 1 {
		t.Errorf("expected 'Torch' to appear once in inventory, got %d", count)
	}
}

// --- ValidateGmResponse tests ---

func validRawResponse() map[string]any {
	return map[string]any{
		"allowed":           true,
		"reason":            "The torch is a plausible fire source.",
		"save_stat":         "dex",
		"modifier":          float64(1),
		"trait_matches":     []any{"fire"},
		"effect":            "damage",
		"effect_die":        "d8",
		"consumes_item":     "torch",
		"success_narration": "The goblin catches fire and shrieks.",
		"failure_narration": "You miss and the torch gutters out.",
	}
}

func TestValidateGmResponse_ValidParsesCorrectly(t *testing.T) {
	result, err := ValidateGmResponse(validRawResponse())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected Allowed=true")
	}
	if result.SaveStat != "dex" {
		t.Errorf("expected save_stat='dex', got '%s'", result.SaveStat)
	}
	if result.Modifier != 1 {
		t.Errorf("expected modifier=1, got %d", result.Modifier)
	}
	if result.Effect != "damage" {
		t.Errorf("expected effect='damage', got '%s'", result.Effect)
	}
	if result.EffectDie != "d8" {
		t.Errorf("expected effect_die='d8', got '%s'", result.EffectDie)
	}
	if result.ConsumesItem == nil || *result.ConsumesItem != "torch" {
		t.Errorf("expected consumes_item='torch'")
	}
	if len(result.TraitMatches) != 1 || result.TraitMatches[0] != "fire" {
		t.Errorf("expected trait_matches=['fire'], got %v", result.TraitMatches)
	}
	if result.Reason != "The torch is a plausible fire source." {
		t.Errorf("unexpected reason: %s", result.Reason)
	}
	if result.SuccessNarration != "The goblin catches fire and shrieks." {
		t.Errorf("unexpected success_narration: %s", result.SuccessNarration)
	}
	if result.FailureNarration != "You miss and the torch gutters out." {
		t.Errorf("unexpected failure_narration: %s", result.FailureNarration)
	}
}

func TestValidateGmResponse_MissingAllowed_ReturnsError(t *testing.T) {
	raw := validRawResponse()
	delete(raw, "allowed")
	_, err := ValidateGmResponse(raw)
	if err == nil {
		t.Error("expected error for missing 'allowed' field")
	}
}

func TestValidateGmResponse_InvalidAllowedType_ReturnsError(t *testing.T) {
	raw := validRawResponse()
	raw["allowed"] = "yes" // wrong type
	_, err := ValidateGmResponse(raw)
	if err == nil {
		t.Error("expected error for invalid 'allowed' type")
	}
}

func TestValidateGmResponse_InvalidSaveStat_ReturnsError(t *testing.T) {
	raw := validRawResponse()
	raw["save_stat"] = "luck"
	_, err := ValidateGmResponse(raw)
	if err == nil {
		t.Error("expected error for invalid save_stat")
	}
}

func TestValidateGmResponse_InvalidEffect_ReturnsError(t *testing.T) {
	raw := validRawResponse()
	raw["effect"] = "explode"
	_, err := ValidateGmResponse(raw)
	if err == nil {
		t.Error("expected error for invalid effect")
	}
}

func TestValidateGmResponse_ModifierClampedHigh(t *testing.T) {
	raw := validRawResponse()
	raw["modifier"] = float64(10)
	result, err := ValidateGmResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Modifier != 2 {
		t.Errorf("expected modifier clamped to 2, got %d", result.Modifier)
	}
}

func TestValidateGmResponse_ModifierClampedLow(t *testing.T) {
	raw := validRawResponse()
	raw["modifier"] = float64(-10)
	result, err := ValidateGmResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Modifier != -2 {
		t.Errorf("expected modifier clamped to -2, got %d", result.Modifier)
	}
}

func TestValidateGmResponse_InvalidEffectDie_DefaultsToD6(t *testing.T) {
	raw := validRawResponse()
	raw["effect_die"] = "d100"
	result, err := ValidateGmResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EffectDie != "d6" {
		t.Errorf("expected effect_die to default to 'd6', got '%s'", result.EffectDie)
	}
}

func TestValidateGmResponse_MissingEffectDie_DefaultsToD6(t *testing.T) {
	raw := validRawResponse()
	delete(raw, "effect_die")
	result, err := ValidateGmResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EffectDie != "d6" {
		t.Errorf("expected effect_die to default to 'd6', got '%s'", result.EffectDie)
	}
}

func TestValidateGmResponse_NullConsumesItem(t *testing.T) {
	raw := validRawResponse()
	raw["consumes_item"] = nil
	result, err := ValidateGmResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ConsumesItem != nil {
		t.Error("expected ConsumesItem to be nil when JSON value is null")
	}
}

func TestValidateGmResponse_AllowedFalse(t *testing.T) {
	raw := validRawResponse()
	raw["allowed"] = false
	result, err := ValidateGmResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected Allowed=false")
	}
}

// --- VerifyTraitMatches tests ---

func TestVerifyTraitMatches_ReturnsIntersection(t *testing.T) {
	char := types.Character{
		Inventory: []*types.Item{
			{ID: "torch", Name: "Torch", Traits: []string{"fire", "light"}},
			{ID: "vial", Name: "Holy Water", Traits: []string{"holy", "water"}},
		},
	}
	monster := types.MonsterInstance{
		Base: types.Monster{
			Weaknesses: []string{"fire", "holy"},
		},
	}
	claimed := []string{"fire", "holy", "light"}
	verified := VerifyTraitMatches(claimed, char, monster)
	if len(verified) != 2 {
		t.Errorf("expected 2 verified traits, got %d: %v", len(verified), verified)
	}
	// "light" should be excluded — player has it, but monster is not weak to it
	for _, v := range verified {
		if v == "light" {
			t.Error("'light' should not be verified — monster is not weak to it")
		}
	}
}

func TestVerifyTraitMatches_ExcludesTraitPlayerLacks(t *testing.T) {
	char := types.Character{
		Inventory: []*types.Item{
			{ID: "sword", Name: "Iron Sword", Traits: []string{"sharp"}},
		},
	}
	monster := types.MonsterInstance{
		Base: types.Monster{
			Weaknesses: []string{"fire", "sharp"},
		},
	}
	claimed := []string{"fire", "sharp"}
	verified := VerifyTraitMatches(claimed, char, monster)
	if len(verified) != 1 || verified[0] != "sharp" {
		t.Errorf("expected only 'sharp' to be verified, got %v", verified)
	}
}

func TestVerifyTraitMatches_EmptyClaimed(t *testing.T) {
	char := makeTestCharacter()
	monster := makeTestMonster()
	verified := VerifyTraitMatches([]string{}, char, monster)
	if len(verified) != 0 {
		t.Errorf("expected empty result for empty claimed, got %v", verified)
	}
}

func TestVerifyTraitMatches_NilInventoryItems(t *testing.T) {
	char := types.Character{
		Inventory: []*types.Item{nil, nil},
	}
	monster := types.MonsterInstance{
		Base: types.Monster{Weaknesses: []string{"fire"}},
	}
	// Should not panic
	verified := VerifyTraitMatches([]string{"fire"}, char, monster)
	if len(verified) != 0 {
		t.Errorf("expected no verified traits with nil inventory items, got %v", verified)
	}
}

func TestVerifyTraitMatches_NoWeaknesses(t *testing.T) {
	char := makeTestCharacter()
	monster := types.MonsterInstance{
		Base: types.Monster{Weaknesses: nil},
	}
	verified := VerifyTraitMatches([]string{"fire", "holy"}, char, monster)
	if len(verified) != 0 {
		t.Errorf("expected no verified traits when monster has no weaknesses, got %v", verified)
	}
}
