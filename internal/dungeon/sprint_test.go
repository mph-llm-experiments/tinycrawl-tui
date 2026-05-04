package dungeon

import (
	"fmt"
	"testing"

	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

func makeTestRooms(n int) []types.DungeonRoom {
	rooms := make([]types.DungeonRoom, n)
	for i := range rooms {
		rooms[i] = types.DungeonRoom{
			TemplateID: fmt.Sprintf("room_%d", i),
			Name:       fmt.Sprintf("Room %d", i),
			Exits: []types.ExitDef{
				{Label: "Forward", Direction: "forward"},
				{Label: "Back", Direction: "back"},
			},
		}
	}
	return rooms
}

func TestCreateSprintDungeon(t *testing.T) {
	rooms := makeTestRooms(5)
	d := CreateSprintDungeon(rooms)
	if d.CurrentRoomIndex != 0 {
		t.Fatalf("expected index 0, got %d", d.CurrentRoomIndex)
	}
	if len(d.Rooms) != 5 {
		t.Fatalf("expected 5 rooms, got %d", len(d.Rooms))
	}
}

func TestNavigateForward(t *testing.T) {
	rooms := makeTestRooms(5)
	d := CreateSprintDungeon(rooms)
	d = NavigateSprint(d, 0) // forward exit
	if d.CurrentRoomIndex != 1 {
		t.Fatalf("expected index 1 after forward, got %d", d.CurrentRoomIndex)
	}
}

func TestNavigateBack(t *testing.T) {
	rooms := makeTestRooms(5)
	d := CreateSprintDungeon(rooms)
	d = NavigateSprint(d, 0) // go to room 1
	d = NavigateSprint(d, 1) // back exit
	if d.CurrentRoomIndex != 0 {
		t.Fatalf("expected index 0 after back, got %d", d.CurrentRoomIndex)
	}
}

func TestNavigateBounds(t *testing.T) {
	rooms := makeTestRooms(3)
	d := CreateSprintDungeon(rooms)
	// Try going back from room 0
	d2 := NavigateSprint(d, 1)
	if d2.CurrentRoomIndex != 0 {
		t.Fatal("should not go below 0")
	}
	// Go to last room and try forward
	d = NavigateSprint(d, 0)
	d = NavigateSprint(d, 0)
	d3 := NavigateSprint(d, 0)
	if d3.CurrentRoomIndex != 2 {
		t.Fatal("should not exceed last room")
	}
}

func TestMarkCleared(t *testing.T) {
	rooms := makeTestRooms(3)
	d := CreateSprintDungeon(rooms)
	d = MarkSprintCleared(d, 0)
	if !d.Rooms[0].Cleared {
		t.Fatal("room 0 should be cleared")
	}
	if d.Rooms[1].Cleared {
		t.Fatal("room 1 should not be cleared")
	}
}

func TestMarkVisited(t *testing.T) {
	rooms := makeTestRooms(3)
	d := CreateSprintDungeon(rooms)
	d = MarkSprintVisited(d, 0)
	if !d.Rooms[0].Visited {
		t.Fatal("room 0 should be visited")
	}
	if d.Rooms[1].Visited {
		t.Fatal("room 1 should not be visited")
	}
}
