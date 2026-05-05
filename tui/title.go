package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

type menuItem struct {
	label  string
	action func() tea.Msg
}

// TitleView displays the title screen with a menu.
type TitleView struct {
	online    bool
	cursor    int
	items     []menuItem
	quickPack *types.ContentPack
}

// NewTitleView creates a new title screen view.
// If quickPack is provided, a "Quick Play" option appears that starts immediately.
func NewTitleView(online bool, quickPack *types.ContentPack) *TitleView {
	var items []menuItem

	if quickPack != nil {
		pack := *quickPack
		items = append(items, menuItem{
			label: "Quick Play — " + pack.Meta.Name,
			action: func() tea.Msg {
				return PackSelectedMsg{Pack: pack, Seed: int(time.Now().UnixNano())}
			},
		})
	}

	items = append(items,
		menuItem{label: "New Game", action: func() tea.Msg { return GameAction{Action: types.StartGameSetup()} }},
		menuItem{label: "Ossuary", action: func() tea.Msg { return GameAction{Action: types.OpenOssuary()} }},
		menuItem{label: "Quit", action: func() tea.Msg { return tea.Quit() }},
	)
	return &TitleView{
		online:    online,
		cursor:    0,
		items:     items,
		quickPack: quickPack,
	}
}

func (v *TitleView) Init() tea.Cmd { return nil }

func (v *TitleView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "enter":
			return v, v.items[v.cursor].action
		case " ":
			return v, v.items[v.cursor].action
		case "q":
			return v, tea.Quit
		}
	}
	return v, nil
}

func (v *TitleView) View() string {
	title := styleTitle.Render("CAIRN CRAWLER")
	tagline := styleDim.Render("A terminal dungeon crawler")

	// Online status
	indicator := styleDim.Render("○ offline")
	if v.online {
		indicator = lipgloss.NewStyle().Foreground(colorStat).Render("◉ online")
	}

	// Menu
	var menu string
	for i, item := range v.items {
		if i == v.cursor {
			menu += styleTitle.Render("▸ ") + lipgloss.NewStyle().Bold(true).Foreground(colorStat).Render(item.label) + "\n"
		} else {
			menu += styleDim.Render("  ") + styleStat.Render(item.label) + "\n"
		}
	}

	return title + "\n" +
		tagline + "\n\n" +
		indicator + "\n\n" +
		menu
}

func (v *TitleView) KeyHints() []KeyHint {
	return []KeyHint{
		{Key: "↑↓", Desc: "choose"},
		{Key: "enter", Desc: "select"},
		{Key: "q", Desc: "quit"},
	}
}
