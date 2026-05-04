package dungeon

import (
	"fmt"

	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// Direction constants and helpers

var directionOffsets = map[string]types.Pos{
	"n": {X: 0, Y: -1},
	"e": {X: 1, Y: 0},
	"s": {X: 0, Y: 1},
	"w": {X: -1, Y: 0},
}

var directions = []string{"n", "e", "s", "w"}

var dirNames = map[string]string{
	"n": "North",
	"e": "East",
	"s": "South",
	"w": "West",
}

// wallOpen returns whether a wall is open (passable) for the given direction string.
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

// setWall sets a wall direction to open (true) or closed (false).
func setWall(walls *types.Walls, dir string, open bool) {
	switch dir {
	case "n":
		walls.N = open
	case "e":
		walls.E = open
	case "s":
		walls.S = open
	case "w":
		walls.W = open
	}
}

// PosKey returns a string key for a position.
func PosKey(pos types.Pos) string {
	return fmt.Sprintf("%d,%d", pos.X, pos.Y)
}

// GetRoom safely retrieves a room from the grid, returning nil if out of bounds or empty.
func GetRoom(grid [][]*types.ExpeditionRoom, pos types.Pos) *types.ExpeditionRoom {
	if pos.Y < 0 || pos.Y >= len(grid) {
		return nil
	}
	row := grid[pos.Y]
	if pos.X < 0 || pos.X >= len(row) {
		return nil
	}
	return row[pos.X]
}

// containsString checks if a string slice contains a value.
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// ComputePeeked returns keys of adjacent rooms visible through open walls, excluding given keys.
func ComputePeeked(grid [][]*types.ExpeditionRoom, pos types.Pos, exclude []string) []string {
	room := GetRoom(grid, pos)
	if room == nil {
		return nil
	}
	var peeked []string
	for _, dir := range directions {
		if !wallOpen(room.Walls, dir) {
			continue
		}
		offset := directionOffsets[dir]
		neighborPos := types.Pos{X: pos.X + offset.X, Y: pos.Y + offset.Y}
		key := PosKey(neighborPos)
		if containsString(exclude, key) {
			continue
		}
		if GetRoom(grid, neighborPos) != nil {
			peeked = append(peeked, key)
		}
	}
	return peeked
}

// BuildExits builds the exit list for a room based on open walls leading to existing rooms.
func BuildExits(grid [][]*types.ExpeditionRoom, room *types.ExpeditionRoom, entry *types.Pos, previousPos *types.Pos) []types.ExitDef {
	var exits []types.ExitDef
	for _, dir := range directions {
		if !wallOpen(room.Walls, dir) {
			continue
		}
		offset := directionOffsets[dir]
		neighborPos := types.Pos{X: room.Pos.X + offset.X, Y: room.Pos.Y + offset.Y}
		neighborRoom := GetRoom(grid, neighborPos)
		if neighborRoom == nil {
			continue
		}
		isEntry := entry != nil && neighborPos.X == entry.X && neighborPos.Y == entry.Y
		hint := GetExitHint(neighborRoom, neighborPos.X, neighborPos.Y, isEntry)
		isRetreat := previousPos != nil && neighborPos.X == previousPos.X && neighborPos.Y == previousPos.Y
		var label string
		if isRetreat {
			label = fmt.Sprintf("← %s — %s", dirNames[dir], hint)
		} else {
			label = fmt.Sprintf("%s — %s", dirNames[dir], hint)
		}
		exits = append(exits, types.ExitDef{Label: label, Direction: dir})
	}
	return exits
}

// CreateExpeditionDungeon initializes an expedition dungeon at the entry position.
func CreateExpeditionDungeon(grid [][]*types.ExpeditionRoom, entry types.Pos, bossPos types.Pos, roomCount int) types.ExpeditionDungeon {
	visited := []string{PosKey(entry)}
	peeked := ComputePeeked(grid, entry, visited)
	return types.ExpeditionDungeon{
		Grid:        grid,
		PlayerPos:   entry,
		PreviousPos: nil,
		Entry:       entry,
		BossPos:     bossPos,
		RoomCount:   roomCount,
		Visited:     visited,
		Peeked:      peeked,
	}
}

// NavigateExpedition moves the player in the given direction if possible.
func NavigateExpedition(d types.ExpeditionDungeon, direction string) types.ExpeditionDungeon {
	currentRoom := GetRoom(d.Grid, d.PlayerPos)
	if currentRoom == nil || !wallOpen(currentRoom.Walls, direction) {
		return d
	}
	offset := directionOffsets[direction]
	targetPos := types.Pos{X: d.PlayerPos.X + offset.X, Y: d.PlayerPos.Y + offset.Y}
	targetRoom := GetRoom(d.Grid, targetPos)
	if targetRoom == nil {
		return d
	}
	targetKey := PosKey(targetPos)
	var newVisited []string
	if containsString(d.Visited, targetKey) {
		newVisited = d.Visited
	} else {
		newVisited = append(append([]string{}, d.Visited...), targetKey)
	}
	newPeeked := ComputePeeked(d.Grid, targetPos, newVisited)
	prev := d.PlayerPos
	d.PreviousPos = &prev
	d.PlayerPos = targetPos
	d.Visited = newVisited
	d.Peeked = newPeeked
	return d
}

// MarkExpeditionCleared marks the current room as cleared.
func MarkExpeditionCleared(d types.ExpeditionDungeon) types.ExpeditionDungeon {
	room := GetRoom(d.Grid, d.PlayerPos)
	if room != nil {
		room.Cleared = true
	}
	return d
}

// MarkExpeditionVisited marks the current room as visited.
func MarkExpeditionVisited(d types.ExpeditionDungeon) types.ExpeditionDungeon {
	room := GetRoom(d.Grid, d.PlayerPos)
	if room != nil {
		room.Visited = true
	}
	return d
}

// Exit hint pools

var hintOminous = []string{
	"something stirs beyond",
	"a low growl echoes",
	"the stench of rot",
	"scratching against stone",
	"a shape moves in the dark",
	"the air tastes of iron",
}

var hintCuriosity = []string{
	"a glint of metal",
	"something catches the light",
	"the sound of dripping water",
	"a faint shimmer",
	"an alcove holds something",
	"dust disturbed recently",
}

var hintNeutral = []string{
	"quiet passage",
	"cold stone",
	"echoing footsteps",
	"still air",
	"worn flagstones",
	"empty silence",
	"a draft from somewhere",
}

var hintDread = []string{
	"oppressive heat",
	"the air thrums with menace",
	"a presence watches",
	"the walls seem to breathe",
	"dread pools in your gut",
	"something ancient waits",
}

const entryHint = "the way you came in"

// CategorizeRoom categorizes a room for exit hint purposes.
func CategorizeRoom(room *types.ExpeditionRoom, isEntry bool) string {
	if isEntry {
		return "entry"
	}
	if !room.Cleared && room.Encounter != nil && room.Encounter.Type == "monster" {
		if room.Encounter.Monster != nil && room.Encounter.Monster.Base.Tier == "boss" {
			return "dread"
		}
		return "ominous"
	}
	if !room.Cleared && len(room.Loot) > 0 {
		return "curiosity"
	}
	return "neutral"
}

// GetExitHint returns a deterministic hint string for an exit leading to the given room.
func GetExitHint(room *types.ExpeditionRoom, x, y int, isEntry bool) string {
	category := CategorizeRoom(room, isEntry)
	if category == "entry" {
		return entryHint
	}
	var pool []string
	switch category {
	case "dread":
		pool = hintDread
	case "ominous":
		pool = hintOminous
	case "curiosity":
		pool = hintCuriosity
	default:
		pool = hintNeutral
	}
	idx := (x*31 + y*17)
	if idx < 0 {
		idx = -idx
	}
	idx = idx % len(pool)
	return pool[idx]
}
