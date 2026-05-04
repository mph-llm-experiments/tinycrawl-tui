package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/client"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// ossuaryDataMsg is sent when ossuary stats have been fetched.
type ossuaryDataMsg struct {
	stats *types.OssuaryStats
	err   error
}

// OssuaryView displays run statistics.
type OssuaryView struct {
	client  *client.Client
	online  bool
	stats   *types.OssuaryStats
	loading bool
	err     string
}

// NewOssuaryView creates an ossuary statistics view.
func NewOssuaryView(c *client.Client, online bool) *OssuaryView {
	return &OssuaryView{
		client:  c,
		online:  online,
		loading: true,
	}
}

func (v *OssuaryView) Init() tea.Cmd {
	return v.fetchStats()
}

func (v *OssuaryView) fetchStats() tea.Cmd {
	return func() tea.Msg {
		if v.online {
			stats, err := v.client.FetchOssuary()
			if err != nil {
				// Fall back to local stats on error
				return v.fetchLocalStats()
			}
			return ossuaryDataMsg{stats: stats, err: nil}
		}
		return v.fetchLocalStats()
	}
}

func (v *OssuaryView) fetchLocalStats() tea.Msg {
	local, err := client.LoadLocalStats()
	if err != nil {
		return ossuaryDataMsg{stats: nil, err: err}
	}

	// Convert PlayerStats to OssuaryStats
	stats := &types.OssuaryStats{
		TotalRuns:           local.TotalRuns,
		TotalVictories:      local.Victories,
		TotalDeaths:         local.TotalRuns - local.Victories,
		TotalMonstersKilled: local.TotalMonstersKilled,
	}

	// Calculate average depth
	if local.TotalRuns > 0 {
		totalDepth := 0
		for _, r := range local.Runs {
			totalDepth += r.Depth
		}
		stats.AverageDepth = float64(totalDepth) / float64(local.TotalRuns)
	}

	return ossuaryDataMsg{stats: stats, err: nil}
}

func (v *OssuaryView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ossuaryDataMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err.Error()
		} else {
			v.stats = msg.stats
		}
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return v, func() tea.Msg {
				return GameAction{Action: types.ReturnToTitle()}
			}
		}
	}
	return v, nil
}

func (v *OssuaryView) View() string {
	var sb strings.Builder

	sb.WriteString(styleTitle.Render("THE OSSUARY"))
	sb.WriteString("\n")
	sb.WriteString(styleDim.Render("Hall of the Fallen"))
	sb.WriteString("\n\n")

	if v.loading {
		sb.WriteString(styleDim.Render("Loading..."))
		return styleBox.Render(sb.String())
	}

	if v.err != "" {
		sb.WriteString(styleDanger.Render("Error: " + v.err))
		return styleBox.Render(sb.String())
	}

	if v.stats == nil {
		sb.WriteString(styleDim.Render("No runs recorded yet."))
		return styleBox.Render(sb.String())
	}

	s := v.stats

	// Summary stats
	sb.WriteString(styleLabel.Render("Total runs:   "))
	sb.WriteString(styleStat.Render(fmt.Sprintf("%d", s.TotalRuns)))
	sb.WriteString("\n")
	sb.WriteString(styleLabel.Render("Victories:    "))
	sb.WriteString(styleStat.Render(fmt.Sprintf("%d", s.TotalVictories)))
	sb.WriteString("\n")
	sb.WriteString(styleLabel.Render("Deaths:       "))
	sb.WriteString(styleStat.Render(fmt.Sprintf("%d", s.TotalDeaths)))
	sb.WriteString("\n")

	// Win rate
	winRate := 0
	if s.TotalRuns > 0 {
		winRate = int(float64(s.TotalVictories) / float64(s.TotalRuns) * 100)
	}
	sb.WriteString(styleLabel.Render("Win rate:     "))
	sb.WriteString(styleStat.Render(fmt.Sprintf("%d%%", winRate)))
	sb.WriteString("\n")

	sb.WriteString(styleLabel.Render("Avg depth:    "))
	sb.WriteString(styleStat.Render(fmt.Sprintf("%.1f", s.AverageDepth)))
	sb.WriteString("\n")

	sb.WriteString(styleLabel.Render("Monsters killed: "))
	sb.WriteString(styleStat.Render(fmt.Sprintf("%d", s.TotalMonstersKilled)))
	sb.WriteString("\n")

	// Deadliest creatures
	if len(s.Deadliest) > 0 {
		sb.WriteString("\n")
		sb.WriteString(styleLabel.Render("Deadliest creatures:"))
		sb.WriteString("\n")
		limit := len(s.Deadliest)
		if limit > 5 {
			limit = 5
		}
		for _, d := range s.Deadliest[:limit] {
			tier := ""
			if d.Tier != nil {
				tier = " (" + *d.Tier + ")"
			}
			sb.WriteString(styleDim.Render("  "))
			sb.WriteString(styleStat.Render(d.Name + tier))
			sb.WriteString(styleDim.Render(fmt.Sprintf(" — %d kills", d.Kills)))
			sb.WriteString("\n")
		}
	}

	// Pack stats
	if len(s.Packs) > 0 {
		sb.WriteString("\n")
		sb.WriteString(styleLabel.Render("By pack:"))
		sb.WriteString("\n")
		for _, p := range s.Packs {
			sb.WriteString(styleDim.Render("  "))
			sb.WriteString(styleStat.Render(p.Name))
			sb.WriteString(styleDim.Render(fmt.Sprintf(" — %d runs, %d%% win rate", p.Runs, p.WinRate)))
			sb.WriteString("\n")
		}
	}

	return styleBox.Render(sb.String())
}

func (v *OssuaryView) KeyHints() []KeyHint {
	return []KeyHint{
		{Key: "esc", Desc: "return"},
		{Key: "q", Desc: "return"},
	}
}
