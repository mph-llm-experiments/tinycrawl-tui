package dungeon

import (
	"fmt"
	"math"

	"github.com/mph-llm-experiments/tinycrawl-tui/internal/content"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/rng"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

var opposite = map[string]string{
	"n": "s",
	"s": "n",
	"e": "w",
	"w": "e",
}

// GeneratorResult holds the output of GenerateExpeditionGrid.
type GeneratorResult struct {
	Grid      [][]*types.ExpeditionRoom
	Entry     types.Pos
	BossPos   types.Pos
	RoomCount int
}

// getTierForDistance maps a BFS distance to a monster tier string.
func getTierForDistance(distance, maxDistance int) string {
	if maxDistance == 0 {
		return "weak"
	}
	pct := float64(distance) / float64(maxDistance)
	switch {
	case pct >= 0.9:
		return "boss"
	case pct >= 0.6:
		return "tough"
	case pct >= 0.3:
		return "moderate"
	default:
		return "weak"
	}
}

// distEntry holds BFS state.
type distEntry struct {
	pos  types.Pos
	dist int
}

// computeDistances performs BFS from entry over open walls, returning a map of posKey -> distance.
func computeDistances(grid [][]*types.ExpeditionRoom, entry types.Pos) map[string]int {
	distances := make(map[string]int)
	queue := []distEntry{{pos: entry, dist: 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		key := PosKey(cur.pos)
		if _, seen := distances[key]; seen {
			continue
		}
		distances[key] = cur.dist
		room := GetRoom(grid, cur.pos)
		if room == nil {
			continue
		}
		for _, dir := range directions {
			if !wallOpen(room.Walls, dir) {
				continue
			}
			offset := directionOffsets[dir]
			npos := types.Pos{X: cur.pos.X + offset.X, Y: cur.pos.Y + offset.Y}
			nkey := PosKey(npos)
			if _, seen := distances[nkey]; seen {
				continue
			}
			if GetRoom(grid, npos) != nil {
				queue = append(queue, distEntry{pos: npos, dist: cur.dist + 1})
			}
		}
	}
	return distances
}

// makeMonsterInstance picks a monster of the given tier and returns an instance.
func makeMonsterInstance(r *rng.Rng, monsters []types.Monster, tier string) *types.MonsterInstance {
	var pool []types.Monster
	for _, m := range monsters {
		if m.Tier == tier {
			pool = append(pool, m)
		}
	}
	if len(pool) == 0 {
		return nil
	}
	base := rng.Pick(r, pool)
	return &types.MonsterInstance{
		Base:       base,
		CurrentHP:  base.HP,
		CurrentStr: base.Str,
	}
}

// posKeyXY is a convenience for raw x,y coordinates.
func posKeyXY(x, y int) string {
	return fmt.Sprintf("%d,%d", x, y)
}

// GenerateExpeditionGrid generates a procedural dungeon grid and populates it with content.
func GenerateExpeditionGrid(r *rng.Rng, cd types.ContentData, config types.ExpeditionConfig) GeneratorResult {
	minRooms, maxRooms := config.RoomBudget[0], config.RoomBudget[1]
	budget := minRooms + int(math.Floor(r.Next()*float64(maxRooms-minRooms+1)))

	gridSize := config.GridSize
	grid := make([][]*types.ExpeditionRoom, gridSize)
	for i := range grid {
		grid[i] = make([]*types.ExpeditionRoom, gridSize)
	}

	occupied := make(map[string]bool)
	center := gridSize / 2
	entryPos := types.Pos{X: center, Y: center}

	placeRoom := func(x, y int) *types.ExpeditionRoom {
		room := &types.ExpeditionRoom{
			DungeonRoom: types.DungeonRoom{
				TemplateID: "",
				Name:       "",
				Loot:       []types.Item{},
				Exits:      []types.ExitDef{},
			},
			Pos:   types.Pos{X: x, Y: y},
			Walls: types.Walls{},
		}
		grid[y][x] = room
		occupied[posKeyXY(x, y)] = true
		return room
	}

	placeRoom(entryPos.X, entryPos.Y)
	frontier := []types.Pos{entryPos}

	for len(occupied) < budget && len(frontier) > 0 {
		idx := int(math.Floor(r.Next() * float64(len(frontier))))
		current := frontier[idx]

		// Find candidate neighbors
		type candidate struct {
			pos types.Pos
			dir string
		}
		var candidates []candidate
		for _, dir := range directions {
			offset := directionOffsets[dir]
			nx, ny := current.X+offset.X, current.Y+offset.Y
			if nx >= 0 && nx < gridSize && ny >= 0 && ny < gridSize && !occupied[posKeyXY(nx, ny)] {
				candidates = append(candidates, candidate{pos: types.Pos{X: nx, Y: ny}, dir: dir})
			}
		}

		if len(candidates) == 0 {
			frontier = append(frontier[:idx], frontier[idx+1:]...)
			continue
		}

		// Determine how many exits to open
		exitCount := 1
		if r.Chance(config.BranchProbability) {
			if r.Chance(config.BranchProbability) {
				exitCount = 3
			} else {
				exitCount = 2
			}
		}
		remaining := budget - len(occupied)
		if exitCount > len(candidates) {
			exitCount = len(candidates)
		}
		if exitCount > remaining {
			exitCount = remaining
		}

		// Shuffle candidates
		rng.Shuffle(r, candidates)
		chosen := candidates[:exitCount]

		for _, c := range chosen {
			if len(occupied) >= budget {
				break
			}
			child := placeRoom(c.pos.X, c.pos.Y)
			parent := grid[current.Y][current.X]
			setWall(&parent.Walls, c.dir, true)
			setWall(&child.Walls, opposite[c.dir], true)
			frontier = append(frontier, c.pos)
		}

		// Loop connections: try to connect current to unconnected occupied neighbors
		currentRoom := grid[current.Y][current.X]
		for _, dir := range directions {
			if wallOpen(currentRoom.Walls, dir) {
				continue
			}
			offset := directionOffsets[dir]
			nx, ny := current.X+offset.X, current.Y+offset.Y
			neighbor := GetRoom(grid, types.Pos{X: nx, Y: ny})
			if neighbor != nil && r.Chance(config.LoopProbability) {
				setWall(&currentRoom.Walls, dir, true)
				setWall(&neighbor.Walls, opposite[dir], true)
			}
		}

		// Remove from frontier if no more expansion possible
		hasRemaining := false
		for _, dir := range directions {
			offset := directionOffsets[dir]
			nx, ny := current.X+offset.X, current.Y+offset.Y
			if nx >= 0 && nx < gridSize && ny >= 0 && ny < gridSize && !occupied[posKeyXY(nx, ny)] {
				hasRemaining = true
				break
			}
		}
		if !hasRemaining {
			frontier = append(frontier[:idx], frontier[idx+1:]...)
		}
	}

	// BFS to compute distances from entry
	distances := computeDistances(grid, entryPos)

	maxDist := 0
	for _, d := range distances {
		if d > maxDist {
			maxDist = d
		}
	}

	// Find boss position (farthest room from entry)
	bossPos := entryPos
	bestDist := 0
	for key, dist := range distances {
		if dist > bestDist {
			bestDist = dist
			var x, y int
			fmt.Sscanf(key, "%d,%d", &x, &y)
			bossPos = types.Pos{X: x, Y: y}
		}
	}

	// Filter light-restoring items
	var lightItems []types.Item
	for _, item := range cd.Items {
		if item.UseEffect != nil && item.UseEffect.Type == "restore_light" {
			lightItems = append(lightItems, item)
		}
	}
	lightPlacedInFirstThree := false

	// Assign content to each room
	for key, dist := range distances {
		var x, y int
		fmt.Sscanf(key, "%d,%d", &x, &y)
		room := grid[y][x]
		isEntry := x == entryPos.X && y == entryPos.Y
		isBoss := x == bossPos.X && y == bossPos.Y
		tier := getTierForDistance(dist, maxDist)

		// Pick room template
		if len(cd.Rooms) > 0 {
			tmpl := rng.Pick(r, cd.Rooms)
			room.TemplateID = tmpl.ID
			room.Name = tmpl.Name
			room.Description = content.RenderDescription(tmpl.Features, cd.Fragments, r)

			// Assign encounter
			if isEntry {
				room.Encounter = nil
			} else if isBoss {
				monster := makeMonsterInstance(r, cd.Monsters, "boss")
				if monster != nil {
					room.Encounter = &types.Encounter{Type: "monster", Monster: monster}
				}
			} else {
				room.Encounter = nil
				for _, enc := range tmpl.Encounters {
					if r.Chance(enc.Chance) {
						if enc.Type == "monster" {
							monster := makeMonsterInstance(r, cd.Monsters, tier)
							if monster != nil {
								room.Encounter = &types.Encounter{Type: "monster", Monster: monster}
							}
						} else if enc.Type == "trap" {
							saveStat := rng.Pick(r, []string{"str", "dex", "wil"})
							room.Encounter = &types.Encounter{
								Type:        "trap",
								Damage:      "d4",
								SaveStat:    saveStat,
								Description: "A hidden trap springs!",
							}
						}
						break
					}
				}
			}

			// Assign loot from template
			if tmpl.Loot != nil && len(cd.Items) > 0 && r.Chance(tmpl.Loot.Chance) {
				room.Loot = append(room.Loot, rng.Pick(r, cd.Items))
			}
		}

		// Place light sources
		if len(lightItems) > 0 {
			if !lightPlacedInFirstThree && dist >= 1 && dist <= 3 {
				room.Loot = append(room.Loot, rng.Pick(r, lightItems))
				lightPlacedInFirstThree = true
			} else if dist > 0 && r.Chance(config.LightSourceFrequency) {
				room.Loot = append(room.Loot, rng.Pick(r, lightItems))
			}
		}
	}

	return GeneratorResult{
		Grid:      grid,
		Entry:     entryPos,
		BossPos:   bossPos,
		RoomCount: len(occupied),
	}
}
