package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// InventoryView is an overlay for managing the player's inventory.
type InventoryView struct {
	char     *types.Character
	cursor   int
	items    []inventoryEntry
	action   *types.Action // set when use/drop is triggered
}

type inventoryEntry struct {
	name      string
	itemType  string
	slots     int
	damage    string
	armor     int
	hasUse    bool
	slotIndex int // first slot index of this item in character inventory
}

// NewInventoryView creates an inventory overlay.
func NewInventoryView(char *types.Character) *InventoryView {
	v := &InventoryView{char: char}
	v.items = v.buildEntries()
	return v
}

func (v *InventoryView) buildEntries() []inventoryEntry {
	seen := make(map[string]bool)
	var entries []inventoryEntry
	for i, item := range v.char.Inventory {
		if item == nil {
			continue
		}
		if seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		entries = append(entries, inventoryEntry{
			name:      item.Name,
			itemType:  item.Type,
			slots:     item.Slots,
			damage:    item.Damage,
			armor:     item.ArmorValue,
			hasUse:    item.UseEffect != nil,
			slotIndex: i,
		})
	}
	return entries
}

func (v *InventoryView) Init() tea.Cmd { return nil }

func (v *InventoryView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if v.cursor > 0 {
				v.cursor--
			}
		case "down", "j":
			if v.cursor < len(v.items)-1 {
				v.cursor++
			}
		case "u":
			if len(v.items) > 0 && v.items[v.cursor].hasUse {
				action := types.UseItem(v.items[v.cursor].slotIndex)
				v.action = &action
			}
		case "d":
			if len(v.items) > 0 {
				action := types.DropItem(v.items[v.cursor].slotIndex)
				v.action = &action
			}
		}
	}
	return v, nil
}

func (v *InventoryView) View() string {
	var lines []string

	title := styleTitle.Render("INVENTORY")
	slotsUsed := 0
	for _, item := range v.char.Inventory {
		if item != nil {
			slotsUsed++
		}
	}
	slotInfo := styleDim.Render(fmt.Sprintf("(%d/%d slots)", slotsUsed, types.MaxSlots))
	lines = append(lines, title+"  "+slotInfo)
	lines = append(lines, "")

	if len(v.items) == 0 {
		lines = append(lines, styleDim.Render("  Empty."))
	} else {
		for i, entry := range v.items {
			line := v.renderEntry(entry, i == v.cursor)
			lines = append(lines, line)
		}
	}

	lines = append(lines, "")

	// Actions for selected item
	if len(v.items) > 0 {
		selected := v.items[v.cursor]
		var actions []string
		if selected.hasUse {
			actions = append(actions, styleKey.Render("u")+" "+styleDim.Render("use"))
		}
		actions = append(actions, styleKey.Render("d")+" "+styleDim.Render("drop"))
		actions = append(actions, styleKey.Render("esc")+" "+styleDim.Render("close"))
		lines = append(lines, strings.Join(actions, "  "))
	}

	content := strings.Join(lines, "\n")
	return styleBox.Render(content)
}

func (v *InventoryView) renderEntry(entry inventoryEntry, selected bool) string {
	// Build item description
	var parts []string

	slotStr := ""
	if entry.slots > 1 {
		slotStr = fmt.Sprintf("[%d slots] ", entry.slots)
	}

	detail := ""
	switch {
	case entry.itemType == "weapon" && entry.damage != "":
		detail = fmt.Sprintf(" (%s)", entry.damage)
	case entry.armor > 0:
		detail = fmt.Sprintf(" (Armor %d)", entry.armor)
	}

	typeLabel := ""
	if entry.itemType == "weapon" || entry.itemType == "armor" || entry.itemType == "shield" {
		typeLabel = " " + styleDim.Render("["+entry.itemType+"]")
	}

	useMark := ""
	if entry.hasUse {
		useMark = lipgloss.NewStyle().Foreground(colorLoot).Render(" *")
	}

	parts = append(parts, slotStr+entry.name+detail+typeLabel+useMark)

	line := strings.Join(parts, "")

	// Cursor indicator
	prefix := "  "
	if selected {
		prefix = "> "
		line = lipgloss.NewStyle().Bold(true).Render(line)
	}

	return prefix + line
}

func (v *InventoryView) KeyHints() []KeyHint {
	hints := []KeyHint{
		{Key: "up/dn", Desc: "navigate"},
	}
	if len(v.items) > 0 {
		if v.items[v.cursor].hasUse {
			hints = append(hints, KeyHint{Key: "u", Desc: "use"})
		}
		hints = append(hints, KeyHint{Key: "d", Desc: "drop"})
	}
	hints = append(hints, KeyHint{Key: "esc", Desc: "close"})
	return hints
}
