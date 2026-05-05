package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/engine"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/rng"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// SwapItemMsg tells root to drop an inventory item and take a loot item.
type SwapItemMsg struct {
	DropSlot  int
	TakeIndex int
}

// TakeAndEquipMsg tells root to take a loot item and equip it (move to active slot).
type TakeAndEquipMsg struct {
	TakeIndex int
}

// lootChoice represents one selectable option during looting.
type lootChoice struct {
	label     string
	takeIndex int  // -1 for skip
	swapSlot  int  // -1 for normal take, >=0 for swap (drop this slot first)
}

// RoomView displays room exploration and looting phases.
type RoomView struct {
	state     *types.GameState
	cd        types.ContentData
	kitty     bool
	cursor    int
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

func (v *RoomView) choiceCount() int {
	if v.state.Phase == types.PhaseLooting {
		return len(v.buildLootChoices())
	}
	n := len(v.state.Exits())
	if v.canRest() {
		n++
	}
	return n
}

// dieMax returns the maximum value of a dice notation (e.g., "d8" → 8, "2d6" → 12).
func dieMax(notation string) int {
	count, sides, err := rng.ParseDice(notation)
	if err != nil {
		return 0
	}
	return count * sides
}

// findEquippedWeapon returns the first weapon in inventory (the one used in combat).
func (v *RoomView) findEquippedWeapon() (*types.Item, int) {
	if v.state.Character == nil {
		return nil, -1
	}
	for i, item := range v.state.Character.Inventory {
		if item != nil && item.Type == "weapon" {
			return item, i
		}
	}
	return nil, -1
}

// buildLootChoices creates the list of choices during looting, including equip/swap options.
func (v *RoomView) buildLootChoices() []lootChoice {
	if v.state.Character == nil {
		return nil
	}
	char := v.state.Character
	currentWeapon, currentWeaponSlot := v.findEquippedWeapon()
	var choices []lootChoice

	for i, item := range v.state.PendingLoot {
		canFit := engine.CanAddItem(*char, item.Slots)

		// For weapons: offer equip/swap only if it's BETTER than current
		isBetterWeapon := item.Type == "weapon" && currentWeapon != nil &&
			dieMax(item.Damage) > dieMax(currentWeapon.Damage)

		if isBetterWeapon {
			comparison := styleDim.Render(fmt.Sprintf(" (%s vs your %s)", item.Damage, currentWeapon.Damage))
			if canFit {
				choices = append(choices, lootChoice{
					label: lipgloss.NewStyle().Foreground(colorLoot).Render(
						"Equip "+v.itemDetail(item)) + comparison,
					takeIndex: i,
					swapSlot:  -2,
				})
			} else {
				choices = append(choices, lootChoice{
					label: lipgloss.NewStyle().Foreground(colorLoot).Render(
						"Swap for "+currentWeapon.Name) + comparison,
					takeIndex: i,
					swapSlot:  currentWeaponSlot,
				})
			}
		} else if canFit {
			choices = append(choices, lootChoice{
				label:     lipgloss.NewStyle().Foreground(colorLoot).Render("Take " + v.itemDetail(item)),
				takeIndex: i,
				swapSlot:  -1,
			})
		} else {
			choices = append(choices, lootChoice{
				label:     styleDim.Render(v.itemDetail(item)) + styleDanger.Render(" (full)"),
				takeIndex: -1, // can't take, not actionable
				swapSlot:  -1,
			})
		}
	}

	choices = append(choices, lootChoice{
		label:     styleDim.Render("Skip"),
		takeIndex: -1,
		swapSlot:  -1,
	})
	return choices
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
			case "p", "esc":
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
		count := v.choiceCount()
		switch msg.String() {
		case "up", "left", "k":
			if count > 0 {
				v.cursor = (v.cursor - 1 + count) % count
			}
			return v, nil
		case "down", "right", "j":
			if count > 0 {
				v.cursor = (v.cursor + 1) % count
			}
			return v, nil
		case "enter", " ":
			return v, v.selectCurrent()
		case "s":
			if v.state.Phase == types.PhaseLooting {
				return v, func() tea.Msg { return GameAction{Action: types.SkipLoot()} }
			}
		case "n":
			if v.state.Expedition != nil {
				return v, v.expeditionMove("n")
			}
		case "e":
			if v.state.Expedition != nil {
				return v, v.expeditionMove("e")
			}
		case "w":
			if v.state.Expedition != nil {
				return v, v.expeditionMove("w")
			}
		// "s" handled above for skip loot; expedition south via arrow keys or map
		case "m":
			if v.state.Expedition != nil {
				v.mapView = NewMapView(v.state.Expedition, v.state.Light)
			}
			return v, nil
		case "p":
			if v.state.Character != nil {
				v.inventory = NewInventoryView(v.state.Character)
			}
			return v, nil
		case "r":
			if v.canRest() {
				return v, func() tea.Msg { return GameAction{Action: types.Rest()} }
			}
			return v, nil
		case "q":
			return v, tea.Quit
		}
	}
	return v, nil
}

func (v *RoomView) expeditionMove(dir string) tea.Cmd {
	for i, exit := range v.state.Exits() {
		if exit.Direction == dir {
			idx := i
			return func() tea.Msg { return GameAction{Action: types.ChooseExit(idx)} }
		}
	}
	return nil
}

func (v *RoomView) selectCurrent() tea.Cmd {
	if v.state.Phase == types.PhaseLooting {
		choices := v.buildLootChoices()
		if v.cursor >= len(choices) {
			return nil
		}
		choice := choices[v.cursor]
		if choice.takeIndex < 0 {
			return func() tea.Msg { return GameAction{Action: types.SkipLoot()} }
		}
		if choice.swapSlot == -2 {
			// Take + equip: take the item, then equip it to active slot
			idx := choice.takeIndex
			return func() tea.Msg { return TakeAndEquipMsg{TakeIndex: idx} }
		}
		if choice.swapSlot >= 0 {
			// Swap: drop old, then take new
			slot := choice.swapSlot
			idx := choice.takeIndex
			return func() tea.Msg { return SwapItemMsg{DropSlot: slot, TakeIndex: idx} }
		}
		// Normal take
		idx := choice.takeIndex
		return func() tea.Msg { return GameAction{Action: types.TakeItem(idx)} }
	}

	exits := v.state.Exits()
	if v.cursor < len(exits) {
		idx := v.cursor
		return func() tea.Msg { return GameAction{Action: types.ChooseExit(idx)} }
	}
	if v.canRest() {
		return func() tea.Msg { return GameAction{Action: types.Rest()} }
	}
	return nil
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
	band := types.GetLightBand(v.state.Light)

	// Room header (centered)
	depth := v.state.Depth()
	total := v.state.TotalRooms()
	center := lipgloss.NewStyle().Width(contentWidth - 4).Align(lipgloss.Center)
	sections = append(sections, center.Render(
		styleDim.Render(fmt.Sprintf("%d/%d", depth, total))+" "+styleTitle.Render(room.Name)))

	// Equipped weapon
	if v.state.Character != nil {
		for _, item := range v.state.Character.Inventory {
			if item != nil && item.Type == "weapon" {
				sections = append(sections, center.Render(
					styleDim.Render(fmt.Sprintf("wielding %s (%s)", item.Name, item.Damage))))
				break
			}
		}
	}

	// Room description (filtered by light)
	desc := engine.GetVisibleDescription(room.Description, band)
	if desc != "" {
		sections = append(sections, "")
		sections = append(sections, styleStat.Render(desc))
	}

	// Game log entries — show prominently (these include combat results)
	if len(v.state.Log) > 0 {
		sections = append(sections, "")
		sections = append(sections, v.renderLog())
	}

	sections = append(sections, divider())

	// Looting UI or Exits
	if v.state.Phase == types.PhaseLooting && len(v.state.PendingLoot) > 0 {
		sections = append(sections, v.renderLoot())
	} else {
		exits := v.state.Exits()
		if len(exits) > 0 {
			sections = append(sections, v.renderExits(exits))
		}
	}

	return styleBox.Render(strings.Join(sections, "\n"))
}

func (v *RoomView) renderLog() string {
	var lines []string
	for _, entry := range v.state.Log {
		lines = append(lines, v.styleLogEntry(entry))
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
	for i, exit := range exits {
		label := exit.Label
		if exit.Direction != "" && exit.Direction != exit.Label {
			label += styleDim.Render(" — "+exit.Direction)
		}
		lines = append(lines, v.choiceLine(i, styleStat.Render(label)))
	}
	if v.canRest() {
		lines = append(lines, v.choiceLine(len(exits), styleLabel.Render("Rest")))
	}
	return strings.Join(lines, "\n")
}

func (v *RoomView) renderLoot() string {
	choices := v.buildLootChoices()
	var lines []string
	for i, c := range choices {
		lines = append(lines, v.choiceLine(i, c.label))
	}
	return strings.Join(lines, "\n")
}

func (v *RoomView) choiceLine(index int, label string) string {
	if index == v.cursor {
		return styleTitle.Render("▸ ") + lipgloss.NewStyle().Bold(true).Render(label)
	}
	return styleDim.Render("  ") + label
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

	hints := []KeyHint{
		{Key: "↑↓", Desc: "choose"},
		{Key: "enter", Desc: "select"},
		{Key: "p", Desc: "pack"},
	}
	if v.canRest() {
		hints = append(hints, KeyHint{Key: "r", Desc: "rest"})
	}
	if v.state.Expedition != nil {
		hints = append(hints, KeyHint{Key: "m", Desc: "map"})
	}
	hints = append(hints, KeyHint{Key: "q", Desc: "quit"})
	return hints
}
