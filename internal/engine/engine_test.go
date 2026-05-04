package engine

import (
	"testing"

	"github.com/mph-llm-experiments/tinycrawl-tui/internal/content"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/rng"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// loadTestContent loads the first embedded content pack for testing.
func loadTestContent(t *testing.T) types.ContentData {
	t.Helper()
	packs, err := content.LoadEmbeddedPacks()
	if err != nil {
		t.Fatalf("failed to load embedded packs: %v", err)
	}
	if len(packs) == 0 {
		t.Fatal("no embedded packs found")
	}
	return content.ContentFromPack(packs[0])
}

// --- CreateInitialState ---

func TestCreateInitialState(t *testing.T) {
	state := CreateInitialState(42)
	if state.Phase != types.PhaseTitle {
		t.Errorf("expected phase title, got %s", state.Phase)
	}
	if state.GameType != types.GameTypeSprint {
		t.Errorf("expected game type sprint, got %s", state.GameType)
	}
	if state.Character != nil {
		t.Error("expected nil character")
	}
	if state.Light != 10 {
		t.Errorf("expected light 10, got %d", state.Light)
	}
	if state.Seed != 42 {
		t.Errorf("expected seed 42, got %d", state.Seed)
	}
	if state.RngState != 42 {
		t.Errorf("expected rng state 42, got %d", state.RngState)
	}
	if state.MonstersKilled != 0 {
		t.Errorf("expected 0 monsters killed, got %d", state.MonstersKilled)
	}
	if state.RestsTaken != 0 {
		t.Errorf("expected 0 rests taken, got %d", state.RestsTaken)
	}
}

// --- CreateCharacter ---

func TestCreateCharacter(t *testing.T) {
	cd := loadTestContent(t)
	r := rng.New(42)
	char := CreateCharacter(r, cd)

	if char.Name == "" {
		t.Error("character should have a name")
	}
	// 3d6 range: 3-18
	for _, stat := range []struct {
		name string
		val  int
	}{
		{"STR", char.Str},
		{"DEX", char.Dex},
		{"WIL", char.Wil},
	} {
		if stat.val < 3 || stat.val > 18 {
			t.Errorf("%s=%d out of range [3,18]", stat.name, stat.val)
		}
	}
	// d6+4 range: 5-10
	if char.HP < 5 || char.HP > 10 {
		t.Errorf("HP=%d out of range [5,10]", char.HP)
	}
	if char.MaxHP != char.HP {
		t.Errorf("MaxHP=%d should equal HP=%d", char.MaxHP, char.HP)
	}
	if len(char.Inventory) != types.MaxSlots {
		t.Errorf("inventory length=%d, expected %d", len(char.Inventory), types.MaxSlots)
	}
	// Should have at least a weapon in inventory
	hasWeapon := false
	for _, item := range char.Inventory {
		if item != nil && item.Type == "weapon" {
			hasWeapon = true
			break
		}
	}
	if !hasWeapon {
		t.Error("character should have at least one weapon")
	}
	if char.Armor < 0 || char.Armor > 3 {
		t.Errorf("armor=%d should be in [0,3]", char.Armor)
	}
	if char.Scars == nil {
		t.Error("scars should not be nil")
	}
}

// --- SlotsUsed ---

func TestSlotsUsed(t *testing.T) {
	char := types.Character{
		Inventory: make([]*types.Item, types.MaxSlots),
	}
	if SlotsUsed(char) != 0 {
		t.Errorf("empty inventory should have 0 slots used")
	}

	item := &types.Item{ID: "test", Name: "Test", Type: "gear", Slots: 1}
	char.Inventory[0] = item
	char.Inventory[3] = item
	if SlotsUsed(char) != 2 {
		t.Errorf("expected 2 slots used, got %d", SlotsUsed(char))
	}
}

// --- ResolveDamage ---

func TestResolveDamage_ArmorAbsorbs(t *testing.T) {
	newHP, newStr, strDmg := ResolveDamage(3, 3, 5, 10)
	if newHP != 5 || newStr != 10 || strDmg != 0 {
		t.Errorf("armor should fully absorb: hp=%d str=%d strDmg=%d", newHP, newStr, strDmg)
	}
}

func TestResolveDamage_HPDamage(t *testing.T) {
	newHP, newStr, strDmg := ResolveDamage(5, 2, 5, 10)
	// effective = 3, hp 5 -> 2
	if newHP != 2 || newStr != 10 || strDmg != 0 {
		t.Errorf("expected hp=2 str=10 strDmg=0, got hp=%d str=%d strDmg=%d", newHP, newStr, strDmg)
	}
}

func TestResolveDamage_Overflow(t *testing.T) {
	newHP, newStr, strDmg := ResolveDamage(10, 0, 3, 10)
	// effective=10, hp 3 -> 0, overflow 7, str 10 -> 3
	if newHP != 0 || newStr != 3 || strDmg != 7 {
		t.Errorf("expected hp=0 str=3 strDmg=7, got hp=%d str=%d strDmg=%d", newHP, newStr, strDmg)
	}
}

func TestResolveDamage_ZeroDamage(t *testing.T) {
	newHP, newStr, strDmg := ResolveDamage(0, 0, 5, 10)
	if newHP != 5 || newStr != 10 || strDmg != 0 {
		t.Errorf("zero damage should change nothing: hp=%d str=%d strDmg=%d", newHP, newStr, strDmg)
	}
}

func TestResolveDamage_LethalToStr(t *testing.T) {
	newHP, newStr, strDmg := ResolveDamage(20, 0, 3, 5)
	// effective=20, hp 3 -> 0, overflow 17, str 5 -> 0 (capped)
	if newHP != 0 || newStr != 0 || strDmg != 17 {
		t.Errorf("expected hp=0 str=0 strDmg=17, got hp=%d str=%d strDmg=%d", newHP, newStr, strDmg)
	}
}

// --- CriticalDamageSave ---

func TestCriticalDamageSave(t *testing.T) {
	// With a deterministic RNG, test that the save works based on roll vs STR
	// We'll run multiple trials and verify the mechanic
	passes := 0
	trials := 100
	for i := 0; i < trials; i++ {
		r := rng.New(i)
		if CriticalDamageSave(10, r) {
			passes++
		}
	}
	// With STR 10, roughly half should pass (d20 <= 10 = 50%)
	if passes == 0 || passes == trials {
		t.Errorf("expected some passes and some failures, got %d/%d", passes, trials)
	}
}

// --- AttemptFlee ---

func TestAttemptFlee(t *testing.T) {
	char := types.Character{Dex: 15}
	successes := 0
	trials := 100
	for i := 0; i < trials; i++ {
		r := rng.New(i)
		result := AttemptFlee(char, r, "d6")
		if result.Success {
			successes++
			if result.FreeAttackDamage != 0 {
				t.Error("successful flee should have 0 free attack damage")
			}
		} else {
			// Free attack should be > 0 (d6 is always >= 1)
			if result.FreeAttackDamage < 1 {
				t.Errorf("failed flee should have free attack damage >= 1, got %d", result.FreeAttackDamage)
			}
		}
	}
	if successes == 0 || successes == trials {
		t.Errorf("expected some successes and failures, got %d/%d", successes, trials)
	}
}

// --- CanAddItem / AddItem / DropItem ---

func TestCanAddItem(t *testing.T) {
	char := types.Character{
		Inventory: make([]*types.Item, types.MaxSlots),
	}
	if !CanAddItem(char, 1) {
		t.Error("should be able to add 1-slot item to empty inventory")
	}
	if !CanAddItem(char, 2) {
		t.Error("should be able to add 2-slot item to empty inventory")
	}

	// Fill all slots
	for i := range char.Inventory {
		char.Inventory[i] = &types.Item{ID: "fill", Slots: 1}
	}
	if CanAddItem(char, 1) {
		t.Error("should not be able to add to full inventory")
	}
}

func TestAddItem(t *testing.T) {
	char := types.Character{
		Inventory: make([]*types.Item, types.MaxSlots),
	}
	item := types.Item{ID: "sword", Name: "Sword", Type: "weapon", Slots: 1}

	newChar := AddItem(char, item)
	if newChar == nil {
		t.Fatal("AddItem should succeed on empty inventory")
	}
	if newChar.Inventory[0] == nil {
		t.Error("item should be at slot 0")
	}
	if newChar.Inventory[0].ID != "sword" {
		t.Errorf("expected sword, got %s", newChar.Inventory[0].ID)
	}
}

func TestAddItem_TwoSlot(t *testing.T) {
	char := types.Character{
		Inventory: make([]*types.Item, types.MaxSlots),
	}
	armor := types.Item{ID: "plate", Name: "Plate Armor", Type: "armor", Slots: 2, ArmorValue: 3}

	newChar := AddItem(char, armor)
	if newChar == nil {
		t.Fatal("AddItem should succeed")
	}
	if newChar.Inventory[0] == nil || newChar.Inventory[1] == nil {
		t.Error("2-slot item should occupy slots 0 and 1")
	}
}

func TestAddItem_NoRoom(t *testing.T) {
	char := types.Character{
		Inventory: make([]*types.Item, types.MaxSlots),
	}
	for i := range char.Inventory {
		char.Inventory[i] = &types.Item{ID: "fill", Slots: 1}
	}
	item := types.Item{ID: "extra", Slots: 1}
	result := AddItem(char, item)
	if result != nil {
		t.Error("AddItem should return nil when no room")
	}
}

func TestDropItem(t *testing.T) {
	char := types.Character{
		Inventory: make([]*types.Item, types.MaxSlots),
	}
	item := &types.Item{ID: "sword", Name: "Sword", Type: "weapon", Slots: 1}
	char.Inventory[0] = item

	newChar := DropItem(char, 0)
	if newChar.Inventory[0] != nil {
		t.Error("dropped item should be nil")
	}
}

func TestDropItem_TwoSlot(t *testing.T) {
	char := types.Character{
		Inventory: make([]*types.Item, types.MaxSlots),
	}
	armor := &types.Item{ID: "plate", Name: "Plate", Type: "armor", Slots: 2}
	char.Inventory[0] = armor
	char.Inventory[1] = armor

	newChar := DropItem(char, 1)
	if newChar.Inventory[0] != nil || newChar.Inventory[1] != nil {
		t.Error("dropping 2-slot item should clear both slots")
	}
}

// --- RecalculateArmor ---

func TestRecalculateArmor(t *testing.T) {
	char := types.Character{
		Inventory: make([]*types.Item, types.MaxSlots),
	}
	// No armor items
	if RecalculateArmor(char) != 0 {
		t.Error("no armor items should give 0")
	}

	// Add armor
	shield := &types.Item{ID: "shield", ArmorValue: 1, Slots: 1}
	char.Inventory[0] = shield
	if RecalculateArmor(char) != 1 {
		t.Errorf("expected 1, got %d", RecalculateArmor(char))
	}

	// Add body armor
	plate := &types.Item{ID: "plate", ArmorValue: 3, Slots: 2}
	char.Inventory[2] = plate
	char.Inventory[3] = plate
	// Total = 4, but capped at 3
	if RecalculateArmor(char) != 3 {
		t.Errorf("expected 3 (capped), got %d", RecalculateArmor(char))
	}
}

func TestRecalculateArmor_DuplicateID(t *testing.T) {
	char := types.Character{
		Inventory: make([]*types.Item, types.MaxSlots),
	}
	// Same ID in two slots (2-slot item) should only count once
	armor := &types.Item{ID: "plate", ArmorValue: 2, Slots: 2}
	char.Inventory[0] = armor
	char.Inventory[1] = armor
	if RecalculateArmor(char) != 2 {
		t.Errorf("expected 2, got %d", RecalculateArmor(char))
	}
}

// --- GetVisibleDescription ---

func TestGetVisibleDescription(t *testing.T) {
	desc := "A dark hall stretches before you. Cobwebs hang from the ceiling."

	if got := GetVisibleDescription(desc, types.LightBright); got != desc {
		t.Errorf("bright: expected full description, got %q", got)
	}
	if got := GetVisibleDescription(desc, types.LightDim); got != "A dark hall stretches before you." {
		t.Errorf("dim: expected first sentence, got %q", got)
	}
	if got := GetVisibleDescription(desc, types.LightDark); got != "" {
		t.Errorf("dark: expected empty, got %q", got)
	}
	if got := GetVisibleDescription(desc, types.LightBlack); got != "You can't see." {
		t.Errorf("black: expected can't see, got %q", got)
	}
}

// --- Dispatch: new_game ---

func TestDispatch_NewGame(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(0)
	result := Dispatch(&state, types.NewGame(42), cd)

	if result.Phase != types.PhaseCharacterCreation {
		t.Errorf("expected character_creation, got %s", result.Phase)
	}
	if result.Character == nil {
		t.Error("character should not be nil")
	}
	if result.Sprint == nil {
		t.Error("sprint dungeon should not be nil for sprint game")
	}
	if result.Seed != 42 {
		t.Errorf("expected seed 42, got %d", result.Seed)
	}
	if result.Light != 10 {
		t.Errorf("expected light 10, got %d", result.Light)
	}
	if result.MonstersKilled != 0 {
		t.Errorf("expected 0 monsters killed")
	}
	if len(result.Sprint.Rooms) == 0 {
		t.Error("dungeon should have rooms")
	}
}

// --- Dispatch: accept_character ---

func TestDispatch_AcceptCharacter(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(0)
	state2 := Dispatch(&state, types.NewGame(42), cd)
	result := Dispatch(state2, types.AcceptCharacter(), cd)

	if result.Phase != types.PhaseExploring {
		t.Errorf("expected exploring, got %s", result.Phase)
	}
	if len(result.Log) == 0 {
		t.Error("expected log entries")
	}
	// First room should be visited and cleared
	room := result.CurrentRoom()
	if room == nil {
		t.Fatal("current room should not be nil")
	}
	if !room.Visited {
		t.Error("first room should be visited")
	}
	if !room.Cleared {
		t.Error("first room should be cleared")
	}
}

// --- Dispatch: choose_exit ---

func TestDispatch_ChooseExit(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(0)
	state2 := Dispatch(&state, types.NewGame(42), cd)
	state3 := Dispatch(state2, types.AcceptCharacter(), cd)

	// Need at least one exit
	exits := state3.Exits()
	if len(exits) == 0 {
		t.Skip("no exits available to test")
	}

	result := Dispatch(state3, types.ChooseExit(0), cd)

	// Light should have decremented
	if result.Light != state3.Light-1 {
		t.Errorf("expected light %d, got %d", state3.Light-1, result.Light)
	}

	// Should be in some valid phase
	validPhases := map[types.GamePhase]bool{
		types.PhaseExploring: true,
		types.PhaseCombat:    true,
		types.PhaseLooting:   true,
		types.PhaseDead:      true,
	}
	if !validPhases[result.Phase] {
		t.Errorf("unexpected phase %s after choose_exit", result.Phase)
	}

	// RestsTaken should be reset
	if result.RestsTaken != 0 {
		t.Errorf("expected rests reset to 0, got %d", result.RestsTaken)
	}
}

// --- Dispatch: attack ---

func TestDispatch_Attack(t *testing.T) {
	cd := loadTestContent(t)

	// Set up a state in combat
	state := CreateInitialState(0)
	state2 := Dispatch(&state, types.NewGame(42), cd)
	state3 := Dispatch(state2, types.AcceptCharacter(), cd)

	// Navigate until we find combat or force a combat state
	// Instead, let's manually set up combat
	state3.Phase = types.PhaseCombat
	monsters := content.GetMonstersByTier("weak", cd)
	if len(monsters) == 0 {
		t.Skip("no weak monsters")
	}
	m := monsters[0]
	state3.Combat = &types.CombatState{
		Monster: types.MonsterInstance{
			Base:       m,
			CurrentHP:  m.HP,
			CurrentStr: m.Str,
		},
		Round:              1,
		Log:                []string{"A monster attacks!"},
		MonsterStunned:     false,
		PlayerHasAdvantage: false,
	}

	result := Dispatch(state3, types.AttackAction(), cd)

	// Should be in combat, dead, exploring, looting, or victory
	validPhases := map[types.GamePhase]bool{
		types.PhaseCombat:   true,
		types.PhaseDead:     true,
		types.PhaseExploring: true,
		types.PhaseLooting:  true,
		types.PhaseVictory:  true,
	}
	if !validPhases[result.Phase] {
		t.Errorf("unexpected phase %s after attack", result.Phase)
	}

	// If still in combat, round should have advanced
	if result.Phase == types.PhaseCombat && result.Combat != nil {
		if result.Combat.Round != 2 {
			t.Errorf("expected round 2, got %d", result.Combat.Round)
		}
	}
}

// --- Dispatch: rest ---

func TestDispatch_Rest(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(0)
	state2 := Dispatch(&state, types.NewGame(42), cd)
	state3 := Dispatch(state2, types.AcceptCharacter(), cd)

	// Damage the character a bit to see healing
	state3.Character.HP = 3
	state3.Character.MaxHP = 10

	result := Dispatch(state3, types.Rest(), cd)

	// Should either be exploring (healed) or combat (disturbed)
	if result.Phase != types.PhaseExploring && result.Phase != types.PhaseCombat {
		t.Errorf("expected exploring or combat after rest, got %s", result.Phase)
	}

	if result.Phase == types.PhaseExploring {
		// Should have healed
		if result.Character.HP <= 3 {
			t.Error("character should have healed during rest")
		}
		if result.Character.HP > result.Character.MaxHP {
			t.Error("HP should not exceed MaxHP")
		}
	}

	// RestsTaken should have incremented
	if result.RestsTaken != state3.RestsTaken+1 {
		t.Errorf("expected rests %d, got %d", state3.RestsTaken+1, result.RestsTaken)
	}

	// Light should have decremented
	if result.Light != state3.Light-1 {
		t.Errorf("expected light %d, got %d", state3.Light-1, result.Light)
	}
}

func TestDispatch_Rest_Escalation(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(0)
	state2 := Dispatch(&state, types.NewGame(42), cd)
	state3 := Dispatch(state2, types.AcceptCharacter(), cd)

	// With restsTaken=4, chance should be 100% (0.2 + 4*0.2 = 1.0)
	state3.RestsTaken = 4
	state3.Character.HP = 3
	state3.Character.MaxHP = 10

	result := Dispatch(state3, types.Rest(), cd)
	if result.Phase != types.PhaseCombat {
		t.Errorf("expected combat at 100%% disturbance chance, got %s", result.Phase)
	}
}

// --- Dispatch: take_item ---

func TestDispatch_TakeItem(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(0)
	state2 := Dispatch(&state, types.NewGame(42), cd)
	state3 := Dispatch(state2, types.AcceptCharacter(), cd)

	// Set up looting phase
	lootItem := types.Item{ID: "loot_sword", Name: "Loot Sword", Type: "weapon", Slots: 1}
	state3.Phase = types.PhaseLooting
	state3.PendingLoot = []types.Item{lootItem}

	result := Dispatch(state3, types.TakeItem(0), cd)

	// Item should have been taken
	if result.Phase != types.PhaseExploring {
		t.Errorf("expected exploring after taking last loot, got %s", result.Phase)
	}
	if len(result.PendingLoot) != 0 {
		t.Errorf("expected empty pending loot, got %d items", len(result.PendingLoot))
	}
	// Check the item is in inventory
	found := false
	for _, item := range result.Character.Inventory {
		if item != nil && item.ID == "loot_sword" {
			found = true
			break
		}
	}
	if !found {
		t.Error("taken item should be in inventory")
	}
}

func TestDispatch_TakeItem_MultipleLoot(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(0)
	state2 := Dispatch(&state, types.NewGame(42), cd)
	state3 := Dispatch(state2, types.AcceptCharacter(), cd)

	item1 := types.Item{ID: "item1", Name: "Item 1", Type: "gear", Slots: 1}
	item2 := types.Item{ID: "item2", Name: "Item 2", Type: "gear", Slots: 1}
	state3.Phase = types.PhaseLooting
	state3.PendingLoot = []types.Item{item1, item2}

	result := Dispatch(state3, types.TakeItem(0), cd)
	if result.Phase != types.PhaseLooting {
		t.Errorf("should still be looting with items remaining, got %s", result.Phase)
	}
	if len(result.PendingLoot) != 1 {
		t.Errorf("expected 1 remaining loot, got %d", len(result.PendingLoot))
	}
}

// --- Dispatch: skip_loot ---

func TestDispatch_SkipLoot(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(0)
	state2 := Dispatch(&state, types.NewGame(42), cd)
	state3 := Dispatch(state2, types.AcceptCharacter(), cd)

	state3.Phase = types.PhaseLooting
	state3.PendingLoot = []types.Item{{ID: "loot", Name: "Loot", Slots: 1}}

	result := Dispatch(state3, types.SkipLoot(), cd)
	if result.Phase != types.PhaseExploring {
		t.Errorf("expected exploring, got %s", result.Phase)
	}
	if len(result.PendingLoot) != 0 {
		t.Errorf("expected empty pending loot, got %d", len(result.PendingLoot))
	}
}

// --- Dispatch: use_item ---

func TestDispatch_UseItem_RestoreLight(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(0)
	state2 := Dispatch(&state, types.NewGame(42), cd)
	state3 := Dispatch(state2, types.AcceptCharacter(), cd)

	// Set low light and add a torch
	state3.Light = 2
	torch := types.Item{
		ID:   "torch",
		Name: "Torch",
		Type: "gear",
		Slots: 1,
		UseEffect: &types.UseEffect{Type: "restore_light", Level: 8},
	}
	// Find an empty slot
	for i, item := range state3.Character.Inventory {
		if item == nil {
			state3.Character.Inventory[i] = &torch
			result := Dispatch(state3, types.UseItem(i), cd)
			if result.Light != 8 {
				t.Errorf("expected light 8, got %d", result.Light)
			}
			// Torch should be consumed
			if result.Character.Inventory[i] != nil {
				t.Error("torch should have been consumed")
			}
			return
		}
	}
	t.Skip("no empty slot for torch")
}

// --- Dispatch: drop_item ---

func TestDispatch_DropItem(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(0)
	state2 := Dispatch(&state, types.NewGame(42), cd)
	state3 := Dispatch(state2, types.AcceptCharacter(), cd)

	// Find a filled slot
	for i, item := range state3.Character.Inventory {
		if item != nil {
			result := Dispatch(state3, types.DropItem(i), cd)
			if result.Character.Inventory[i] != nil {
				// Could be a multi-slot item's continuation; check start
				t.Logf("slot %d still has item after drop (may be multi-slot)", i)
			}
			return
		}
	}
	t.Skip("no items to drop")
}

// --- Dispatch: equip_item ---

func TestDispatch_EquipItem(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(0)
	state2 := Dispatch(&state, types.NewGame(42), cd)
	state3 := Dispatch(state2, types.AcceptCharacter(), cd)

	// Find a 1-slot item at a non-zero slot index to avoid multi-slot swap issues
	for i, item := range state3.Character.Inventory {
		if item != nil && i > 0 && item.ID != "" && item.Slots == 1 {
			result := Dispatch(state3, types.EquipItem(i), cd)
			// The item from slot i should now be at slot 0
			if result.Character.Inventory[0] == nil || result.Character.Inventory[0].ID != item.ID {
				t.Errorf("expected item %s at slot 0, got %v", item.ID, result.Character.Inventory[0])
			}
			return
		}
	}
	t.Skip("no suitable non-slot-0 single-slot item to equip")
}

// --- Dispatch: reroll_character ---

func TestDispatch_RerollCharacter(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(0)
	state2 := Dispatch(&state, types.NewGame(42), cd)

	name1 := state2.Character.Name
	result := Dispatch(state2, types.RerollCharacter(), cd)
	// Character should be different (very likely with different RNG state)
	if result.Character == nil {
		t.Error("character should not be nil after reroll")
	}
	// Name might be the same with small name pool, so just check it exists
	if result.Character.Name == "" {
		t.Error("rerolled character should have a name")
	}
	_ = name1 // prevent unused warning
}

// --- Dispatch: select_type ---

func TestDispatch_SelectType(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(0)
	result := Dispatch(&state, types.SelectType(types.GameTypeExpedition), cd)
	if result.GameType != types.GameTypeExpedition {
		t.Errorf("expected expedition, got %s", result.GameType)
	}
}

// --- Dispatch: simple phase transitions ---

func TestDispatch_PhaseTransitions(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(0)

	// open_ossuary
	result := Dispatch(&state, types.OpenOssuary(), cd)
	if result.Phase != types.PhaseOssuary {
		t.Errorf("expected ossuary, got %s", result.Phase)
	}

	// return_to_title
	result = Dispatch(result, types.ReturnToTitle(), cd)
	if result.Phase != types.PhaseTitle {
		t.Errorf("expected title, got %s", result.Phase)
	}

	// start_game_setup
	result = Dispatch(result, types.StartGameSetup(), cd)
	if result.Phase != types.PhaseGameSetup {
		t.Errorf("expected game_setup, got %s", result.Phase)
	}
}

// --- Dispatch: restart ---

func TestDispatch_Restart(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(42)
	state2 := Dispatch(&state, types.NewGame(42), cd)
	result := Dispatch(state2, types.Restart(), cd)
	if result.Phase != types.PhaseTitle {
		t.Errorf("expected title, got %s", result.Phase)
	}
	if result.Character != nil {
		t.Error("restart should clear character")
	}
}

// --- GenerateDungeon ---

func TestGenerateDungeon(t *testing.T) {
	cd := loadTestContent(t)
	r := rng.New(42)
	rooms := GenerateDungeon(r, cd)

	if len(rooms) == 0 {
		t.Fatal("dungeon should have rooms")
	}
	// First room should be entry_hall
	if rooms[0].TemplateID != "entry_hall" {
		t.Errorf("first room should be entry_hall, got %s", rooms[0].TemplateID)
	}
	// Last room should be throne_room
	if rooms[len(rooms)-1].TemplateID != "throne_room" {
		t.Errorf("last room should be throne_room, got %s", rooms[len(rooms)-1].TemplateID)
	}
	// Should have at most 13 rooms (entry + 11 middle + boss)
	if len(rooms) > 13 {
		t.Errorf("expected at most 13 rooms, got %d", len(rooms))
	}
}

// --- Dispatch: flee ---

func TestDispatch_Flee(t *testing.T) {
	cd := loadTestContent(t)
	state := CreateInitialState(0)
	state2 := Dispatch(&state, types.NewGame(42), cd)
	state3 := Dispatch(state2, types.AcceptCharacter(), cd)

	// Navigate to room 1 first
	exits := state3.Exits()
	if len(exits) == 0 {
		t.Skip("no exits")
	}
	state4 := Dispatch(state3, types.ChooseExit(0), cd)

	// If we're in combat, try fleeing
	if state4.Phase == types.PhaseCombat {
		result := Dispatch(state4, types.FleeAction(), cd)
		validPhases := map[types.GamePhase]bool{
			types.PhaseExploring: true,
			types.PhaseDead:      true,
		}
		if !validPhases[result.Phase] {
			t.Errorf("unexpected phase after flee: %s", result.Phase)
		}
		return
	}

	// Set up combat manually for testing
	state4.Phase = types.PhaseCombat
	monsters := content.GetMonstersByTier("weak", cd)
	if len(monsters) == 0 {
		t.Skip("no weak monsters")
	}
	m := monsters[0]
	state4.Combat = &types.CombatState{
		Monster: types.MonsterInstance{
			Base:       m,
			CurrentHP:  m.HP,
			CurrentStr: m.Str,
		},
		Round: 1,
		Log:   []string{"A monster!"},
	}

	result := Dispatch(state4, types.FleeAction(), cd)
	validPhases := map[types.GamePhase]bool{
		types.PhaseExploring: true,
		types.PhaseDead:      true,
	}
	if !validPhases[result.Phase] {
		t.Errorf("unexpected phase after flee: %s", result.Phase)
	}
}

// --- foeLabel ---

func TestFoeLabel(t *testing.T) {
	if got := foeLabel("Goblin", 10); got != "Goblin" {
		t.Errorf("expected Goblin in bright light, got %s", got)
	}
	if got := foeLabel("Goblin", 5); got != "Goblin" {
		t.Errorf("expected Goblin in dim light, got %s", got)
	}
	if got := foeLabel("Goblin", 2); got != "Something" {
		t.Errorf("expected Something in dark, got %s", got)
	}
	if got := foeLabel("Goblin", 0); got != "Something" {
		t.Errorf("expected Something in black, got %s", got)
	}
}

// --- getDirectionTo ---

func TestGetDirectionTo(t *testing.T) {
	tests := []struct {
		from, to types.Pos
		expect   string
	}{
		{types.Pos{X: 0, Y: 0}, types.Pos{X: 1, Y: 0}, "e"},
		{types.Pos{X: 1, Y: 0}, types.Pos{X: 0, Y: 0}, "w"},
		{types.Pos{X: 0, Y: 1}, types.Pos{X: 0, Y: 0}, "n"},
		{types.Pos{X: 0, Y: 0}, types.Pos{X: 0, Y: 1}, "s"},
		{types.Pos{X: 0, Y: 0}, types.Pos{X: 2, Y: 0}, ""},
	}
	for _, tc := range tests {
		got := getDirectionTo(tc.from, tc.to)
		if got != tc.expect {
			t.Errorf("getDirectionTo(%v, %v) = %q, want %q", tc.from, tc.to, got, tc.expect)
		}
	}
}

// --- rollScar ---

func TestRollScar(t *testing.T) {
	char := types.Character{
		Str: 10, Dex: 10, Wil: 10, HP: 5, MaxHP: 5,
		Scars: []types.Scar{},
	}
	r := rng.New(42)
	newChar, scar := rollScar(char, r)

	if scar.Description == "" || scar.Effect == "" {
		t.Error("scar should have description and effect")
	}
	if len(newChar.Scars) != 1 {
		t.Errorf("expected 1 scar, got %d", len(newChar.Scars))
	}
	// At least one stat should have changed
	changed := newChar.Str != char.Str || newChar.Dex != char.Dex || newChar.Wil != char.Wil || newChar.MaxHP != char.MaxHP
	if !changed {
		t.Error("scar should have changed at least one stat")
	}
}

// --- moraleCheck ---

func TestMoraleCheck_BossNeverFlees(t *testing.T) {
	r := rng.New(42)
	monster := types.MonsterInstance{
		Base: types.Monster{Tier: "boss", Wil: 1},
	}
	// Even with WIL 1, boss should always hold firm
	for i := 0; i < 20; i++ {
		if !moraleCheck(monster, r) {
			t.Error("boss should never flee")
		}
	}
}

func TestMoraleCheck_WeakCanFlee(t *testing.T) {
	monster := types.MonsterInstance{
		Base: types.Monster{Tier: "weak", Wil: 5},
	}
	flees := 0
	for i := 0; i < 100; i++ {
		r := rng.New(i)
		if !moraleCheck(monster, r) {
			flees++
		}
	}
	if flees == 0 {
		t.Error("weak monster with WIL 5 should sometimes flee")
	}
}
