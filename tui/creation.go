package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// CreationView displays the character creation screen.
type CreationView struct {
	char *types.Character
}

// NewCreationView creates a character creation view.
func NewCreationView(char *types.Character) *CreationView {
	return &CreationView{char: char}
}

func (v *CreationView) Init() tea.Cmd { return nil }

func (v *CreationView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "a":
			return v, func() tea.Msg {
				return GameAction{Action: types.AcceptCharacter()}
			}
		case "r":
			return v, func() tea.Msg {
				return GameAction{Action: types.RerollCharacter()}
			}
		case "esc":
			return v, func() tea.Msg {
				return GameAction{Action: types.ReturnToTitle()}
			}
		}
	}
	return v, nil
}

func (v *CreationView) View() string {
	if v.char == nil {
		return styleDim.Render("No character generated.")
	}

	// Character name
	name := styleTitle.Render(v.char.Name)

	// Stats grid
	stats := fmt.Sprintf(
		"%s %s    %s %s    %s %s    %s %s",
		styleLabel.Render("STR"),
		styleStat.Render(fmt.Sprintf("%d", v.char.Str)),
		styleLabel.Render("DEX"),
		styleStat.Render(fmt.Sprintf("%d", v.char.Dex)),
		styleLabel.Render("WIL"),
		styleStat.Render(fmt.Sprintf("%d", v.char.Wil)),
		styleLabel.Render("HP"),
		styleStat.Render(fmt.Sprintf("%d", v.char.HP)),
	)

	// Armor
	armor := styleLabel.Render("Armor") + " " + styleStat.Render(fmt.Sprintf("%d", v.char.Armor))

	// Equipment list (deduplicated with slot counts)
	equipment := v.renderEquipment()

	content := name + "\n\n" +
		stats + "\n" +
		armor + "\n\n" +
		styleLabel.Render("Equipment") + "\n" +
		equipment

	return styleBox.Render(content)
}

func (v *CreationView) renderEquipment() string {
	type itemEntry struct {
		name  string
		slots int
	}

	// Deduplicate items by ID, counting slot usage
	seen := make(map[string]bool)
	var items []itemEntry
	for _, item := range v.char.Inventory {
		if item == nil {
			continue
		}
		if seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		items = append(items, itemEntry{name: item.Name, slots: item.Slots})
	}

	var lines []string
	for _, item := range items {
		slotInfo := ""
		if item.slots > 1 {
			slotInfo = styleDim.Render(fmt.Sprintf(" (%d slots)", item.slots))
		}
		lines = append(lines, "  "+styleStat.Render(item.name)+slotInfo)
	}

	if len(lines) == 0 {
		return "  " + styleDim.Render("Nothing.")
	}
	return strings.Join(lines, "\n")
}

func (v *CreationView) KeyHints() []KeyHint {
	return []KeyHint{
		{Key: "a", Desc: "accept"},
		{Key: "r", Desc: "reroll"},
		{Key: "esc", Desc: "back"},
	}
}
