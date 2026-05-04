package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/dungeon"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// MapView is a box-drawing map overlay for expedition dungeons.
type MapView struct {
	exp   *types.ExpeditionDungeon
	light int
}

// NewMapView creates a map overlay for the given expedition dungeon.
func NewMapView(exp *types.ExpeditionDungeon, light int) *MapView {
	return &MapView{exp: exp, light: light}
}

func (v *MapView) Init() tea.Cmd { return nil }

func (v *MapView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Navigation and close handled by room.go; map.go just needs to satisfy tea.Model.
	return v, nil
}

// directionForMsg returns the expedition direction string for a key message, or "".
func directionForMsg(key string) string {
	switch key {
	case "n", "up":
		return "n"
	case "e", "right":
		return "e"
	case "s", "down":
		return "s"
	case "w", "left":
		return "w"
	}
	return ""
}

func (v *MapView) View() string {
	var lines []string

	title := styleTitle.Render("MAP")
	lines = append(lines, title)
	lines = append(lines, "")

	grid := renderExpeditionGrid(v.exp, v.light)
	lines = append(lines, grid)
	lines = append(lines, "")

	legend := renderMapLegend()
	lines = append(lines, legend)

	content := strings.Join(lines, "\n")
	return styleBox.Render(content)
}

func (v *MapView) KeyHints() []KeyHint {
	return []KeyHint{
		{Key: "↑↓←→", Desc: "navigate"},
		{Key: "m/esc", Desc: "close map"},
	}
}

// containsKey checks if a string slice contains a value.
func containsKey(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// renderExpeditionGrid renders the 2D expedition map as a string.
func renderExpeditionGrid(exp *types.ExpeditionDungeon, light int) string {
	band := types.GetLightBand(light)
	grid := exp.Grid
	size := len(grid)
	if size == 0 {
		return styleDim.Render("(empty)")
	}

	// Determine which rooms are visible
	visible := map[string]bool{}
	for _, key := range exp.Visited {
		visible[key] = true
	}
	for _, key := range exp.Peeked {
		visible[key] = true
	}

	// Apply light filter
	switch band {
	case types.LightBlack:
		// Only current room
		visible = map[string]bool{dungeon.PosKey(exp.PlayerPos): true}
	case types.LightDark:
		// Current room + peeked (distance 1)
		visible = map[string]bool{dungeon.PosKey(exp.PlayerPos): true}
		for _, key := range exp.Peeked {
			visible[key] = true
		}
	}
	// LightBright / LightDim: all explored + peeked (already set above)

	styleMap := lipgloss.NewStyle().Foreground(colorStat)
	stylePeeked := lipgloss.NewStyle().Foreground(colorDim)
	stylePlayer := lipgloss.NewStyle().Bold(true).Foreground(colorName)
	styleEntry := lipgloss.NewStyle().Foreground(colorLabel)
	styleLoot := lipgloss.NewStyle().Foreground(colorLoot)

	var sb strings.Builder

	for y := 0; y < size; y++ {
		// Room row
		for x := 0; x < size; x++ {
			if x > 0 {
				// Horizontal connector between (x-1,y) and (x,y)
				leftKey := fmt.Sprintf("%d,%d", x-1, y)
				rightKey := fmt.Sprintf("%d,%d", x, y)
				leftRoom := dungeon.GetRoom(grid, types.Pos{X: x - 1, Y: y})
				rightRoom := dungeon.GetRoom(grid, types.Pos{X: x, Y: y})
				connected := visible[leftKey] && visible[rightKey] &&
					leftRoom != nil && rightRoom != nil &&
					leftRoom.Walls.E && rightRoom.Walls.W
				if connected {
					sb.WriteString(styleMap.Render("─"))
				} else {
					sb.WriteString(" ")
				}
			}

			key := fmt.Sprintf("%d,%d", x, y)
			if !visible[key] {
				sb.WriteString("   ")
				continue
			}
			room := dungeon.GetRoom(grid, types.Pos{X: x, Y: y})
			if room == nil {
				sb.WriteString("   ")
				continue
			}

			isPlayer := x == exp.PlayerPos.X && y == exp.PlayerPos.Y
			isEntry := x == exp.Entry.X && y == exp.Entry.Y
			isVisited := containsKey(exp.Visited, key)
			isPeeked := containsKey(exp.Peeked, key)

			var cell string
			switch {
			case isPlayer:
				cell = stylePlayer.Render("[▲]")
			case isEntry:
				cell = styleEntry.Render("[⇅]")
			case room.Cleared:
				cell = styleMap.Render("[☆]")
			case len(room.Loot) > 0 && isVisited:
				cell = styleLoot.Render("[♦]")
			case isVisited:
				cell = styleMap.Render("[·]")
			case isPeeked:
				cell = stylePeeked.Render("[░]")
			default:
				cell = "   "
			}
			sb.WriteString(cell)
		}
		sb.WriteString("\n")

		// Vertical connector row between y and y+1
		if y < size-1 {
			for x := 0; x < size; x++ {
				if x > 0 {
					// Spacer for horizontal gap column
					sb.WriteString(" ")
				}
				topKey := fmt.Sprintf("%d,%d", x, y)
				botKey := fmt.Sprintf("%d,%d", x, y+1)
				topRoom := dungeon.GetRoom(grid, types.Pos{X: x, Y: y})
				botRoom := dungeon.GetRoom(grid, types.Pos{X: x, Y: y + 1})
				connected := visible[topKey] && visible[botKey] &&
					topRoom != nil && botRoom != nil &&
					topRoom.Walls.S && botRoom.Walls.N
				if connected {
					sb.WriteString(styleMap.Render(" │ "))
				} else {
					sb.WriteString("   ")
				}
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// renderMapLegend renders the map symbol legend.
func renderMapLegend() string {
	dim := styleDim
	entries := []string{
		styleDim.Render("[▲]") + " you",
		styleDim.Render("[⇅]") + " entry",
		styleDim.Render("[☆]") + " cleared",
		styleDim.Render("[♦]") + " loot",
		styleDim.Render("[·]") + " visited",
		styleDim.Render("[░]") + " glimpsed",
	}
	return dim.Render("Legend: ") + strings.Join(entries, "  ")
}
