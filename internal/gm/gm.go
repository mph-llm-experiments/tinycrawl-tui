package gm

import (
	"fmt"
	"math"
	"strings"

	"github.com/mph-llm-experiments/tinycrawl-tui/internal/dungeon"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

const GmJSONSchema = `Respond ONLY with valid JSON matching this exact schema (no markdown, no explanation outside the JSON):
{
  "allowed": boolean,
  "reason": "1 sentence explaining your ruling",
  "save_stat": "str" | "dex" | "wil",
  "modifier": number from -2 to +2 (situational difficulty),
  "trait_matches": ["trait1", ...] (item traits that match monster weaknesses),
  "effect": "damage" | "stun" | "frighten" | "advantage",
  "effect_die": "d4" | "d6" | "d8" | "d10" | "d12",
  "consumes_item": "item_id" | null (if the action uses up, destroys, or tosses away an item, set this to that item's id; null if the item is reusable),
  "success_narration": "1-2 sentences, what happens on success",
  "failure_narration": "1-2 sentences, what happens on failure"
}`

const DefaultGMPersonality = `You are the GM for a dungeon crawler. You evaluate creative actions proposed by the player during combat. You are generous and imaginative — if an action is physically plausible and uses items the player actually has, allow it. Reward cleverness. Only reject actions that are physically impossible, reference items the player doesn't have, or make no sense given the situation.`

var (
	validEffects = []string{"damage", "stun", "frighten", "advantage"}
	validStats   = []string{"str", "dex", "wil"}
	validDice    = []string{"d4", "d6", "d8", "d10", "d12"}
)

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// wallOpen checks if a wall is open for the given direction using the Walls struct directly.
// We cannot call dungeon.wallOpen (unexported), so we access the struct fields.
func wallOpen(walls types.Walls, dir string) bool {
	switch dir {
	case "n":
		return walls.N
	case "e":
		return walls.E
	case "s":
		return walls.S
	case "w":
		return walls.W
	}
	return false
}

func buildSpatialContext(exp *types.ExpeditionDungeon) string {
	room := dungeon.GetRoom(exp.Grid, exp.PlayerPos)
	if room == nil {
		return ""
	}

	dirLabels := map[string]string{"n": "north", "e": "east", "s": "south", "w": "west"}
	offsets := map[string]types.Pos{
		"n": {X: 0, Y: -1},
		"e": {X: 1, Y: 0},
		"s": {X: 0, Y: 1},
		"w": {X: -1, Y: 0},
	}

	var neighbors []string
	for _, dir := range []string{"n", "e", "s", "w"} {
		if !wallOpen(room.Walls, dir) {
			continue
		}
		off := offsets[dir]
		npos := types.Pos{X: room.Pos.X + off.X, Y: room.Pos.Y + off.Y}
		neighbor := dungeon.GetRoom(exp.Grid, npos)
		if neighbor == nil {
			continue
		}
		key := dungeon.PosKey(npos)
		explored := false
		for _, v := range exp.Visited {
			if v == key {
				explored = true
				break
			}
		}
		label := dirLabels[dir]
		if explored {
			neighbors = append(neighbors, fmt.Sprintf("To the %s: %s.", label, neighbor.Name))
		} else {
			neighbors = append(neighbors, fmt.Sprintf("To the %s: an unexplored passage.", label))
		}
	}

	if len(neighbors) > 0 {
		return "\nNearby: " + strings.Join(neighbors, " ")
	}
	return ""
}

// BuildGmSystemPrompt assembles the GM system prompt from personality, JSON schema,
// and optional spatial context from an expedition dungeon.
func BuildGmSystemPrompt(personality string, exp *types.ExpeditionDungeon) string {
	if personality == "" {
		personality = DefaultGMPersonality
	}
	prompt := personality + "\n\n" + GmJSONSchema
	if exp != nil {
		prompt += buildSpatialContext(exp)
	}
	return prompt
}

// BuildCreativeActionPrompt assembles the user prompt for a creative combat action.
func BuildCreativeActionPrompt(monster types.MonsterInstance, char types.Character, room types.DungeonRoom, actionText string) string {
	m := monster.Base
	weaknesses := "none"
	if len(m.Weaknesses) > 0 {
		weaknesses = strings.Join(m.Weaknesses, ", ")
	}

	seen := map[string]bool{}
	var invLines []string
	for _, item := range char.Inventory {
		if item == nil || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		line := "- " + item.Name
		if item.Damage != "" {
			line += fmt.Sprintf(" (%s)", item.Damage)
		}
		if len(item.Traits) > 0 {
			line += fmt.Sprintf(" [traits: %s]", strings.Join(item.Traits, ", "))
		}
		invLines = append(invLines, line)
	}

	return fmt.Sprintf(`MONSTER: %s (STR %d, DEX %d, WIL %d, HP %d, Armor %d, attacks with %s %s)
WEAKNESSES: %s
ROOM: %s — %s

PLAYER INVENTORY:
%s

PLAYER ACTION: "%s"

Evaluate this action.`, m.Name, m.Str, m.Dex, m.Wil, monster.CurrentHP, m.Armor, m.Attack.Name, m.Attack.Die, weaknesses, room.Name, room.Description, strings.Join(invLines, "\n"), actionText)
}

// ValidateGmResponse validates and sanitizes the raw GM JSON response into a CreativeActionResult.
func ValidateGmResponse(raw map[string]any) (*types.CreativeActionResult, error) {
	allowed, ok := raw["allowed"].(bool)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'allowed' field")
	}

	reason, _ := raw["reason"].(string)

	saveStat, _ := raw["save_stat"].(string)
	if !contains(validStats, saveStat) {
		return nil, fmt.Errorf("invalid save_stat: %s", saveStat)
	}

	effect, _ := raw["effect"].(string)
	if !contains(validEffects, effect) {
		return nil, fmt.Errorf("invalid effect: %s", effect)
	}

	modifier := 0
	if m, ok := raw["modifier"].(float64); ok {
		modifier = int(math.Max(-2, math.Min(2, math.Round(m))))
	}

	effectDie := "d6"
	if d, ok := raw["effect_die"].(string); ok && contains(validDice, d) {
		effectDie = d
	}

	var traitMatches []string
	if arr, ok := raw["trait_matches"].([]any); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok {
				traitMatches = append(traitMatches, s)
			}
		}
	}

	var consumesItem *string
	if ci, ok := raw["consumes_item"].(string); ok {
		consumesItem = &ci
	}

	successNarration, _ := raw["success_narration"].(string)
	failureNarration, _ := raw["failure_narration"].(string)

	return &types.CreativeActionResult{
		Allowed:          allowed,
		Reason:           reason,
		SaveStat:         saveStat,
		Modifier:         modifier,
		TraitMatches:     traitMatches,
		Effect:           effect,
		EffectDie:        effectDie,
		ConsumesItem:     consumesItem,
		SuccessNarration: successNarration,
		FailureNarration: failureNarration,
	}, nil
}

// VerifyTraitMatches filters claimed traits to only those present in both
// the player's inventory and the monster's weaknesses.
func VerifyTraitMatches(claimed []string, char types.Character, monster types.MonsterInstance) []string {
	playerTraits := map[string]bool{}
	for _, item := range char.Inventory {
		if item == nil {
			continue
		}
		for _, t := range item.Traits {
			playerTraits[t] = true
		}
	}

	monsterWeaknesses := map[string]bool{}
	for _, w := range monster.Base.Weaknesses {
		monsterWeaknesses[w] = true
	}

	var verified []string
	for _, t := range claimed {
		if playerTraits[t] && monsterWeaknesses[t] {
			verified = append(verified, t)
		}
	}
	return verified
}
