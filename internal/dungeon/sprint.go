package dungeon

import "github.com/mph-llm-experiments/tinycrawl-tui/internal/types"

// CreateSprintDungeon creates a sprint dungeon from a slice of rooms.
func CreateSprintDungeon(rooms []types.DungeonRoom) types.SprintDungeon {
	return types.SprintDungeon{
		Rooms:            rooms,
		CurrentRoomIndex: 0,
	}
}

// NavigateSprint moves the player through the sprint dungeon.
func NavigateSprint(d types.SprintDungeon, exitIndex int) types.SprintDungeon {
	if exitIndex < 0 || exitIndex >= len(d.Rooms[d.CurrentRoomIndex].Exits) {
		return d
	}
	exit := d.Rooms[d.CurrentRoomIndex].Exits[exitIndex]

	var nextIndex int
	if exit.Direction == "forward" {
		nextIndex = min(d.CurrentRoomIndex+1, len(d.Rooms)-1)
	} else {
		nextIndex = max(0, d.CurrentRoomIndex-1)
	}
	if nextIndex == d.CurrentRoomIndex {
		return d
	}
	d.CurrentRoomIndex = nextIndex
	return d
}

// MarkSprintCleared marks a room as cleared.
func MarkSprintCleared(d types.SprintDungeon, index int) types.SprintDungeon {
	if index >= 0 && index < len(d.Rooms) {
		d.Rooms[index].Cleared = true
	}
	return d
}

// MarkSprintVisited marks a room as visited.
func MarkSprintVisited(d types.SprintDungeon, index int) types.SprintDungeon {
	if index >= 0 && index < len(d.Rooms) {
		d.Rooms[index].Visited = true
	}
	return d
}
