package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	var sb strings.Builder

	// Headline
	sb.WriteString(styleDanger.Render("YOU HAVE FALLEN"))
	sb.WriteString("\n\n")

	// Character info
	if v.state.Character != nil {
		c := v.state.Character
		sb.WriteString(styleTitle.Render(c.Name))
		sb.WriteString("\n")
		sb.WriteString(styleStat.Render(fmt.Sprintf(
			"STR %d  DEX %d  WIL %d  HP %d/%d  Armor %d",
			c.Str, c.Dex, c.Wil, c.HP, c.MaxHP, c.Armor,
		)))
		sb.WriteString("\n\n")
	}

	// Run stats
	sb.WriteString(styleLabel.Render("Depth reached: "))
	sb.WriteString(styleStat.Render(fmt.Sprintf("%d", v.state.Depth())))
	sb.WriteString("\n")
	sb.WriteString(styleLabel.Render("Monsters killed: "))
	sb.WriteString(styleStat.Render(fmt.Sprintf("%d", v.state.MonstersKilled)))
	sb.WriteString("\n\n")

	// Final combat log entries
	if v.state.Combat != nil && len(v.state.Combat.Log) > 0 {
		sb.WriteString(styleLabel.Render("Final moments:"))
		sb.WriteString("\n")
		logEntries := v.state.Combat.Log
		start := 0
		if len(logEntries) > 5 {
			start = len(logEntries) - 5
		}
		for _, entry := range logEntries[start:] {
			sb.WriteString(styleDim.Render("  " + entry))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return styleBox.Render(sb.String())
}

func (v *DeathView) victoryView() string {
	var sb strings.Builder

	// Headline
	sb.WriteString(styleTitle.Render("YOU SURVIVED"))
	sb.WriteString("\n")
	sb.WriteString(styleDim.Render("~ * ~ * ~ * ~"))
	sb.WriteString("\n\n")

	// Character info
	if v.state.Character != nil {
		c := v.state.Character
		sb.WriteString(styleTitle.Render(c.Name))
		sb.WriteString("\n")
		sb.WriteString(styleStat.Render(fmt.Sprintf(
			"STR %d  DEX %d  WIL %d  HP %d/%d  Armor %d",
			c.Str, c.Dex, c.Wil, c.HP, c.MaxHP, c.Armor,
		)))
		sb.WriteString("\n\n")
	}

	// Run stats
	sb.WriteString(styleLabel.Render("Rooms explored: "))
	sb.WriteString(styleStat.Render(fmt.Sprintf("%d", v.state.RoomsVisited())))
	sb.WriteString("\n")
	sb.WriteString(styleLabel.Render("Monsters killed: "))
	sb.WriteString(styleStat.Render(fmt.Sprintf("%d", v.state.MonstersKilled)))
	sb.WriteString("\n\n")

	// Decorative flourish
	sb.WriteString(styleDim.Render("~ * ~ * ~ * ~"))
	sb.WriteString("\n")

	return styleBox.Render(sb.String())
}

func (v *DeathView) KeyHints() []KeyHint {
	return []KeyHint{
		{Key: "p", Desc: "play again"},
		{Key: "q", Desc: "quit"},
	}
}
