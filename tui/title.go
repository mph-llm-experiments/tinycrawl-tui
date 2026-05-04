package tui

import (
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
	online   bool
	cursor   int
	items    []menuItem
}

// NewTitleView creates a new title screen view.
func NewTitleView(online bool) *TitleView {
	items := []menuItem{
		{label: "Play", action: func() tea.Msg { return GameAction{Action: types.StartGameSetup()} }},
		{label: "Ossuary", action: func() tea.Msg { return GameAction{Action: types.OpenOssuary()} }},
		{label: "Quit", action: func() tea.Msg { return tea.Quit() }},
	}
	return &TitleView{
		online: online,
		cursor: 0,
		items:  items,
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
		case "p":
			return v, v.items[0].action
		case "o":
			return v, v.items[1].action
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
		cursor := "  "
		style := styleStat
		if i == v.cursor {
			cursor = "▸ "
			style = styleTitle
		}
		menu += cursor + style.Render(item.label) + "\n"
	}

	return title + "\n" +
		tagline + "\n\n" +
		indicator + "\n\n" +
		menu
}

func (v *TitleView) KeyHints() []KeyHint {
	return []KeyHint{
		{Key: "p", Desc: "play"},
		{Key: "o", Desc: "ossuary"},
		{Key: "q", Desc: "quit"},
	}
}
