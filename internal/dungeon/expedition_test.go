package dungeon

import (
	"strings"
	"testing"

	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// makeTestGrid creates a 3x3 grid for testing.
// Layout (y=0 is top row):
//
//	  x=0    x=1    x=2
//	y=0: nil   r01    nil
//	y=1: r10   r11    r12
//	y=2: nil   r21    nil
//
// Connections (open walls):
//
//	r01 <-> r11 (r01.S open, r11.N open)
//	r11 <-> r10 (r11.W open, r10.E open)
//	r11 <-> r12 (r11.E open, r12.W open)
//	r11 <-> r21 (r11.S open, r21.N open)
func makeTestGrid() [][]*types.ExpeditionRoom {
	grid := make([][]*types.ExpeditionRoom, 3)
	for i := range grid {
		grid[i] = make([]*types.ExpeditionRoom, 3)
	}

	newRoom := func(x, y int) *types.ExpeditionRoom {
		return &types.ExpeditionRoom{
			DungeonRoom: types.DungeonRoom{
				TemplateID: "test",
				Name:       "Test Room",
			},
			Pos:   types.Pos{X: x, Y: y},
			Walls: types.Walls{},
		}
	}

	r01 := newRoom(1, 0)
	r10 := newRoom(0, 1)
	r11 := newRoom(1, 1)
	r12 := newRoom(2, 1)
	r21 := newRoom(1, 2)

	// r01 <-> r11
	r01.Walls.S = true
	r11.Walls.N = true

	// r11 <-> r10
	r11.Walls.W = true
	r10.Walls.E = true

	// r11 <-> r12
	r11.Walls.E = true
	r12.Walls.W = true

	// r11 <-> r21
	r11.Walls.S = true
	r21.Walls.N = true

	grid[0][1] = r01
	grid[1][0] = r10
	grid[1][1] = r11
	grid[1][2] = r12
	grid[2][1] = r21

	return grid
}

func TestPosKey(t *testing.T) {
	pos := types.Pos{X: 3, Y: 7}
	key := PosKey(pos)
	if key != "3,7" {
		t.Fatalf("expected '3,7', got '%s'", key)
	}
	zeroKey := PosKey(types.Pos{X: 0, Y: 0})
	if zeroKey != "0,0" {
		t.Fatalf("expected '0,0', got '%s'", zeroKey)
	}
}

func TestGetRoom(t *testing.T) {
	grid := makeTestGrid()

	// Valid room
	r := GetRoom(grid, types.Pos{X: 1, Y: 1})
	if r == nil {
		t.Fatal("expected room at (1,1), got nil")
	}

	// Out of bounds
	r = GetRoom(grid, types.Pos{X: -1, Y: 0})
	if r != nil {
		t.Fatal("expected nil for negative x")
	}
	r = GetRoom(grid, types.Pos{X: 0, Y: 10})
	if r != nil {
		t.Fatal("expected nil for y out of bounds")
	}

	// Nil cell (grid position exists but is nil)
	r = GetRoom(grid, types.Pos{X: 0, Y: 0})
	if r != nil {
		t.Fatal("expected nil for empty grid cell (0,0)")
	}
}

func TestComputePeeked(t *testing.T) {
	grid := makeTestGrid()
	// From r11 at (1,1), open walls go N, W, E, S
	// Neighbors: r01(1,0), r10(0,1), r12(2,1), r21(1,2)
	// Exclude visited = ["1,1"]
	exclude := []string{"1,1"}
	peeked := ComputePeeked(grid, types.Pos{X: 1, Y: 1}, exclude)
	if len(peeked) != 4 {
		t.Fatalf("expected 4 peeked rooms from center, got %d: %v", len(peeked), peeked)
	}

	// Exclude all neighbors: peeked should be empty
	allExclude := []string{"1,1", "1,0", "0,1", "2,1", "1,2"}
	peeked2 := ComputePeeked(grid, types.Pos{X: 1, Y: 1}, allExclude)
	if len(peeked2) != 0 {
		t.Fatalf("expected 0 peeked when all excluded, got %d", len(peeked2))
	}

	// From r01 at (1,0): only S wall open, neighbor is r11(1,1)
	peeked3 := ComputePeeked(grid, types.Pos{X: 1, Y: 0}, []string{"1,0"})
	if len(peeked3) != 1 || peeked3[0] != "1,1" {
		t.Fatalf("expected ['1,1'] from r01, got %v", peeked3)
	}
}

func TestBuildExits(t *testing.T) {
	grid := makeTestGrid()
	r11 := GetRoom(grid, types.Pos{X: 1, Y: 1})

	entry := types.Pos{X: 1, Y: 1}
	exits := BuildExits(grid, r11, &entry, nil)

	// r11 has 4 open walls, each leading to a real room
	if len(exits) != 4 {
		t.Fatalf("expected 4 exits from center room, got %d", len(exits))
	}

	// Check that one of the exits has the retreat marker when previousPos is set
	prev := types.Pos{X: 1, Y: 0}
	exitsWithRetreat := BuildExits(grid, r11, nil, &prev)
	retreatFound := false
	for _, ex := range exitsWithRetreat {
		if ex.Direction == "n" {
			if !strings.HasPrefix(ex.Label, "← ") {
				t.Errorf("expected retreat label for north exit, got '%s'", ex.Label)
			}
			retreatFound = true
		}
	}
	if !retreatFound {
		t.Fatal("did not find north exit (retreat direction)")
	}
}

func TestNavigateExpedition(t *testing.T) {
	grid := makeTestGrid()
	entry := types.Pos{X: 1, Y: 1}
	boss := types.Pos{X: 1, Y: 2}
	d := CreateExpeditionDungeon(grid, entry, boss, 5)

	// Initial state
	if d.PlayerPos.X != 1 || d.PlayerPos.Y != 1 {
		t.Fatalf("expected start at (1,1), got %v", d.PlayerPos)
	}
	if d.PreviousPos != nil {
		t.Fatal("expected nil previousPos initially")
	}
	if len(d.Visited) != 1 {
		t.Fatalf("expected 1 visited initially, got %d", len(d.Visited))
	}

	// Move north (r11 -> r01)
	d2 := NavigateExpedition(d, "n")
	if d2.PlayerPos.X != 1 || d2.PlayerPos.Y != 0 {
		t.Fatalf("expected (1,0) after north, got %v", d2.PlayerPos)
	}
	if d2.PreviousPos == nil || d2.PreviousPos.X != 1 || d2.PreviousPos.Y != 1 {
		t.Fatalf("expected previousPos (1,1), got %v", d2.PreviousPos)
	}
	if len(d2.Visited) != 2 {
		t.Fatalf("expected 2 visited after move, got %d", len(d2.Visited))
	}

	// Blocked by wall: r01 has no W wall open
	d3 := NavigateExpedition(d2, "w")
	if d3.PlayerPos != d2.PlayerPos {
		t.Fatalf("expected no movement through closed wall, got %v", d3.PlayerPos)
	}

	// Move back south (revisit r11) — visited count should not increase
	d4 := NavigateExpedition(d2, "s")
	if d4.PlayerPos.X != 1 || d4.PlayerPos.Y != 1 {
		t.Fatalf("expected back at (1,1), got %v", d4.PlayerPos)
	}
	if len(d4.Visited) != 2 {
		t.Fatalf("expected visited unchanged at 2, got %d", len(d4.Visited))
	}
}

func TestMarkExpeditionCleared(t *testing.T) {
	grid := makeTestGrid()
	entry := types.Pos{X: 1, Y: 1}
	boss := types.Pos{X: 1, Y: 2}
	d := CreateExpeditionDungeon(grid, entry, boss, 5)

	room := GetRoom(d.Grid, d.PlayerPos)
	if room.Cleared {
		t.Fatal("room should not be cleared initially")
	}
	d = MarkExpeditionCleared(d)
	if !room.Cleared {
		t.Fatal("room should be cleared after mark")
	}
}

func TestMarkExpeditionVisited(t *testing.T) {
	grid := makeTestGrid()
	entry := types.Pos{X: 1, Y: 1}
	boss := types.Pos{X: 1, Y: 2}
	d := CreateExpeditionDungeon(grid, entry, boss, 5)

	room := GetRoom(d.Grid, d.PlayerPos)
	if room.Visited {
		t.Fatal("room should not be visited initially")
	}
	d = MarkExpeditionVisited(d)
	if !room.Visited {
		t.Fatal("room should be visited after mark")
	}
}
