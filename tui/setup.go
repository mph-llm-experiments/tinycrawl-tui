package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// PackSelectedMsg is sent when the player chooses a content pack.
type PackSelectedMsg struct {
	Pack types.ContentPack
	Seed int
}

type setupStep int

const (
	stepGameType setupStep = iota
	stepPackSelection
)

// SetupView handles game type selection and pack selection.
type SetupView struct {
	packs    []types.ContentPack
	gameType types.GameType
	step     setupStep
	cursor   int
}

// NewSetupView creates a new setup view.
func NewSetupView(packs []types.ContentPack, gameType types.GameType) *SetupView {
	return &SetupView{
		packs:    packs,
		gameType: gameType,
		step:     stepGameType,
		cursor:   0,
	}
}

func (v *SetupView) Init() tea.Cmd { return nil }

func (v *SetupView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch v.step {
		case stepGameType:
			return v.updateGameType(msg)
		case stepPackSelection:
			return v.updatePackSelection(msg)
		}
	}
	return v, nil
}

func (v *SetupView) updateGameType(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "down", "left", "right", "t", "tab", "k", "j":
		if v.gameType == types.GameTypeSprint {
			v.gameType = types.GameTypeExpedition
		} else {
			v.gameType = types.GameTypeSprint
		}
		return v, func() tea.Msg {
			return GameAction{Action: types.SelectType(v.gameType)}
		}
	case "enter", " ":
		v.step = stepPackSelection
		v.cursor = 0
	case "esc":
		return v, func() tea.Msg {
			return GameAction{Action: types.ReturnToTitle()}
		}
	}
	return v, nil
}

func (v *SetupView) updatePackSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	available := v.availablePacks()
	switch msg.String() {
	case "up", "k":
		if v.cursor > 0 {
			v.cursor--
		}
	case "down", "j":
		if v.cursor < len(available)-1 {
			v.cursor++
		}
	case "enter":
		if len(available) > 0 {
			pack := available[v.cursor]
			seed := int(time.Now().UnixNano() % 2147483647)
			return v, func() tea.Msg {
				return PackSelectedMsg{Pack: pack, Seed: seed}
			}
		}
	case "esc":
		v.step = stepGameType
	}
	return v, nil
}

func (v *SetupView) availablePacks() []types.ContentPack {
	var result []types.ContentPack
	for _, pack := range v.packs {
		if v.packSupportsType(pack) {
			result = append(result, pack)
		}
	}
	// If no packs explicitly support the type, show all
	if len(result) == 0 {
		return v.packs
	}
	return result
}

func (v *SetupView) packSupportsType(pack types.ContentPack) bool {
	if len(pack.Supports) == 0 {
		return true
	}
	for _, gt := range pack.Supports {
		if gt == v.gameType {
			return true
		}
	}
	return false
}

func (v *SetupView) View() string {
	switch v.step {
	case stepGameType:
		return v.viewGameType()
	case stepPackSelection:
		return v.viewPackSelection()
	}
	return ""
}

func (v *SetupView) viewGameType() string {
	title := styleLabel.Render("Game Type") + "\n\n"

	sprintStyle := styleStat
	expeditionStyle := styleStat
	sprintIndicator := "  "
	expeditionIndicator := "  "

	if v.gameType == types.GameTypeSprint {
		sprintStyle = styleTitle
		sprintIndicator = "▸ "
	} else {
		expeditionStyle = styleTitle
		expeditionIndicator = "▸ "
	}

	sprint := sprintIndicator + sprintStyle.Render("Sprint") + "\n"
	sprintDesc := "    " + styleDim.Render("A linear run through 13 rooms. Fast and deadly.") + "\n\n"

	expedition := expeditionIndicator + expeditionStyle.Render("Expedition") + "\n"
	expeditionDesc := "    " + styleDim.Render("A branching dungeon with 25+ rooms. Explore freely.") + "\n\n"

	help := styleDim.Render("Press Tab to toggle, Enter to confirm")

	return title + sprint + sprintDesc + expedition + expeditionDesc + help
}

func (v *SetupView) viewPackSelection() string {
	title := styleLabel.Render("Choose a Pack") + "\n\n"

	available := v.availablePacks()
	var list string
	for i, pack := range available {
		cursor := "  "
		nameStyle := styleStat
		if i == v.cursor {
			cursor = "▸ "
			nameStyle = styleTitle
		}
		name := nameStyle.Render(pack.Meta.Name)
		desc := styleDim.Render(pack.Meta.Description)
		list += fmt.Sprintf("%s%s\n    %s\n\n", cursor, name, desc)
	}

	if len(available) == 0 {
		list = styleDim.Render("No packs available for this game type.") + "\n"
	}

	help := styleDim.Render("Arrow keys to navigate, Enter to select")

	return title + list + help
}

func (v *SetupView) KeyHints() []KeyHint {
	switch v.step {
	case stepGameType:
		return []KeyHint{
			{Key: "tab", Desc: "toggle"},
			{Key: "enter", Desc: "confirm"},
			{Key: "esc", Desc: "back"},
		}
	case stepPackSelection:
		return []KeyHint{
			{Key: "↑↓", Desc: "navigate"},
			{Key: "enter", Desc: "select"},
			{Key: "esc", Desc: "back"},
		}
	}
	return nil
}
