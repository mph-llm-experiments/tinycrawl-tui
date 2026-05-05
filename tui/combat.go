package tui

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/client"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/gm"
	kittyPkg "github.com/mph-llm-experiments/tinycrawl-tui/internal/kitty"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// combatSubState tracks the current sub-state of the combat view.
type combatSubState int

const (
	combatNormal combatSubState = iota
	combatTextInput
	combatWaitingGM
)

// gmResponseMsg carries the result of an async GM API call.
type gmResponseMsg struct {
	result *types.CreativeActionResult
	err    error
}

// CombatView displays the combat phase with monster card, log, and actions.
type CombatView struct {
	state     *types.GameState
	cd        types.ContentData
	client    *client.Client
	pack      types.ContentPack
	kitty     bool
	cursor    int
	inventory *InventoryView

	subState  combatSubState
	textInput textinput.Model
	spinner   spinner.Model
	gmError   string
}

// NewCombatView creates a combat view.
func NewCombatView(state *types.GameState, cd types.ContentData, c *client.Client, pack types.ContentPack, kitty bool) *CombatView {
	ti := textinput.New()
	ti.Placeholder = "Describe your creative action..."
	ti.CharLimit = 500
	ti.Width = 50

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return &CombatView{
		state:     state,
		cd:        cd,
		client:    c,
		pack:      pack,
		kitty:     kitty,
		subState:  combatNormal,
		textInput: ti,
		spinner:   sp,
	}
}

func (v *CombatView) Init() tea.Cmd { return nil }

func (v *CombatView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	switch v.subState {
	case combatTextInput:
		return v.updateTextInput(msg)
	case combatWaitingGM:
		return v.updateWaiting(msg)
	default:
		return v.updateNormal(msg)
	}
}

func (v *CombatView) combatActions() []combatAction {
	actions := []combatAction{
		{label: "Attack", action: types.AttackAction()},
		{label: "Flee", action: types.FleeAction()},
	}
	if !v.state.IsFinalRoom() {
		actions = append(actions, combatAction{label: "Run past", action: types.RunPast()})
	}
	actions = append(actions, combatAction{label: "Creative action", creative: true})
	return actions
}

type combatAction struct {
	label     string
	action    types.Action
	creative  bool
	inventory bool
}

func (v *CombatView) updateNormal(msg tea.Msg) (tea.Model, tea.Cmd) {
	actions := v.combatActions()
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "left", "k":
			v.cursor = (v.cursor - 1 + len(actions)) % len(actions)
			return v, nil
		case "down", "right", "j":
			v.cursor = (v.cursor + 1) % len(actions)
			return v, nil
		case "enter", " ":
			return v.selectCombatAction(actions[v.cursor])
		// Letter shortcuts (match web app)
		case "a":
			return v, func() tea.Msg { return GameAction{Action: types.AttackAction()} }
		case "f":
			return v, func() tea.Msg { return GameAction{Action: types.FleeAction()} }
		case "r":
			if !v.state.IsFinalRoom() {
				return v, func() tea.Msg { return GameAction{Action: types.RunPast()} }
			}
		case "c":
			v.subState = combatTextInput
			v.textInput.Reset()
			v.textInput.Focus()
			v.gmError = ""
			return v, v.textInput.Cursor.BlinkCmd()
		}
	}
	return v, nil
}

func (v *CombatView) selectCombatAction(a combatAction) (tea.Model, tea.Cmd) {
	if a.creative {
		v.subState = combatTextInput
		v.textInput.Reset()
		v.textInput.Focus()
		v.gmError = ""
		return v, v.textInput.Cursor.BlinkCmd()
	}
	if a.inventory {
		if v.state.Character != nil {
			v.inventory = NewInventoryView(v.state.Character)
		}
		return v, nil
	}
	action := a.action
	return v, func() tea.Msg { return GameAction{Action: action} }
}

func (v *CombatView) updateTextInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			v.subState = combatNormal
			v.textInput.Blur()
			return v, nil
		case "enter":
			text := strings.TrimSpace(v.textInput.Value())
			if text == "" {
				return v, nil
			}
			v.subState = combatWaitingGM
			v.textInput.Blur()
			return v, tea.Batch(
				v.spinner.Tick,
				v.sendCreativeAction(text),
			)
		}
	}
	var cmd tea.Cmd
	v.textInput, cmd = v.textInput.Update(msg)
	return v, cmd
}

func (v *CombatView) updateWaiting(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case gmResponseMsg:
		if msg.err != nil {
			v.gmError = msg.err.Error()
			v.subState = combatNormal
			return v, nil
		}
		v.subState = combatNormal
		return v, func() tea.Msg {
			return GameAction{Action: types.CreativeAction(msg.result)}
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *CombatView) sendCreativeAction(actionText string) tea.Cmd {
	return func() tea.Msg {
		if v.state.Combat == nil || v.state.Character == nil {
			return gmResponseMsg{err: fmt.Errorf("invalid combat state")}
		}

		room := v.state.CurrentRoom()
		if room == nil {
			return gmResponseMsg{err: fmt.Errorf("no current room")}
		}

		gmSystemPrompt := gm.BuildGmSystemPrompt(v.pack.GmPrompt, v.state.Expedition)
		userPrompt := gm.BuildCreativeActionPrompt(
			v.state.Combat.Monster,
			*v.state.Character,
			*room,
			actionText,
		)

		resp, err := v.client.SendCreativeAction(gmSystemPrompt, userPrompt)
		if err != nil {
			return gmResponseMsg{err: err}
		}
		result, err := gm.ValidateGmResponse(resp)
		return gmResponseMsg{result: result, err: err}
	}
}

func (v *CombatView) View() string {
	if v.inventory != nil {
		return v.inventory.View()
	}

	if v.state.Combat == nil {
		return styleDim.Render("No combat state.")
	}

	monster := v.state.Combat.Monster
	band := types.GetLightBand(v.state.Light)
	m := monster.Base

	var sections []string

	// ── Section header ──
	sections = append(sections, lipgloss.NewStyle().Width(contentWidth-4).Align(lipgloss.Center).
		Render(styleDim.Render("— combat —")))

	// ── Monster portrait (centered, stacked above name) ──
	if v.kitty && m.Image != "" && (band == types.LightBright || band == types.LightDim) {
		imgData := m.Image
		if idx := strings.Index(imgData, ","); idx >= 0 {
			imgData = imgData[idx+1:]
		}
		pngData, err := base64.StdEncoding.DecodeString(imgData)
		if err == nil {
			imageStr := kittyPkg.RenderImage(1, pngData, 10, 5)
			sections = append(sections, lipgloss.NewStyle().Width(contentWidth-4).Align(lipgloss.Center).
				Render(imageStr))
		}
	}

	// ── Monster name + stats ──
	var name string
	switch band {
	case types.LightBlack:
		name = "???"
	case types.LightDark:
		name = "Something"
	default:
		name = m.Name
	}

	monsterLines := []string{
		lipgloss.NewStyle().Width(contentWidth-4).Align(lipgloss.Center).
			Render(styleTitle.Render("\u2620 " + name)),
	}
	if band == types.LightBright {
		monsterLines = append(monsterLines,
			lipgloss.NewStyle().Width(contentWidth-4).Align(lipgloss.Center).
				Render(styleStat.Render(fmt.Sprintf("STR %d  DEX %d  WIL %d  HP %d/%d  Armor %d",
					m.Str, m.Dex, m.Wil, monster.CurrentHP, m.HP, m.Armor))),
			lipgloss.NewStyle().Width(contentWidth-4).Align(lipgloss.Center).
				Render(styleDim.Render(fmt.Sprintf("%s (%s)", m.Attack.Name, m.Attack.Die))),
		)
		if len(m.Weaknesses) > 0 {
			monsterLines = append(monsterLines,
				lipgloss.NewStyle().Width(contentWidth-4).Align(lipgloss.Center).
					Render(styleDanger.Render("Weak: "+strings.Join(m.Weaknesses, ", "))))
		}
	} else if band == types.LightDim {
		monsterLines = append(monsterLines,
			lipgloss.NewStyle().Width(contentWidth-4).Align(lipgloss.Center).
				Render(styleStat.Render(fmt.Sprintf("HP %d/%d  Armor %d", monster.CurrentHP, m.HP, m.Armor))))
	}
	sections = append(sections, strings.Join(monsterLines, "\n"))

	// ── Divider ──
	sections = append(sections, divider())

	// ── Combat log ──
	for _, entry := range v.state.Combat.Log {
		sections = append(sections, styleStat.Render(entry))
	}

	// ── Actions or creative input ──
	sections = append(sections, divider())

	switch v.subState {
	case combatTextInput:
		sections = append(sections, styleLabel.Render("Creative Action:"))
		sections = append(sections, v.textInput.View())
	case combatWaitingGM:
		sections = append(sections, styleDim.Render(v.spinner.View()+" Thinking..."))
	default:
		actions := v.combatActions()
		for i, a := range actions {
			if i == v.cursor {
				sections = append(sections, styleTitle.Render("▸ ")+lipgloss.NewStyle().Bold(true).Foreground(colorStat).Render(a.label))
			} else {
				sections = append(sections, styleDim.Render("  ")+styleStat.Render(a.label))
			}
		}
	}
	if v.gmError != "" {
		sections = append(sections, styleDanger.Render("GM: "+v.gmError))
	}

	return styleBox.Render(strings.Join(sections, "\n"))
}

func (v *CombatView) KeyHints() []KeyHint {
	if v.inventory != nil {
		return v.inventory.KeyHints()
	}

	switch v.subState {
	case combatTextInput:
		return []KeyHint{
			{Key: "enter", Desc: "submit"},
			{Key: "esc", Desc: "cancel"},
		}
	case combatWaitingGM:
		return []KeyHint{
			{Key: "...", Desc: "waiting for GM"},
		}
	}

	return []KeyHint{
		{Key: "↑↓", Desc: "choose"},
		{Key: "enter", Desc: "select"},
	}
}
