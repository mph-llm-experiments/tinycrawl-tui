package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/engine"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// RoomView displays room exploration and looting phases.
type RoomView struct {
	state     *types.GameState
	cd        types.ContentData
	kitty     bool
	inventory *InventoryView
	mapView   *MapView
}

// NewRoomView creates a room exploration view.
func NewRoomView(state *types.GameState, cd types.ContentData, kitty bool) *RoomView {
	return &RoomView{
		state: state,
		cd:    cd,
		kitty: kitty,
	}
}

func (v *RoomView) Init() tea.Cmd { return nil }

func (v *RoomView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If map is open, delegate to it
	if v.mapView != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "m", "esc":
				v.mapView = nil
				return v, nil
			default:
				dir := directionForMsg(msg.String())
				if dir != "" && v.state.Expedition != nil {
					v.mapView = nil
					for i, exit := range v.state.Exits() {
						if exit.Direction == dir {
							idx := i
							return v, func() tea.Msg {
								return GameAction{Action: types.ChooseExit(idx)}
							}
						}
					}
					return v, nil
				}
			}
		}
		updated, cmd := v.mapView.Update(msg)
		v.mapView = updated.(*MapView)
		return v, cmd
	}

	// If inventory is open, delegate to it
	if v.inventory != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "i", "p", "esc":
				v.inventory = nil
				return v, nil
			}
		}
		updated, cmd := v.inventory.Update(msg)
		v.inventory = updated.(*InventoryView)
		if v.inventory.action != nil {
			action := *v.inventory.action
			v.inventory = nil
			return v, func() tea.Msg {
				return GameAction{Action: action}
			}
		}
		return v, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "m":
			if v.state.Expedition != nil {
				v.mapView = NewMapView(v.state.Expedition, v.state.Light)
			}
			return v, nil
		case "i", "p":
			if v.state.Character != nil {
				v.inventory = NewInventoryView(v.state.Character)
			}
			return v, nil
		case "q":
			return v, tea.Quit
		case "z":
			if v.canRest() {
				return v, func() tea.Msg {
					return GameAction{Action: types.Rest()}
				}
			}
		case "s":
			if v.state.Phase == types.PhaseLooting {
				return v, func() tea.Msg {
					return GameAction{Action: types.SkipLoot()}
				}
			}
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0]-'0') - 1
			if v.state.Phase == types.PhaseLooting {
				if idx < len(v.state.PendingLoot) {
					return v, func() tea.Msg {
						return GameAction{Action: types.TakeItem(idx)}
					}
				}
			} else {
				exits := v.state.Exits()
				if idx < len(exits) {
					return v, func() tea.Msg {
						return GameAction{Action: types.ChooseExit(idx)}
					}
				}
			}
		}
	}
	return v, nil
}

func (v *RoomView) View() string {
	if v.mapView != nil {
		return v.mapView.View()
	}
	if v.inventory != nil {
		return v.inventory.View()
	}

	room := v.state.CurrentRoom()
	if room == nil {
		return styleDim.Render("No room.")
	}

	var sections []string

	// Room header: depth and name
	header := v.renderHeader(room)
	sections = append(sections, header)

	// Light band
	band := types.GetLightBand(v.state.Light)
	bandLabel := styleDim.Render("(" + string(band) + ")")
	sections = append(sections, bandLabel)

	// Room description (filtered by light)
	desc := engine.GetVisibleDescription(room.Description, band)
	if desc != "" {
		sections = append(sections, styleStat.Render(desc))
	}

	// Game log entries
	if len(v.state.Log) > 0 {
		logSection := v.renderLog()
		sections = append(sections, logSection)
	}

	// Looting UI or Exits
	if v.state.Phase == types.PhaseLooting && len(v.state.PendingLoot) > 0 {
		lootSection := v.renderLoot()
		sections = append(sections, lootSection)
	} else {
		exits := v.state.Exits()
		if len(exits) > 0 {
			exitSection := v.renderExits(exits)
			sections = append(sections, exitSection)
		}
	}

	content := strings.Join(sections, "\n\n")
	return styleBox.Render(content)
}

func (v *RoomView) renderHeader(room *types.DungeonRoom) string {
	depth := v.state.Depth()
	total := v.state.TotalRooms()
	roomNum := fmt.Sprintf("Room %d of %d", depth, total)
	return styleDim.Render(roomNum) + " — " + styleTitle.Render(room.Name)
}

func (v *RoomView) renderLog() string {
	var lines []string
	for _, entry := range v.state.Log {
		styled := v.styleLogEntry(entry)
		lines = append(lines, styled)
	}
	return strings.Join(lines, "\n")
}

func (v *RoomView) styleLogEntry(entry types.LogEntry) string {
	switch entry.Type {
	case "danger":
		return styleDanger.Render(entry.Text)
	case "loot":
		return lipgloss.NewStyle().Foreground(colorLoot).Render(entry.Text)
	case "combat":
		return styleStat.Render(entry.Text)
	case "system":
		return styleDim.Render(entry.Text)
	default:
		return styleStat.Render(entry.Text)
	}
}

func (v *RoomView) renderExits(exits []types.ExitDef) string {
	var lines []string
	lines = append(lines, styleLabel.Render("Exits"))
	for i, exit := range exits {
		line := fmt.Sprintf("  %d. %s", i+1, exit.Label)
		if exit.Direction != "" && exit.Direction != exit.Label {
			line += styleDim.Render(" — "+exit.Direction)
		}
		lines = append(lines, styleStat.Render(line))
	}
	return strings.Join(lines, "\n")
}

func (v *RoomView) renderLoot() string {
	var lines []string
	lines = append(lines, styleLabel.Render("Loot"))
	for i, item := range v.state.PendingLoot {
		detail := v.itemDetail(item)
		line := fmt.Sprintf("  %d. %s", i+1, detail)
		lines = append(lines, lipgloss.NewStyle().Foreground(colorLoot).Render(line))
	}
	return strings.Join(lines, "\n")
}

func (v *RoomView) itemDetail(item types.Item) string {
	parts := []string{item.Name}
	if item.Type == "weapon" && item.Damage != "" {
		parts = append(parts, fmt.Sprintf("(%s)", item.Damage))
	} else if item.ArmorValue > 0 {
		parts = append(parts, fmt.Sprintf("(Armor %d)", item.ArmorValue))
	}
	if item.Slots > 1 {
		parts = append(parts, fmt.Sprintf("[%d slots]", item.Slots))
	}
	return strings.Join(parts, " ")
}

func (v *RoomView) canRest() bool {
	if v.state.Phase != types.PhaseExploring {
		return false
	}
	room := v.state.CurrentRoom()
	if room == nil || !room.Cleared {
		return false
	}
	if v.state.Character == nil {
		return false
	}
	return v.state.Character.HP < v.state.Character.MaxHP
}

func (v *RoomView) KeyHints() []KeyHint {
	if v.mapView != nil {
		return v.mapView.KeyHints()
	}
	if v.inventory != nil {
		return v.inventory.KeyHints()
	}

	var hints []KeyHint
	if v.state.Phase == types.PhaseLooting {
		hints = append(hints, KeyHint{Key: "1-9", Desc: "take"})
		hints = append(hints, KeyHint{Key: "s", Desc: "skip"})
	} else {
		exits := v.state.Exits()
		if len(exits) > 0 {
			if len(exits) == 1 {
				hints = append(hints, KeyHint{Key: "1", Desc: "go"})
			} else {
				hints = append(hints, KeyHint{Key: fmt.Sprintf("1-%d", len(exits)), Desc: "exit"})
			}
		}
	}
	hints = append(hints, KeyHint{Key: "i", Desc: "inventory"})
	if v.state.Expedition != nil {
		hints = append(hints, KeyHint{Key: "m", Desc: "map"})
	}
	if v.canRest() {
		hints = append(hints, KeyHint{Key: "z", Desc: "rest"})
	}
	hints = append(hints, KeyHint{Key: "q", Desc: "quit"})
	return hints
}
