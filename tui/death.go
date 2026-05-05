package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/client"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// runRecordedMsg is sent when the run record has been saved.
type runRecordedMsg struct{ err error }

// DeathView handles both the death and victory end screens.
type DeathView struct {
	state    *types.GameState
	client   *client.Client
	online   bool
	recorded bool
}

// NewDeathView creates a death or victory screen view.
// It checks state.Phase to determine which to display.
func NewDeathView(state *types.GameState, c *client.Client, online bool) *DeathView {
	return &DeathView{
		state:  state,
		client: c,
		online: online,
	}
}

func (v *DeathView) Init() tea.Cmd {
	return v.recordRun()
}

func (v *DeathView) recordRun() tea.Cmd {
	return func() tea.Msg {
		if v.state.Character == nil {
			return runRecordedMsg{err: fmt.Errorf("no character state")}
		}

		result := "death"
		if v.state.Phase == types.PhaseVictory {
			result = "victory"
		}

		run := types.RunRecord{
			Name:           v.state.Character.Name,
			Depth:          v.state.Depth(),
			TotalRooms:     v.state.TotalRooms(),
			MonstersKilled: v.state.MonstersKilled,
			Result:         result,
			Timestamp:      time.Now().Unix(),
		}

		// Always save locally
		if err := client.SaveRun(run); err != nil {
			return runRecordedMsg{err: err}
		}

		// If online, also send to server
		if v.online {
			// Best-effort — don't fail the local save if remote fails
			_ = v.client.RecordRun(run)
		}

		return runRecordedMsg{err: nil}
	}
}

func (v *DeathView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case runRecordedMsg:
		v.recorded = true
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "p":
			return v, func() tea.Msg {
				return GameAction{Action: types.Restart()}
			}
		case "q":
			return v, tea.Quit
		}
	}
	return v, nil
}

func (v *DeathView) View() string {
	if v.state.Phase == types.PhaseVictory {
		return v.victoryView()
	}
	return v.deathView()
}

func (v *DeathView) deathView() string {
	center := lipgloss.NewStyle().Width(contentWidth - 4).Align(lipgloss.Center)
	var sections []string

	// Headline — bold, centered
	sections = append(sections, center.Render(
		lipgloss.NewStyle().Bold(true).Foreground(colorDanger).Render("YOU HAVE FALLEN")))
	sections = append(sections, "")

	// What killed me
	if len(v.state.Log) > 0 {
		lastLog := v.state.Log[len(v.state.Log)-1]
		sections = append(sections, center.Render(styleDim.Render(lastLog.Text)))
		sections = append(sections, "")
	}

	// Character info
	if v.state.Character != nil {
		c := v.state.Character
		sections = append(sections, center.Render(styleTitle.Render(c.Name)))
		sections = append(sections, center.Render(styleStat.Render(fmt.Sprintf(
			"STR %d  DEX %d  WIL %d  HP %d/%d  Armor %d",
			c.Str, c.Dex, c.Wil, c.HP, c.MaxHP, c.Armor))))
		sections = append(sections, "")
	}

	// Run stats — centered
	sections = append(sections, center.Render(
		styleLabel.Render("Depth reached: ")+styleStat.Render(fmt.Sprintf("%d", v.state.Depth()))))
	sections = append(sections, center.Render(
		styleLabel.Render("Monsters killed: ")+styleStat.Render(fmt.Sprintf("%d", v.state.MonstersKilled))))

	return styleBox.Render(strings.Join(sections, "\n"))
}

func (v *DeathView) victoryView() string {
	center := lipgloss.NewStyle().Width(contentWidth - 4).Align(lipgloss.Center)
	var sections []string

	sections = append(sections, center.Render(
		lipgloss.NewStyle().Bold(true).Foreground(colorName).Render("YOU SURVIVED")))
	sections = append(sections, center.Render(styleDim.Render("~ * ~ * ~ * ~")))
	sections = append(sections, "")

	if v.state.Character != nil {
		c := v.state.Character
		sections = append(sections, center.Render(styleTitle.Render(c.Name)))
		sections = append(sections, center.Render(styleStat.Render(fmt.Sprintf(
			"STR %d  DEX %d  WIL %d  HP %d/%d  Armor %d",
			c.Str, c.Dex, c.Wil, c.HP, c.MaxHP, c.Armor))))
		sections = append(sections, "")
	}

	sections = append(sections, center.Render(
		styleLabel.Render("Rooms explored: ")+styleStat.Render(fmt.Sprintf("%d", v.state.RoomsVisited()))))
	sections = append(sections, center.Render(
		styleLabel.Render("Monsters killed: ")+styleStat.Render(fmt.Sprintf("%d", v.state.MonstersKilled))))
	sections = append(sections, "")
	sections = append(sections, center.Render(styleDim.Render("~ * ~ * ~ * ~")))

	return styleBox.Render(strings.Join(sections, "\n"))
}

func (v *DeathView) KeyHints() []KeyHint {
	return []KeyHint{
		{Key: "p", Desc: "play again"},
		{Key: "q", Desc: "quit"},
	}
}
