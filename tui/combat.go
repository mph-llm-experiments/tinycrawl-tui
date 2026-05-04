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

func (v *CombatView) updateNormal(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "a":
			return v, func() tea.Msg {
				return GameAction{Action: types.AttackAction()}
			}
		case "f":
			return v, func() tea.Msg {
				return GameAction{Action: types.FleeAction()}
			}
		case "r":
			if !v.state.IsFinalRoom() {
				return v, func() tea.Msg {
					return GameAction{Action: types.RunPast()}
				}
			}
		case "c":
			v.subState = combatTextInput
			v.textInput.Reset()
			v.textInput.Focus()
			v.gmError = ""
			return v, v.textInput.Cursor.BlinkCmd()
		case "i":
			if v.state.Character != nil {
				v.inventory = NewInventoryView(v.state.Character)
			}
			return v, nil
		}
	}
	return v, nil
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

	var sections []string

	// Monster card
	monsterCard := renderMonsterCard(v.state.Combat.Monster, v.state.Light, v.kitty)
	sections = append(sections, monsterCard)

	// Combat log
	if len(v.state.Combat.Log) > 0 {
		logSection := v.renderCombatLog()
		sections = append(sections, logSection)
	}

	// GM error (if any)
	if v.gmError != "" {
		errLine := styleDanger.Render("GM Error: " + v.gmError)
		sections = append(sections, errLine)
	}

	// Sub-state dependent UI
	switch v.subState {
	case combatTextInput:
		inputSection := styleLabel.Render("Creative Action:") + "\n" + v.textInput.View()
		sections = append(sections, inputSection)
	case combatWaitingGM:
		waitSection := v.spinner.View() + " Thinking..."
		sections = append(sections, styleDim.Render(waitSection))
	}

	content := strings.Join(sections, "\n\n")
	return styleBox.Render(content)
}

func (v *CombatView) renderCombatLog() string {
	var lines []string
	lines = append(lines, styleLabel.Render("Combat Log"))
	for _, entry := range v.state.Combat.Log {
		lines = append(lines, styleStat.Render("  "+entry))
	}
	return strings.Join(lines, "\n")
}

func renderMonsterCard(monster types.MonsterInstance, light int, kitty bool) string {
	band := types.GetLightBand(light)
	m := monster.Base

	// Name
	var name string
	switch band {
	case types.LightBlack:
		name = "???"
	case types.LightDark:
		name = "Something"
	default:
		name = m.Name
	}

	header := styleTitle.Render("\u2620 " + name)

	// Stats (only in bright/dim)
	var stats string
	if band == types.LightBright {
		stats = fmt.Sprintf("STR %d  DEX %d  WIL %d\nHP %d/%d  Armor %d\n%s (%s)",
			m.Str, m.Dex, m.Wil, monster.CurrentHP, m.HP, m.Armor, m.Attack.Name, m.Attack.Die)
		if len(m.Weaknesses) > 0 {
			stats += "\nWeak to: " + strings.Join(m.Weaknesses, ", ")
		}
	} else if band == types.LightDim {
		stats = fmt.Sprintf("HP %d/%d  Armor %d", monster.CurrentHP, m.HP, m.Armor)
	}

	// Image (Kitty protocol)
	var image string
	if kitty && m.Image != "" && (band == types.LightBright || band == types.LightDim) {
		pngData, err := base64.StdEncoding.DecodeString(m.Image)
		if err == nil {
			image = kittyPkg.RenderInline(pngData, 6, 3)
		}
	}

	// Layout: image left, text right (if image available)
	if image != "" {
		return lipgloss.JoinHorizontal(lipgloss.Top, image, "  "+header+"\n  "+stats)
	}
	return styleBox.Render(header + "\n" + styleStat.Render(stats))
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

	hints := []KeyHint{
		{Key: "a", Desc: "attack"},
		{Key: "f", Desc: "flee"},
	}
	if !v.state.IsFinalRoom() {
		hints = append(hints, KeyHint{Key: "r", Desc: "run past"})
	}
	hints = append(hints, KeyHint{Key: "c", Desc: "creative"})
	hints = append(hints, KeyHint{Key: "i", Desc: "inventory"})
	return hints
}
