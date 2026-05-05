package engine

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/mph-llm-experiments/tinycrawl-tui/internal/content"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/dungeon"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/rng"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// DefaultExpeditionConfig is the fallback expedition config when a pack doesn't define one.
var DefaultExpeditionConfig = types.ExpeditionConfig{
	RoomBudget:           [2]int{25, 32},
	BranchProbability:    0.4,
	LoopProbability:      0.15,
	DeadEndRatio:         0.25,
	LightSourceFrequency: 0.3,
	GridSize:             9,
}

// AttackResult holds the outcome of a ResolveAttack call.
type AttackResult struct {
	PlayerDamageRolled int
	MonsterDamageRolled int
	PlayerResult       DamageResult
	MonsterResult      DamageResult
	PlayerCritSave     *bool
	MonsterCritSave    *bool
}

// DamageResult holds the result of resolving damage.
type DamageResult struct {
	NewHP     int
	NewStr    int
	StrDamage int
}

// FleeResult holds the outcome of a flee attempt.
type FleeResult struct {
	Success          bool
	Roll             int
	FreeAttackDamage int
}

// scarEntry is an entry in the scar table.
type scarEntry struct {
	Description string
	Effect      string
	Apply       func(c types.Character) types.Character
}

var scarTable = []scarEntry{
	{"A jagged scar across your face. You look dangerous now.", "+1 WIL", func(c types.Character) types.Character { c.Wil++; return c }},
	{"A deep wound that healed wrong. You learned to endure.", "+1 max HP", func(c types.Character) types.Character { c.MaxHP++; c.HP++; return c }},
	{"Broken bones, reset by grit alone. You are harder to break.", "+1 STR", func(c types.Character) types.Character { c.Str++; return c }},
	{"A close call sharpened your reflexes.", "+1 DEX", func(c types.Character) types.Character { c.Dex++; return c }},
	{"You stared death down and it blinked first.", "+1 max HP", func(c types.Character) types.Character { c.MaxHP++; c.HP++; return c }},
	{"The pain taught you where to strike.", "+1 STR", func(c types.Character) types.Character { c.Str++; return c }},
}

// rollScar picks a random scar and applies it to the character.
func rollScar(char types.Character, r *rng.Rng) (types.Character, types.Scar) {
	entry := rng.Pick(r, scarTable)
	scar := types.Scar{Description: entry.Description, Effect: entry.Effect}
	newChar := entry.Apply(char)
	newChar.Scars = append(append([]types.Scar{}, char.Scars...), scar)
	return newChar, scar
}

// moraleCheck returns true if the monster holds firm (passes WIL save). Bosses never flee.
func moraleCheck(monster types.MonsterInstance, r *rng.Rng) bool {
	if monster.Base.Tier == "boss" {
		return true
	}
	roll := r.RollDie(20)
	return roll <= monster.Base.Wil
}

// foeLabel returns "Something" in dark/black, otherwise the monster's name.
func foeLabel(name string, light int) string {
	b := types.GetLightBand(light)
	if b == types.LightDark || b == types.LightBlack {
		return "Something"
	}
	return name
}

// getDirectionTo returns the cardinal direction from one adjacent position to another.
func getDirectionTo(from, to types.Pos) string {
	dx := to.X - from.X
	dy := to.Y - from.Y
	if dx == 1 && dy == 0 {
		return "e"
	}
	if dx == -1 && dy == 0 {
		return "w"
	}
	if dx == 0 && dy == -1 {
		return "n"
	}
	if dx == 0 && dy == 1 {
		return "s"
	}
	return ""
}

// directionOffsets maps direction strings to position offsets.
var directionOffsets = map[string]types.Pos{
	"n": {X: 0, Y: -1},
	"e": {X: 1, Y: 0},
	"s": {X: 0, Y: 1},
	"w": {X: -1, Y: 0},
}

// updateExpeditionExits populates the current expedition room's Exits field
// from its walls. Expedition exits are dynamic (built from grid walls + hints),
// unlike sprint exits which are static from the room template.
func updateExpeditionExits(s *types.GameState) {
	if s.Expedition == nil {
		return
	}
	room := dungeon.GetRoom(s.Expedition.Grid, s.Expedition.PlayerPos)
	if room == nil {
		return
	}
	room.Exits = dungeon.BuildExits(s.Expedition.Grid, room, &s.Expedition.Entry, s.Expedition.PreviousPos)
}

// CreateInitialState creates a new game state at the title screen.
func CreateInitialState(seed int) types.GameState {
	return types.GameState{
		Phase:          types.PhaseTitle,
		GameType:       types.GameTypeSprint,
		Character:      nil,
		Sprint:         nil,
		Expedition:     nil,
		Combat:         nil,
		PendingLoot:    []types.Item{},
		Log:            []types.LogEntry{},
		Seed:           seed,
		RngState:       seed,
		RestsTaken:     0,
		MonstersKilled: 0,
		Light:          10,
	}
}

// CreateCharacter generates a new character using the RNG and content data.
func CreateCharacter(r *rng.Rng, cd types.ContentData) types.Character {
	str := r.RollDice(3, 6)
	dex := r.RollDice(3, 6)
	wil := r.RollDice(3, 6)
	hp := r.RollDie(6) + 4
	name := rng.Pick(r, cd.Names)

	weapon := rng.Pick(r, cd.StartingGear.Weapons)
	armorRoll := rng.Pick(r, cd.StartingGear.Armor)
	gear := rng.Pick(r, cd.StartingGear.Gear)
	trinket := rng.Pick(r, cd.StartingGear.Trinkets)

	inventory := make([]*types.Item, types.MaxSlots)
	slotIdx := 0

	equip := func(item types.Item) {
		if item.ID == "" || item.Slots == 0 {
			return
		}
		if slotIdx+item.Slots > types.MaxSlots {
			return
		}
		itemCopy := item
		inventory[slotIdx] = &itemCopy
		if item.Slots == 2 && slotIdx+1 < types.MaxSlots {
			inventory[slotIdx+1] = &itemCopy
			slotIdx += 2
		} else {
			slotIdx++
		}
	}

	equip(weapon)
	equip(armorRoll)
	equip(gear)
	equip(trinket)

	// Calculate armor (max 3 in Cairn)
	seen := make(map[string]bool)
	armorTotal := 0
	for _, item := range inventory {
		if item == nil || item.ArmorValue == 0 || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		armorTotal += item.ArmorValue
	}

	return types.Character{
		Name:      name,
		Str:       str,
		Dex:       dex,
		Wil:       wil,
		HP:        hp,
		MaxHP:     hp,
		Armor:     min(armorTotal, 3),
		Inventory: inventory,
		Scars:     []types.Scar{},
	}
}

// SlotsUsed counts the number of non-nil inventory slots.
func SlotsUsed(char types.Character) int {
	count := 0
	for _, item := range char.Inventory {
		if item != nil {
			count++
		}
	}
	return count
}

// ResolveDamage applies damage after armor absorption to HP then STR.
func ResolveDamage(rawDamage, armor, hp, str int) (newHP, newStr, strDamage int) {
	effective := max(0, rawDamage-armor)
	if effective <= 0 {
		return hp, str, 0
	}
	if effective <= hp {
		return hp - effective, str, 0
	}
	overflow := effective - hp
	newStr = max(0, str-overflow)
	return 0, newStr, overflow
}

// CriticalDamageSave rolls d20 and returns true if roll <= str.
func CriticalDamageSave(str int, r *rng.Rng) bool {
	roll := r.RollDie(20)
	return roll <= str
}

// ResolveAttack resolves a simultaneous attack exchange between player and monster.
func ResolveAttack(char types.Character, monster types.MonsterInstance, weapon types.Item, r *rng.Rng) AttackResult {
	playerDamageRolled := r.RollDie(4)
	if weapon.Damage != "" {
		playerDamageRolled = r.RollNotation(weapon.Damage)
	}
	monsterDamageRolled := r.RollNotation(monster.Base.Attack.Die)

	mNewHP, mNewStr, mStrDmg := ResolveDamage(playerDamageRolled, monster.Base.Armor, monster.CurrentHP, monster.CurrentStr)
	pNewHP, pNewStr, pStrDmg := ResolveDamage(monsterDamageRolled, char.Armor, char.HP, char.Str)

	var playerCritSave *bool
	if pStrDmg > 0 && pNewStr > 0 {
		saved := CriticalDamageSave(pNewStr, r)
		playerCritSave = &saved
	}

	var monsterCritSave *bool
	if mStrDmg > 0 && mNewStr > 0 {
		saved := CriticalDamageSave(mNewStr, r)
		monsterCritSave = &saved
	}

	return AttackResult{
		PlayerDamageRolled:  playerDamageRolled,
		MonsterDamageRolled: monsterDamageRolled,
		PlayerResult:        DamageResult{NewHP: pNewHP, NewStr: pNewStr, StrDamage: pStrDmg},
		MonsterResult:       DamageResult{NewHP: mNewHP, NewStr: mNewStr, StrDamage: mStrDmg},
		PlayerCritSave:      playerCritSave,
		MonsterCritSave:     monsterCritSave,
	}
}

// AttemptFlee attempts to flee from combat. On failure the monster gets a free attack.
func AttemptFlee(char types.Character, r *rng.Rng, monsterAttackDie string) FleeResult {
	roll := r.RollDie(20)
	success := roll <= char.Dex

	if success {
		return FleeResult{Success: true, Roll: roll}
	}

	freeAttackDamage := 0
	if monsterAttackDie != "" {
		freeAttackDamage = r.RollNotation(monsterAttackDie)
	}
	return FleeResult{Success: false, Roll: roll, FreeAttackDamage: freeAttackDamage}
}

// findEmptySlot finds a contiguous block of empty slots.
func findEmptySlot(inventory []*types.Item, slotsNeeded int) int {
	for i := 0; i <= len(inventory)-slotsNeeded; i++ {
		fits := true
		for j := 0; j < slotsNeeded; j++ {
			if inventory[i+j] != nil {
				fits = false
				break
			}
		}
		if fits {
			return i
		}
	}
	return -1
}

// CanAddItem returns true if the character has room for an item needing slotsNeeded slots.
func CanAddItem(char types.Character, slotsNeeded int) bool {
	return findEmptySlot(char.Inventory, slotsNeeded) != -1
}

// AddItem adds an item to the character's inventory. Returns nil if no room.
func AddItem(char types.Character, item types.Item) *types.Character {
	idx := findEmptySlot(char.Inventory, item.Slots)
	if idx == -1 {
		return nil
	}

	newInv := make([]*types.Item, len(char.Inventory))
	copy(newInv, char.Inventory)
	for i := 0; i < item.Slots; i++ {
		itemCopy := item
		newInv[idx+i] = &itemCopy
	}

	newChar := char
	newChar.Inventory = newInv
	newChar.Armor = RecalculateArmor(newChar)
	return &newChar
}

// DropItem removes an item from the given slot index.
func DropItem(char types.Character, slotIndex int) types.Character {
	if slotIndex < 0 || slotIndex >= len(char.Inventory) {
		return char
	}
	item := char.Inventory[slotIndex]
	if item == nil {
		return char
	}

	newInv := make([]*types.Item, len(char.Inventory))
	copy(newInv, char.Inventory)

	// Find start of this multi-slot item (matching by ID since pointers may not be shared after copy)
	start := slotIndex
	for start > 0 && newInv[start-1] != nil && newInv[start-1].ID == item.ID {
		start--
	}
	// Null all slots belonging to this item
	for i := start; i < start+item.Slots && i < len(newInv); i++ {
		newInv[i] = nil
	}

	newChar := char
	newChar.Inventory = newInv
	newChar.Armor = RecalculateArmor(newChar)
	return newChar
}

// RecalculateArmor sums armor values from inventory items, capped at 3.
func RecalculateArmor(char types.Character) int {
	seen := make(map[string]bool)
	total := 0
	for _, item := range char.Inventory {
		if item == nil || item.ArmorValue == 0 || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		total += item.ArmorValue
	}
	return min(total, 3)
}

// GetVisibleDescription returns room description adjusted for light level.
func GetVisibleDescription(description string, band types.LightBand) string {
	switch band {
	case types.LightBlack:
		return "You can't see."
	case types.LightDark:
		return ""
	case types.LightDim:
		re := regexp.MustCompile(`^[^.!]+[.!]`)
		match := re.FindString(description)
		if match != "" {
			return match
		}
		return description
	default:
		return description
	}
}

// tierOrder returns a numeric tier for room sorting purposes.
func tierOrder(room types.RoomTemplate) int {
	if len(room.Encounters) == 0 {
		return -1
	}
	hardest := 0
	for _, enc := range room.Encounters {
		val := 0
		switch enc.Tier {
		case "weak", "minor":
			val = 0
		case "moderate", "standard":
			val = 1
		default:
			val = 2
		}
		if val > hardest {
			hardest = val
		}
	}
	return hardest
}

// GenerateDungeon creates a linear sprint dungeon room sequence.
func GenerateDungeon(r *rng.Rng, cd types.ContentData) []types.DungeonRoom {
	var entry, boss *types.RoomTemplate
	var middle []types.RoomTemplate

	for i := range cd.Rooms {
		switch cd.Rooms[i].ID {
		case "entry_hall":
			entry = &cd.Rooms[i]
		case "throne_room":
			boss = &cd.Rooms[i]
		default:
			middle = append(middle, cd.Rooms[i])
		}
	}

	if entry == nil || boss == nil {
		return nil
	}

	// Separate into tier bands
	var weak, moderate, tough []types.RoomTemplate
	for _, room := range middle {
		t := tierOrder(room)
		switch {
		case t == 0:
			weak = append(weak, room)
		case t == 1 || t == -1:
			moderate = append(moderate, room)
		default:
			tough = append(tough, room)
		}
	}

	rng.Shuffle(r, weak)
	rng.Shuffle(r, moderate)
	rng.Shuffle(r, tough)

	sorted := make([]types.RoomTemplate, 0, len(weak)+len(moderate)+len(tough))
	sorted = append(sorted, weak...)
	sorted = append(sorted, moderate...)
	sorted = append(sorted, tough...)

	limit := min(11, len(sorted))
	selected := sorted[:limit]

	sequence := make([]types.RoomTemplate, 0, len(selected)+2)
	sequence = append(sequence, *entry)
	sequence = append(sequence, selected...)
	sequence = append(sequence, *boss)

	rooms := make([]types.DungeonRoom, len(sequence))
	for i, tmpl := range sequence {
		description := content.RenderDescription(tmpl.Features, cd.Fragments, r)

		var encounter *types.Encounter
		for _, enc := range tmpl.Encounters {
			if r.Chance(enc.Chance) {
				if enc.Type == "monster" {
					monstersInTier := content.GetMonstersByTier(enc.Tier, cd)
					if len(monstersInTier) > 0 {
						base := rng.Pick(r, monstersInTier)
						encounter = &types.Encounter{
							Type: "monster",
							Monster: &types.MonsterInstance{
								Base:       base,
								CurrentHP:  base.HP,
								CurrentStr: base.Str,
							},
						}
					}
				} else if enc.Type == "trap" {
					damage := "d4"
					if enc.Tier != "minor" {
						damage = "d6"
					}
					encounter = &types.Encounter{
						Type:        "trap",
						Damage:      damage,
						SaveStat:    "dex",
						Description: "A hidden mechanism triggers!",
					}
				}
				break
			}
		}

		var loot []types.Item
		if tmpl.Loot != nil && r.Chance(tmpl.Loot.Chance) {
			pool := content.GetLootByTier(tmpl.Loot.Tier, cd)
			if len(pool) > 0 {
				loot = []types.Item{rng.Pick(r, pool)}
			}
		}
		if loot == nil {
			loot = []types.Item{}
		}

		rooms[i] = types.DungeonRoom{
			TemplateID:  tmpl.ID,
			Name:        tmpl.Name,
			Description: description,
			Encounter:   encounter,
			Loot:        loot,
			Exits:       tmpl.Exits,
			Visited:     false,
			Cleared:     false,
		}
	}

	return rooms
}

// markCleared marks the current room cleared depending on dungeon type.
func markCleared(state *types.GameState) {
	if state.Sprint != nil {
		d := dungeon.MarkSprintCleared(*state.Sprint, state.Sprint.CurrentRoomIndex)
		state.Sprint = &d
	}
	if state.Expedition != nil {
		d := dungeon.MarkExpeditionCleared(*state.Expedition)
		state.Expedition = &d
	}
}

// markVisited marks the current room visited depending on dungeon type.
func markVisited(state *types.GameState) {
	if state.Sprint != nil {
		d := dungeon.MarkSprintVisited(*state.Sprint, state.Sprint.CurrentRoomIndex)
		state.Sprint = &d
	}
	if state.Expedition != nil {
		d := dungeon.MarkExpeditionVisited(*state.Expedition)
		state.Expedition = &d
	}
}

// copyState creates a shallow copy of the state.
func copyState(state *types.GameState) *types.GameState {
	s := *state
	return &s
}

// copyChar creates a copy of the character.
func copyChar(c *types.Character) *types.Character {
	if c == nil {
		return nil
	}
	nc := *c
	// Copy inventory
	nc.Inventory = make([]*types.Item, len(c.Inventory))
	copy(nc.Inventory, c.Inventory)
	// Copy scars
	nc.Scars = make([]types.Scar, len(c.Scars))
	copy(nc.Scars, c.Scars)
	return &nc
}

// findWeapon finds the first weapon in inventory, or returns fists.
func findWeapon(char *types.Character) types.Item {
	for _, item := range char.Inventory {
		if item != nil && item.Type == "weapon" {
			return *item
		}
	}
	return types.Item{ID: "_fist", Name: "fists", Type: "weapon", Damage: "d4"}
}

// resolveAttackAction performs a full combat round with advantage/stun/blind modifiers.
func resolveAttackAction(state *types.GameState, char types.Character, monster types.MonsterInstance, weapon types.Item, r *rng.Rng) *types.GameState {
	result := ResolveAttack(char, monster, weapon, r)
	var combatLog []string
	band := types.GetLightBand(state.Light)
	inDark := band == types.LightDark || band == types.LightBlack
	foe := monster.Base.Name
	if inDark {
		foe = "Something"
	}

	// Apply advantage: double player damage
	playerDamageRolled := result.PlayerDamageRolled
	if state.Combat.PlayerHasAdvantage {
		playerDamageRolled *= 2
		combatLog = append(combatLog, "Advantage! Double damage!")
	}

	if band == types.LightBlack {
		playerDamageRolled = max(0, playerDamageRolled-2)
		combatLog = append(combatLog, "Fighting blind! -2 damage.")
	}

	// Apply stun: monster deals 0
	monsterDamageRolled := result.MonsterDamageRolled
	if state.Combat.MonsterStunned {
		monsterDamageRolled = 0
		combatLog = append(combatLog, fmt.Sprintf("%s is stunned and cannot attack!", foe))
	}

	mNewHP, mNewStr, mStrDmg := ResolveDamage(playerDamageRolled, monster.Base.Armor, monster.CurrentHP, monster.CurrentStr)
	pNewHP, pNewStr, pStrDmg := ResolveDamage(monsterDamageRolled, char.Armor, char.HP, char.Str)

	var playerCritSave *bool
	if pStrDmg > 0 && pNewStr > 0 {
		saved := CriticalDamageSave(pNewStr, r)
		playerCritSave = &saved
	}

	var monsterCritSave *bool
	if mStrDmg > 0 && mNewStr > 0 {
		saved := CriticalDamageSave(mNewStr, r)
		monsterCritSave = &saved
	}

	// Player's attack log
	mArmor := monster.Base.Armor
	if inDark {
		combatLog = append(combatLog, fmt.Sprintf("You swing %s in the dark for %d damage.", weapon.Name, playerDamageRolled))
	} else if mArmor > 0 && playerDamageRolled > 0 {
		effective := max(0, playerDamageRolled-mArmor)
		combatLog = append(combatLog, fmt.Sprintf("You strike with %s for %d (%d absorbed by armor, %d through).", weapon.Name, playerDamageRolled, mArmor, effective))
	} else {
		combatLog = append(combatLog, fmt.Sprintf("You strike with %s for %d damage.", weapon.Name, playerDamageRolled))
	}
	if mStrDmg > 0 {
		combatLog = append(combatLog, fmt.Sprintf("%s: HP gone, %d damage to STR!", foe, mStrDmg))
	}

	// Monster's attack log
	if char.Armor > 0 && monsterDamageRolled > 0 {
		effective := max(0, monsterDamageRolled-char.Armor)
		combatLog = append(combatLog, fmt.Sprintf("%s hits you for %d (%d absorbed by armor, %d through).", foe, monsterDamageRolled, char.Armor, effective))
	} else {
		combatLog = append(combatLog, fmt.Sprintf("%s hits you for %d damage.", foe, monsterDamageRolled))
	}
	if pStrDmg > 0 {
		combatLog = append(combatLog, fmt.Sprintf("Your HP gone — %d damage to STR!", pStrDmg))
	}

	monsterDead := false
	newMonster := monster
	newMonster.CurrentHP = mNewHP
	newMonster.CurrentStr = mNewStr

	if mNewStr <= 0 {
		monsterDead = true
		if inDark {
			combatLog = append(combatLog, "You hear something collapse.")
		} else {
			combatLog = append(combatLog, fmt.Sprintf("%s is destroyed!", monster.Base.Name))
		}
	} else if monsterCritSave != nil && !*monsterCritSave {
		monsterDead = true
		if inDark {
			combatLog = append(combatLog, "You hear something collapse.")
		} else {
			combatLog = append(combatLog, fmt.Sprintf("%s fails its critical damage save and falls!", monster.Base.Name))
		}
	} else if monsterCritSave != nil && *monsterCritSave {
		holdsFirm := moraleCheck(newMonster, r)
		if !holdsFirm {
			monsterDead = true
			if inDark {
				combatLog = append(combatLog, "You hear something scramble away.")
			} else {
				combatLog = append(combatLog, fmt.Sprintf("%s takes a grievous wound and flees!", monster.Base.Name))
			}
		} else {
			if inDark {
				combatLog = append(combatLog, fmt.Sprintf("%s takes a grievous wound but fights on!", foe))
			} else {
				combatLog = append(combatLog, fmt.Sprintf("%s takes a grievous wound but fights on!", monster.Base.Name))
			}
		}
	}

	newChar := char
	newChar.HP = pNewHP
	newChar.Str = pNewStr
	if newChar.Scars == nil {
		newChar.Scars = []types.Scar{}
	}

	if pNewStr <= 0 {
		combatLog = append(combatLog, "Your body gives out.")
		deathLog := make([]types.LogEntry, len(combatLog))
		for i, text := range combatLog {
			deathLog[i] = types.LogEntry{Text: text, Type: "combat"}
		}
		s := copyState(state)
		s.Phase = types.PhaseDead
		s.Character = &newChar
		s.Combat = nil
		s.Log = deathLog
		s.RngState = r.GetState()
		return s
	}
	if playerCritSave != nil && !*playerCritSave {
		combatLog = append(combatLog, "Critical damage! You fail your save and collapse.")
		deathLog := make([]types.LogEntry, len(combatLog))
		for i, text := range combatLog {
			deathLog[i] = types.LogEntry{Text: text, Type: "combat"}
		}
		s := copyState(state)
		s.Phase = types.PhaseDead
		s.Character = &newChar
		s.Combat = nil
		s.Log = deathLog
		s.RngState = r.GetState()
		return s
	}
	if playerCritSave != nil && *playerCritSave {
		scarredChar, scar := rollScar(newChar, r)
		newChar = scarredChar
		combatLog = append(combatLog, fmt.Sprintf("Critical damage! You grit your teeth and hold on. (%s %s)", scar.Description, scar.Effect))
	}

	if monsterDead {
		room := state.CurrentRoom()
		s := copyState(state)
		markCleared(s)
		s.MonstersKilled++

		if state.IsFinalRoom() {
			s.Phase = types.PhaseVictory
			s.Character = &newChar
			s.Combat = nil
			victoryText := fmt.Sprintf("You have defeated %s! The dungeon is yours.", monster.Base.Name)
			if inDark {
				victoryText = "The dungeon falls silent. You survived."
			}
			s.Log = []types.LogEntry{{Text: victoryText, Type: "system"}}
			s.RngState = r.GetState()
			return s
		}

		if room != nil && len(room.Loot) > 0 {
			s.Phase = types.PhaseLooting
			s.Character = &newChar
			s.Combat = nil
			s.PendingLoot = room.Loot
			lootText := fmt.Sprintf("%s falls. You notice something on the ground.", monster.Base.Name)
			if inDark {
				lootText = "It falls. You notice something on the ground."
			}
			s.Log = []types.LogEntry{{Text: lootText, Type: "loot"}}
			s.RngState = r.GetState()
			return s
		}

		s.Phase = types.PhaseExploring
		s.Character = &newChar
		s.Combat = nil
		fallText := fmt.Sprintf("%s falls.", monster.Base.Name)
		if inDark {
			fallText = "It falls."
		}
		s.Log = []types.LogEntry{{Text: fallText, Type: "combat"}}
		s.RngState = r.GetState()
		return s
	}

	// Combat continues
	s := copyState(state)
	s.Character = &newChar
	s.Combat = &types.CombatState{
		Monster:            newMonster,
		Round:              state.Combat.Round + 1,
		Log:                combatLog,
		MonsterStunned:     false,
		PlayerHasAdvantage: false,
	}
	s.RngState = r.GetState()
	return s
}

// handleMonsterDeath returns a new state when a monster dies (from creative damage, etc.).
func handleMonsterDeath(state *types.GameState, char types.Character, monster types.MonsterInstance, inDark bool, r *rng.Rng) *types.GameState {
	room := state.CurrentRoom()
	s := copyState(state)
	markCleared(s)
	s.MonstersKilled++

	if state.IsFinalRoom() {
		s.Phase = types.PhaseVictory
		s.Character = &char
		s.Combat = nil
		victoryText := fmt.Sprintf("You have defeated %s!", monster.Base.Name)
		if inDark {
			victoryText = "The dungeon falls silent. You survived."
		}
		s.Log = []types.LogEntry{{Text: victoryText, Type: "system"}}
		s.RngState = r.GetState()
		return s
	}

	if room != nil && len(room.Loot) > 0 {
		s.Phase = types.PhaseLooting
		s.Character = &char
		s.Combat = nil
		s.PendingLoot = room.Loot
		lootText := fmt.Sprintf("%s falls.", monster.Base.Name)
		if inDark {
			lootText = "It falls. You notice something on the ground."
		}
		s.Log = []types.LogEntry{{Text: lootText, Type: "loot"}}
		s.RngState = r.GetState()
		return s
	}

	s.Phase = types.PhaseExploring
	s.Character = &char
	s.Combat = nil
	fallText := fmt.Sprintf("%s falls.", monster.Base.Name)
	if inDark {
		fallText = "It falls."
	}
	s.Log = []types.LogEntry{{Text: fallText, Type: "combat"}}
	s.RngState = r.GetState()
	return s
}

// handlePlayerDamage handles player damage from a monster attack, including crit saves and death.
// Returns (newChar, dead, combatLog).
func handlePlayerDamage(char types.Character, monsterDmg int, combatLog []string, foe string, r *rng.Rng) (types.Character, bool, []string) {
	pNewHP, pNewStr, pStrDmg := ResolveDamage(monsterDmg, char.Armor, char.HP, char.Str)
	combatLog = append(combatLog, fmt.Sprintf("%s attacks for %d damage.", foe, monsterDmg))
	newChar := char
	newChar.HP = pNewHP
	newChar.Str = pNewStr

	if pNewStr <= 0 {
		combatLog = append(combatLog, "Your body gives out.")
		return newChar, true, combatLog
	}
	if pStrDmg > 0 && pNewStr > 0 {
		saved := CriticalDamageSave(pNewStr, r)
		if !saved {
			combatLog = append(combatLog, "Critical damage! You fail your save.")
			return newChar, true, combatLog
		}
		combatLog = append(combatLog, "Critical damage! You hold on.")
	}

	return newChar, false, combatLog
}

// Dispatch is the main game reducer. It takes a state, an action, and content data,
// and returns a new state.
func Dispatch(state *types.GameState, action types.Action, cd types.ContentData) *types.GameState {
	r := rng.New(state.RngState)

	switch action.Type {
	case "new_game":
		seed := action.Seed
		if seed == 0 {
			seed = int(math.Floor(rng.New(0).Next() * 2147483647))
		}
		gameRng := rng.New(seed)
		character := CreateCharacter(gameRng, cd)

		s := copyState(state)
		s.Phase = types.PhaseCharacterCreation
		s.Character = &character
		s.Combat = nil
		s.PendingLoot = []types.Item{}
		s.Log = []types.LogEntry{}
		s.Seed = seed
		s.RngState = gameRng.GetState()
		s.RestsTaken = 0
		s.MonstersKilled = 0
		s.Light = 10

		if state.GameType == types.GameTypeExpedition {
			config := DefaultExpeditionConfig
			if cd.Rooms != nil { // use pack config if available — currently ContentData doesn't have it directly
				// Config comes from pack, passed through; for now use default
			}
			result := dungeon.GenerateExpeditionGrid(gameRng, cd, config)
			d := dungeon.CreateExpeditionDungeon(result.Grid, result.Entry, result.BossPos, result.RoomCount)
			s.Expedition = &d
			s.Sprint = nil
			s.RngState = gameRng.GetState()
		} else {
			rooms := GenerateDungeon(gameRng, cd)
			d := dungeon.CreateSprintDungeon(rooms)
			s.Sprint = &d
			s.Expedition = nil
			s.RngState = gameRng.GetState()
		}

		return s

	case "reroll_character":
		character := CreateCharacter(r, cd)
		s := copyState(state)
		s.Character = &character
		s.RngState = r.GetState()
		return s

	case "accept_character":
		s := copyState(state)
		s.Phase = types.PhaseExploring
		s.Log = []types.LogEntry{{Text: "You descend into the dark.", Type: "info"}}

		if s.Sprint != nil {
			d := dungeon.MarkSprintVisited(*s.Sprint, 0)
			d = dungeon.MarkSprintCleared(d, 0)
			s.Sprint = &d
		}
		if s.Expedition != nil {
			d := dungeon.MarkExpeditionVisited(*s.Expedition)
			d = dungeon.MarkExpeditionCleared(d)
			s.Expedition = &d
			updateExpeditionExits(s)
		}

		s.RngState = r.GetState()
		return s

	case "choose_exit":
		exits := state.Exits()
		if action.ExitIndex < 0 || action.ExitIndex >= len(exits) {
			return state
		}
		exit := exits[action.ExitIndex]

		s := copyState(state)
		s.RestsTaken = 0

		// Navigate
		var nextRoom *types.DungeonRoom
		if s.Sprint != nil {
			var nextIndex int
			if exit.Direction == "forward" {
				nextIndex = s.Sprint.CurrentRoomIndex + 1
			} else {
				nextIndex = max(0, s.Sprint.CurrentRoomIndex-1)
			}
			if nextIndex >= len(s.Sprint.Rooms) {
				return state
			}
			nextRoom = &s.Sprint.Rooms[nextIndex]
			d := dungeon.NavigateSprint(*s.Sprint, action.ExitIndex)
			s.Sprint = &d
		} else if s.Expedition != nil {
			d := dungeon.NavigateExpedition(*s.Expedition, exit.Direction)
			// Check if navigation actually moved
			if d.PlayerPos == s.Expedition.PlayerPos {
				return state
			}
			s.Expedition = &d
			room := d.Grid[d.PlayerPos.Y][d.PlayerPos.X]
			if room != nil {
				nextRoom = &room.DungeonRoom
			}
			updateExpeditionExits(s)
		} else {
			return state
		}

		if nextRoom == nil {
			return state
		}

		newLight := max(0, s.Light-1)

		var lightLog []types.LogEntry
		if state.Light >= 7 && newLight < 7 {
			lightLog = append(lightLog, types.LogEntry{Text: "Your torch flickers. The shadows grow longer.", Type: "danger"})
		} else if state.Light >= 3 && newLight < 3 {
			lightLog = append(lightLog, types.LogEntry{Text: "Your light gutters and dies. You are in the dark.", Type: "danger"})
		} else if state.Light >= 1 && newLight < 1 {
			lightLog = append(lightLog, types.LogEntry{Text: "Total darkness. You move by touch and sound.", Type: "danger"})
		}

		// Mark visited
		markVisited(s)

		logEntries := []types.LogEntry{{Text: fmt.Sprintf("You enter %s.", nextRoom.Name), Type: "info"}}

		// Healing spring check
		healedChar := *s.Character
		if nextRoom.TemplateID == "healing_spring" && !nextRoom.Visited {
			healed := healedChar.MaxHP - healedChar.HP
			if healed > 0 {
				healedChar.HP = healedChar.MaxHP
				logEntries = append(logEntries, types.LogEntry{Text: fmt.Sprintf("The glowing waters restore you. Recovered %d HP.", healed), Type: "info"})
			} else {
				logEntries = append(logEntries, types.LogEntry{Text: "The spring glows softly. You feel at peace.", Type: "info"})
			}
		}

		// Check encounters
		if !nextRoom.Cleared && nextRoom.Encounter != nil {
			if nextRoom.Encounter.Type == "monster" && nextRoom.Encounter.Monster != nil {
				monsterName := nextRoom.Encounter.Monster.Base.Name
				newBand := types.GetLightBand(newLight)
				combatEntry := fmt.Sprintf("A %s blocks your path!", monsterName)
				if newBand == types.LightDark || newBand == types.LightBlack {
					combatEntry = "Something lurches at you from the dark!"
				}

				s.Phase = types.PhaseCombat
				monsterCopy := *nextRoom.Encounter.Monster
				s.Combat = &types.CombatState{
					Monster:            monsterCopy,
					Round:              1,
					Log:                []string{combatEntry},
					MonsterStunned:     false,
					PlayerHasAdvantage: false,
				}
				combinedLog := append(lightLog, logEntries...)
				s.Log = combinedLog
				s.Light = newLight
				s.Character = &healedChar
				s.RngState = r.GetState()
				return s
			} else if nextRoom.Encounter.Type == "trap" {
				trapBand := types.GetLightBand(newLight)
				saved := false
				if trapBand == types.LightDark || trapBand == types.LightBlack {
					logEntries = append(logEntries, types.LogEntry{Text: "You stumble into a trap in the darkness!", Type: "danger"})
				} else {
					saveRoll := r.RollDie(20)
					saved = saveRoll <= healedChar.Dex
				}
				if saved {
					logEntries = append(logEntries, types.LogEntry{Text: "You spot a trap and avoid it!", Type: "danger"})
					markCleared(s)
					if len(nextRoom.Loot) > 0 {
						s.Phase = types.PhaseLooting
					} else {
						s.Phase = types.PhaseExploring
					}
					s.PendingLoot = nextRoom.Loot
					s.Log = append(lightLog, logEntries...)
					s.Light = newLight
					s.Character = &healedChar
					s.RngState = r.GetState()
					return s
				}
				// Trap damage
				trapDamage := r.RollNotation(nextRoom.Encounter.Damage)
				tNewHP, tNewStr, _ := ResolveDamage(trapDamage, healedChar.Armor, healedChar.HP, healedChar.Str)
				logEntries = append(logEntries, types.LogEntry{Text: fmt.Sprintf("A trap springs! You take %d damage.", trapDamage), Type: "danger"})

				if tNewStr <= 0 {
					s.Phase = types.PhaseDead
					s.Log = append(lightLog, logEntries...)
					s.Light = newLight
					s.RngState = r.GetState()
					return s
				}

				trapChar := healedChar
				trapChar.HP = tNewHP
				trapChar.Str = tNewStr
				markCleared(s)
				if len(nextRoom.Loot) > 0 {
					s.Phase = types.PhaseLooting
				} else {
					s.Phase = types.PhaseExploring
				}
				s.Character = &trapChar
				s.PendingLoot = nextRoom.Loot
				s.Log = append(lightLog, logEntries...)
				s.Light = newLight
				s.RngState = r.GetState()
				return s
			}
		}

		// Check loot (no encounter)
		if !nextRoom.Cleared && len(nextRoom.Loot) > 0 {
			s.Phase = types.PhaseLooting
			s.Character = &healedChar
			markCleared(s)
			s.PendingLoot = nextRoom.Loot
			s.Log = append(lightLog, logEntries...)
			s.Light = newLight
			s.RngState = r.GetState()
			return s
		}

		// Clear room, continue exploring
		s.Phase = types.PhaseExploring
		s.Character = &healedChar
		markCleared(s)
		s.Log = append(lightLog, logEntries...)
		s.Light = newLight
		s.RngState = r.GetState()
		return s

	case "attack":
		if state.Combat == nil || state.Character == nil {
			return state
		}
		char := *state.Character
		weapon := findWeapon(state.Character)
		return resolveAttackAction(state, char, state.Combat.Monster, weapon, r)

	case "flee":
		if state.Combat == nil || state.Character == nil {
			return state
		}
		char := *state.Character
		monster := state.Combat.Monster
		fleeResult := AttemptFlee(char, r, monster.Base.Attack.Die)

		// fleeDungeon navigates backward
		fleeDungeon := func(s *types.GameState) {
			if s.Sprint != nil {
				prevIndex := max(0, s.Sprint.CurrentRoomIndex-1)
				// Navigate backward by finding the "back" exit
				// In sprint, exits are "forward"/"back" — we need to go to prevIndex
				d := *s.Sprint
				d.CurrentRoomIndex = prevIndex
				s.Sprint = &d
			}
			if s.Expedition != nil {
				exp := s.Expedition
				if exp.PreviousPos != nil {
					dir := getDirectionTo(exp.PlayerPos, *exp.PreviousPos)
					if dir != "" {
						d := dungeon.NavigateExpedition(*exp, dir)
						s.Expedition = &d
						updateExpeditionExits(s)
					}
				}
			}
		}

		if fleeResult.Success {
			s := copyState(state)
			s.Phase = types.PhaseExploring
			fleeDungeon(s)
			s.Combat = nil
			s.Log = []types.LogEntry{{Text: "You escape back the way you came.", Type: "info"}}
			s.RngState = r.GetState()
			return s
		}

		if fleeResult.FreeAttackDamage > 0 {
			fNewHP, fNewStr, fStrDmg := ResolveDamage(fleeResult.FreeAttackDamage, char.Armor, char.HP, char.Str)
			fleeLog := []types.LogEntry{
				{Text: fmt.Sprintf("You try to flee! (rolled %d vs DEX %d — failed)", fleeResult.Roll, char.Dex), Type: "combat"},
				{Text: fmt.Sprintf("%s strikes as you turn: %d damage.", foeLabel(monster.Base.Name, state.Light), fleeResult.FreeAttackDamage), Type: "combat"},
			}

			if char.Armor > 0 {
				effective := max(0, fleeResult.FreeAttackDamage-char.Armor)
				fleeLog = append(fleeLog, types.LogEntry{Text: fmt.Sprintf("Armor absorbs %d, %d gets through.", char.Armor, effective), Type: "combat"})
			}

			if fStrDmg > 0 {
				fleeLog = append(fleeLog, types.LogEntry{Text: fmt.Sprintf("HP gone — %d damage to STR!", fStrDmg), Type: "danger"})
			}

			newChar := char
			newChar.HP = fNewHP
			newChar.Str = fNewStr

			if fNewStr <= 0 {
				fleeLog = append(fleeLog, types.LogEntry{Text: "Your body gives out.", Type: "danger"})
				s := copyState(state)
				s.Phase = types.PhaseDead
				s.Character = &newChar
				s.Combat = nil
				s.Log = fleeLog
				s.RngState = r.GetState()
				return s
			}

			if fStrDmg > 0 && fNewStr > 0 {
				saved := CriticalDamageSave(fNewStr, r)
				if !saved {
					fleeLog = append(fleeLog, types.LogEntry{Text: "Critical damage! You fail your save and collapse.", Type: "danger"})
					s := copyState(state)
					s.Phase = types.PhaseDead
					s.Character = &newChar
					s.Combat = nil
					s.Log = fleeLog
					s.RngState = r.GetState()
					return s
				}
				fleeLog = append(fleeLog, types.LogEntry{Text: "Critical damage! You barely hold on.", Type: "danger"})
			}

			s := copyState(state)
			s.Phase = types.PhaseExploring
			s.Character = &newChar
			fleeDungeon(s)
			s.Combat = nil
			s.Log = []types.LogEntry{{Text: "You escape, wounded.", Type: "danger"}}
			s.RngState = r.GetState()
			return s
		}

		// No free attack damage
		s := copyState(state)
		s.Phase = types.PhaseExploring
		fleeDungeon(s)
		s.Combat = nil
		s.Log = []types.LogEntry{{Text: "You scramble away.", Type: "info"}}
		s.RngState = r.GetState()
		return s

	case "run_past":
		if state.Combat == nil || state.Character == nil {
			return state
		}
		char := *state.Character
		monster := state.Combat.Monster
		roll := r.RollDie(20)
		success := roll <= char.Dex

		// Navigate forward
		var nextRoom *types.DungeonRoom
		s := copyState(state)

		if s.Sprint != nil {
			nextIndex := s.Sprint.CurrentRoomIndex + 1
			if nextIndex >= len(s.Sprint.Rooms) {
				return state
			}
			nextRoom = &s.Sprint.Rooms[nextIndex]
			d := *s.Sprint
			d.CurrentRoomIndex = nextIndex
			d = dungeon.MarkSprintVisited(d, nextIndex)
			s.Sprint = &d
		} else if s.Expedition != nil {
			exp := s.Expedition
			exits := state.Exits()
			// Pick a non-retreat exit
			var chosenExit *types.ExitDef
			for i := range exits {
				e := &exits[i]
				if exp.PreviousPos == nil {
					chosenExit = e
					break
				}
				offset := directionOffsets[e.Direction]
				target := types.Pos{X: exp.PlayerPos.X + offset.X, Y: exp.PlayerPos.Y + offset.Y}
				if target.X != exp.PreviousPos.X || target.Y != exp.PreviousPos.Y {
					chosenExit = e
					break
				}
			}
			if chosenExit == nil && len(exits) > 0 {
				chosenExit = &exits[0]
			}
			if chosenExit == nil {
				return state
			}
			d := dungeon.NavigateExpedition(*exp, chosenExit.Direction)
			if d.PlayerPos == exp.PlayerPos {
				return state
			}
			s.Expedition = &d
			updateExpeditionExits(s)
			room := d.Grid[d.PlayerPos.Y][d.PlayerPos.X]
			if room != nil {
				nextRoom = &room.DungeonRoom
			}
		} else {
			return state
		}

		if nextRoom == nil {
			return state
		}

		if success {
			// Mark visited for expedition (sprint already done above)
			if s.Expedition != nil {
				markVisited(s)
			}
			newLight := state.Light
			foeText := "the " + monster.Base.Name
			if foeLabel(monster.Base.Name, state.Light) == "Something" {
				foeText = "it"
			}
			logEntries := []types.LogEntry{
				{Text: fmt.Sprintf("You sprint past %s! (rolled %d vs DEX %d)", foeText, roll, char.Dex), Type: "info"},
				{Text: fmt.Sprintf("You enter %s.", nextRoom.Name), Type: "info"},
			}

			// Check next room encounter
			if !nextRoom.Cleared && nextRoom.Encounter != nil {
				if nextRoom.Encounter.Type == "monster" && nextRoom.Encounter.Monster != nil {
					monsterName := nextRoom.Encounter.Monster.Base.Name
					newBand := types.GetLightBand(newLight)
					combatEntry := fmt.Sprintf("A %s blocks your path!", monsterName)
					if newBand == types.LightDark || newBand == types.LightBlack {
						combatEntry = "Something lurches at you from the dark!"
					}
					monsterCopy := *nextRoom.Encounter.Monster
					s.Phase = types.PhaseCombat
					s.Combat = &types.CombatState{
						Monster:            monsterCopy,
						Round:              1,
						Log:                []string{combatEntry},
						MonsterStunned:     false,
						PlayerHasAdvantage: false,
					}
					s.Log = logEntries
					s.RngState = r.GetState()
					return s
				}
			}

			if len(nextRoom.Loot) > 0 && !nextRoom.Cleared {
				s.Phase = types.PhaseLooting
				markCleared(s)
				s.PendingLoot = nextRoom.Loot
			} else {
				s.Phase = types.PhaseExploring
				if !nextRoom.Cleared {
					markCleared(s)
				}
				s.PendingLoot = []types.Item{}
			}
			s.Combat = nil
			s.Log = logEntries
			s.RngState = r.GetState()
			return s
		}

		// Failed run_past — free attack, stay in combat
		freeAttackDamage := r.RollNotation(monster.Base.Attack.Die)
		fNewHP, fNewStr, fStrDmg := ResolveDamage(freeAttackDamage, char.Armor, char.HP, char.Str)
		combatLog := []string{
			fmt.Sprintf("You try to dash past — failed! (rolled %d vs DEX %d)", roll, char.Dex),
			fmt.Sprintf("%s strikes: %d damage.", foeLabel(monster.Base.Name, state.Light), freeAttackDamage),
		}

		if char.Armor > 0 {
			effective := max(0, freeAttackDamage-char.Armor)
			combatLog = append(combatLog, fmt.Sprintf("Armor absorbs %d, %d gets through.", char.Armor, effective))
		}
		if fStrDmg > 0 {
			combatLog = append(combatLog, fmt.Sprintf("HP gone — %d damage to STR!", fStrDmg))
		}

		newChar := char
		newChar.HP = fNewHP
		newChar.Str = fNewStr

		if fNewStr <= 0 {
			combatLog = append(combatLog, "Your body gives out.")
			deathLog := make([]types.LogEntry, len(combatLog))
			for i, text := range combatLog {
				deathLog[i] = types.LogEntry{Text: text, Type: "combat"}
			}
			// Revert to original state for the death — we didn't actually move
			rs := copyState(state)
			rs.Phase = types.PhaseDead
			rs.Character = &newChar
			rs.Combat = nil
			rs.Log = deathLog
			rs.RngState = r.GetState()
			return rs
		}

		if fStrDmg > 0 && fNewStr > 0 {
			saved := CriticalDamageSave(fNewStr, r)
			if !saved {
				combatLog = append(combatLog, "Critical damage! You fail your save and collapse.")
				deathLog := make([]types.LogEntry, len(combatLog))
				for i, text := range combatLog {
					deathLog[i] = types.LogEntry{Text: text, Type: "combat"}
				}
				rs := copyState(state)
				rs.Phase = types.PhaseDead
				rs.Character = &newChar
				rs.Combat = nil
				rs.Log = deathLog
				rs.RngState = r.GetState()
				return rs
			}
			combatLog = append(combatLog, "Critical damage! You barely hold on.")
		}

		// Stay in combat — revert navigation
		rs := copyState(state)
		rs.Character = &newChar
		rs.Combat = &types.CombatState{
			Monster:            state.Combat.Monster,
			Round:              state.Combat.Round + 1,
			Log:                combatLog,
			MonsterStunned:     false,
			PlayerHasAdvantage: false,
		}
		rs.RngState = r.GetState()
		return rs

	case "creative_action_result":
		if state.Combat == nil || state.Character == nil || action.CreativeResult == nil {
			return state
		}
		result := action.CreativeResult
		char := *state.Character
		monster := state.Combat.Monster
		var combatLog []string
		crDark := types.GetLightBand(state.Light) == types.LightDark || types.GetLightBand(state.Light) == types.LightBlack
		crFoe := monster.Base.Name
		if crDark {
			crFoe = "Something"
		}

		// If GM disallowed, show reason — no turn spent
		if !result.Allowed {
			combatLog = append(combatLog, fmt.Sprintf("The GM rules: %s", result.Reason))
			s := copyState(state)
			s.Combat = &types.CombatState{
				Monster:            state.Combat.Monster,
				Round:              state.Combat.Round,
				Log:                combatLog,
				MonsterStunned:     state.Combat.MonsterStunned,
				PlayerHasAdvantage: state.Combat.PlayerHasAdvantage,
			}
			s.RngState = r.GetState()
			return s
		}

		// Calculate save target
		var statValue int
		switch strings.ToLower(result.SaveStat) {
		case "str":
			statValue = char.Str
		case "dex":
			statValue = char.Dex
		case "wil":
			statValue = char.Wil
		default:
			statValue = char.Str
		}
		traitBonus := len(result.TraitMatches) * 2
		band := types.GetLightBand(state.Light)
		lightPenalty := 0
		if band == types.LightDim {
			lightPenalty = -2
		} else if band == types.LightDark || band == types.LightBlack {
			lightPenalty = -4
		}
		target := statValue + result.Modifier + traitBonus + lightPenalty
		crRoll := r.RollDie(20)
		crSuccess := crRoll <= target

		modStr := fmt.Sprintf("%+d", result.Modifier)
		traitStr := ""
		if traitBonus > 0 {
			traitStr = fmt.Sprintf("+%d traits", traitBonus)
		}
		lightStr := ""
		if lightPenalty < 0 {
			lightStr = fmt.Sprintf("%d dark", lightPenalty)
		}
		combatLog = append(combatLog, fmt.Sprintf("Creative action! %s save: %d vs %d (%d%s%s%s)",
			strings.ToUpper(result.SaveStat), crRoll, target, statValue, modStr, traitStr, lightStr))

		if crSuccess {
			combatLog = append(combatLog, result.SuccessNarration)

			// Consume item if applicable
			if result.ConsumesItem != nil {
				slotIdx := -1
				for i, item := range char.Inventory {
					if item != nil && item.ID == *result.ConsumesItem {
						slotIdx = i
						break
					}
				}
				if slotIdx >= 0 {
					itemName := char.Inventory[slotIdx].Name
					char = DropItem(char, slotIdx)
					combatLog = append(combatLog, fmt.Sprintf("(%s used up)", itemName))
				}
			}

			switch result.Effect {
			case "damage":
				dmg := r.RollNotation(result.EffectDie)
				dmgNewHP, dmgNewStr, dmgStrDmg := ResolveDamage(dmg, monster.Base.Armor, monster.CurrentHP, monster.CurrentStr)
				combatLog = append(combatLog, fmt.Sprintf("Deals %d damage!", dmg))

				newMonster := monster
				newMonster.CurrentHP = dmgNewHP
				newMonster.CurrentStr = dmgNewStr

				if dmgNewStr <= 0 {
					if crDark {
						combatLog = append(combatLog, "You hear something collapse.")
					} else {
						combatLog = append(combatLog, fmt.Sprintf("%s is destroyed!", monster.Base.Name))
					}
					return handleMonsterDeath(state, char, monster, crDark, r)
				}

				// Crit save on monster
				if dmgStrDmg > 0 && dmgNewStr > 0 {
					monsterSaved := CriticalDamageSave(dmgNewStr, r)
					if !monsterSaved {
						if crDark {
							combatLog = append(combatLog, "You hear something collapse.")
						} else {
							combatLog = append(combatLog, fmt.Sprintf("%s fails its critical damage save and falls!", monster.Base.Name))
						}
						return handleMonsterDeath(state, char, monster, crDark, r)
					}
				}

				// Monster still alive — it attacks back
				monsterDmg := r.RollNotation(monster.Base.Attack.Die)
				newChar, dead, combatLog := handlePlayerDamage(char, monsterDmg, combatLog, crFoe, r)
				if dead {
					deathLog := make([]types.LogEntry, len(combatLog))
					for i, text := range combatLog {
						deathLog[i] = types.LogEntry{Text: text, Type: "combat"}
					}
					rs := copyState(state)
					rs.Phase = types.PhaseDead
					rs.Character = &newChar
					rs.Combat = nil
					rs.Log = deathLog
					rs.RngState = r.GetState()
					return rs
				}

				rs := copyState(state)
				rs.Character = &newChar
				rs.Combat = &types.CombatState{
					Monster:            newMonster,
					Round:              state.Combat.Round + 1,
					Log:                combatLog,
					MonsterStunned:     false,
					PlayerHasAdvantage: false,
				}
				rs.RngState = r.GetState()
				return rs

			case "stun":
				combatLog = append(combatLog, fmt.Sprintf("%s is stunned! It cannot attack this round.", crFoe))
				rs := copyState(state)
				rs.Character = &char
				rs.Combat = &types.CombatState{
					Monster:            state.Combat.Monster,
					Round:              state.Combat.Round + 1,
					Log:                combatLog,
					MonsterStunned:     true,
					PlayerHasAdvantage: false,
				}
				rs.RngState = r.GetState()
				return rs

			case "frighten":
				if crDark {
					combatLog = append(combatLog, "You hear something scramble away in terror!")
				} else {
					combatLog = append(combatLog, fmt.Sprintf("%s flees in terror!", monster.Base.Name))
				}
				rs := copyState(state)
				markCleared(rs)
				rs.MonstersKilled++
				room := state.CurrentRoom()
				if room != nil && len(room.Loot) > 0 {
					rs.Phase = types.PhaseLooting
					rs.Character = &char
					rs.Combat = nil
					rs.PendingLoot = room.Loot
					fleeText := fmt.Sprintf("%s flees. You notice something on the ground.", monster.Base.Name)
					if crDark {
						fleeText = "It flees. You notice something on the ground."
					}
					rs.Log = []types.LogEntry{{Text: fleeText, Type: "loot"}}
					rs.RngState = r.GetState()
					return rs
				}
				rs.Phase = types.PhaseExploring
				rs.Character = &char
				rs.Combat = nil
				fleeText := fmt.Sprintf("%s flees into the dark.", monster.Base.Name)
				if crDark {
					fleeText = "It flees into the dark."
				}
				rs.Log = []types.LogEntry{{Text: fleeText, Type: "combat"}}
				rs.RngState = r.GetState()
				return rs

			case "advantage":
				combatLog = append(combatLog, "You gain the upper hand! Next attack deals double damage.")
				// Monster still attacks
				monsterDmg := r.RollNotation(monster.Base.Attack.Die)
				newChar, dead, combatLog := handlePlayerDamage(char, monsterDmg, combatLog, crFoe, r)
				if dead {
					deathLog := make([]types.LogEntry, len(combatLog))
					for i, text := range combatLog {
						deathLog[i] = types.LogEntry{Text: text, Type: "combat"}
					}
					rs := copyState(state)
					rs.Phase = types.PhaseDead
					rs.Character = &newChar
					rs.Combat = nil
					rs.Log = deathLog
					rs.RngState = r.GetState()
					return rs
				}

				rs := copyState(state)
				rs.Character = &newChar
				rs.Combat = &types.CombatState{
					Monster:            state.Combat.Monster,
					Round:              state.Combat.Round + 1,
					Log:                combatLog,
					MonsterStunned:     false,
					PlayerHasAdvantage: true,
				}
				rs.RngState = r.GetState()
				return rs
			}
		}

		// Failed creative action — monster attacks
		combatLog = append(combatLog, result.FailureNarration)
		monsterDmg := r.RollNotation(monster.Base.Attack.Die)
		newChar, dead, combatLog := handlePlayerDamage(char, monsterDmg, combatLog, crFoe, r)
		if dead {
			deathLog := make([]types.LogEntry, len(combatLog))
			for i, text := range combatLog {
				deathLog[i] = types.LogEntry{Text: text, Type: "combat"}
			}
			rs := copyState(state)
			rs.Phase = types.PhaseDead
			rs.Character = &newChar
			rs.Combat = nil
			rs.Log = deathLog
			rs.RngState = r.GetState()
			return rs
		}

		rs := copyState(state)
		rs.Character = &newChar
		rs.Combat = &types.CombatState{
			Monster:            state.Combat.Monster,
			Round:              state.Combat.Round + 1,
			Log:                combatLog,
			MonsterStunned:     false,
			PlayerHasAdvantage: false,
		}
		rs.RngState = r.GetState()
		return rs

	case "use_item":
		if state.Character == nil {
			return state
		}
		if action.SlotIndex < 0 || action.SlotIndex >= len(state.Character.Inventory) {
			return state
		}
		item := state.Character.Inventory[action.SlotIndex]
		if item == nil || item.UseEffect == nil {
			return state
		}

		var logEntries []types.LogEntry
		newChar := *state.Character

		switch item.UseEffect.Type {
		case "heal_str":
			roll := r.RollNotation(item.UseEffect.Die)
			newStr := min(18, newChar.Str+roll)
			healed := newStr - newChar.Str
			newChar.Str = newStr
			logEntries = append(logEntries, types.LogEntry{Text: fmt.Sprintf("Used %s: restored %d STR.", item.Name, healed), Type: "info"})
		case "heal_hp":
			roll := r.RollNotation(item.UseEffect.Die)
			newHP := min(newChar.MaxHP, newChar.HP+roll)
			healed := newHP - newChar.HP
			newChar.HP = newHP
			logEntries = append(logEntries, types.LogEntry{Text: fmt.Sprintf("Used %s: restored %d HP.", item.Name, healed), Type: "info"})
		case "restore_light":
			newLight := min(10, item.UseEffect.Level)
			msg := "A faint glow. Better than nothing."
			if newLight >= 8 {
				msg = "Light!"
			}
			logEntries = append(logEntries, types.LogEntry{Text: fmt.Sprintf("Used %s. %s", item.Name, msg), Type: "info"})
			newChar = DropItem(newChar, action.SlotIndex)
			s := copyState(state)
			s.Character = &newChar
			s.Light = newLight
			s.Log = logEntries
			s.RngState = r.GetState()
			return s
		}

		// Consume the item
		newChar = DropItem(newChar, action.SlotIndex)
		s := copyState(state)
		s.Character = &newChar
		s.Log = logEntries
		s.RngState = r.GetState()
		return s

	case "take_item":
		if state.Character == nil {
			return state
		}
		if action.ItemIndex < 0 || action.ItemIndex >= len(state.PendingLoot) {
			return state
		}
		item := state.PendingLoot[action.ItemIndex]

		newChar := AddItem(*state.Character, item)
		if newChar == nil {
			s := copyState(state)
			s.Log = []types.LogEntry{{Text: fmt.Sprintf("No room for %s. Drop something first.", item.Name), Type: "system"}}
			s.RngState = r.GetState()
			return s
		}

		newLoot := make([]types.Item, 0, len(state.PendingLoot)-1)
		for i, l := range state.PendingLoot {
			if i != action.ItemIndex {
				newLoot = append(newLoot, l)
			}
		}

		s := copyState(state)
		s.Character = newChar
		s.PendingLoot = newLoot
		if len(newLoot) > 0 {
			s.Phase = types.PhaseLooting
		} else {
			s.Phase = types.PhaseExploring
		}
		s.Log = []types.LogEntry{{Text: fmt.Sprintf("Took %s.", item.Name), Type: "loot"}}
		s.RngState = r.GetState()
		return s

	case "drop_item":
		if state.Character == nil {
			return state
		}
		dropped := DropItem(*state.Character, action.SlotIndex)
		s := copyState(state)
		s.Character = &dropped
		s.RngState = r.GetState()
		return s

	case "equip_item":
		if state.Character == nil {
			return state
		}
		if action.SlotIndex < 0 || action.SlotIndex >= len(state.Character.Inventory) {
			return state
		}
		item := state.Character.Inventory[action.SlotIndex]
		if item == nil {
			return state
		}

		newInv := make([]*types.Item, len(state.Character.Inventory))
		copy(newInv, state.Character.Inventory)

		// Find all slots occupied by the selected item (match by ID for multi-slot items)
		var itemSlots []int
		for i, it := range newInv {
			if it != nil && it.ID == item.ID {
				itemSlots = append(itemSlots, i)
			}
		}

		// Find what's currently at slot 0
		displaced := newInv[0]
		var displacedSlots []int
		if displaced != nil {
			for i, it := range newInv {
				if it != nil && it.ID == displaced.ID {
					displacedSlots = append(displacedSlots, i)
				}
			}
		}

		// If already at slot 0, no-op
		if len(itemSlots) > 0 && itemSlots[0] == 0 {
			return state
		}

		// Clear both items from their current positions
		for _, i := range itemSlots {
			newInv[i] = nil
		}
		for _, i := range displacedSlots {
			newInv[i] = nil
		}

		// Place the selected item at slot 0
		for i := 0; i < item.Slots && i < len(newInv); i++ {
			newInv[i] = item
		}

		// Place the displaced item where the selected item was
		if displaced != nil && len(itemSlots) > 0 {
			targetStart := itemSlots[0]
			for i := 0; i < displaced.Slots && targetStart+i < len(newInv); i++ {
				newInv[targetStart+i] = displaced
			}
		}

		newChar := *state.Character
		newChar.Inventory = newInv
		newChar.Armor = RecalculateArmor(newChar)

		s := copyState(state)
		s.Character = &newChar
		s.RngState = r.GetState()
		return s

	case "skip_loot":
		s := copyState(state)
		s.Phase = types.PhaseExploring
		s.PendingLoot = []types.Item{}
		s.RngState = r.GetState()
		return s

	case "rest":
		if state.Character == nil {
			return state
		}
		restChance := math.Min(0.2+float64(state.RestsTaken)*0.2, 1.0)
		disturbed := r.Chance(restChance)
		restLight := max(0, state.Light-1)

		if disturbed {
			wanderers := content.GetMonstersByTier("weak", cd)
			if len(wanderers) == 0 {
				return state
			}
			base := rng.Pick(r, wanderers)
			logEntries := []types.LogEntry{
				{Text: fmt.Sprintf("Something stirs in the dark... a %s finds you!", base.Name), Type: "danger"},
			}
			s := copyState(state)
			s.Phase = types.PhaseCombat
			s.Combat = &types.CombatState{
				Monster: types.MonsterInstance{
					Base:       base,
					CurrentHP:  base.HP,
					CurrentStr: base.Str,
				},
				Round:              1,
				Log:                []string{fmt.Sprintf("A %s catches you resting!", base.Name)},
				MonsterStunned:     false,
				PlayerHasAdvantage: false,
			}
			s.Log = logEntries
			s.RestsTaken = state.RestsTaken + 1
			s.Light = restLight
			s.RngState = r.GetState()
			return s
		}

		healRoll := r.RollDie(8)
		newHP := min(state.Character.MaxHP, state.Character.HP+healRoll)
		healed := newHP - state.Character.HP
		nextChance := math.Min(0.2+float64(state.RestsTaken+1)*0.2, 1.0)
		warning := ""
		if nextChance >= 1.0 {
			warning = "The dungeon feels alive around you. Rest is no longer safe."
		} else if nextChance >= 0.6 {
			warning = "You hear movement in the distance."
		}
		logEntries := []types.LogEntry{
			{Text: fmt.Sprintf("You rest and recover %d HP.", healed), Type: "info"},
		}
		if warning != "" {
			logEntries = append(logEntries, types.LogEntry{Text: warning, Type: "danger"})
		}

		newChar := *state.Character
		newChar.HP = newHP
		s := copyState(state)
		s.Character = &newChar
		s.Log = logEntries
		s.RestsTaken = state.RestsTaken + 1
		s.Light = restLight
		s.RngState = r.GetState()
		return s

	case "restart":
		s := CreateInitialState(0)
		return &s

	case "open_ossuary":
		s := copyState(state)
		s.Phase = types.PhaseOssuary
		return s

	case "return_to_title":
		s := copyState(state)
		s.Phase = types.PhaseTitle
		return s

	case "start_game_setup":
		s := copyState(state)
		s.Phase = types.PhaseGameSetup
		s.RngState = r.GetState()
		return s

	case "select_type":
		s := copyState(state)
		s.GameType = action.SelectedType
		s.RngState = r.GetState()
		return s

	default:
		return state
	}
}
