package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// KeyHint represents a keyboard shortcut hint shown in the UI.
type KeyHint struct {
	Key  string
	Desc string
}

var styleStatLabel = lipgloss.NewStyle().Foreground(colorLabel).Bold(true)

// renderLightHeader renders the light meter as a centered line above the main content.
func renderLightHeader(light int) string {
	band := types.GetLightBand(light)

	var bar strings.Builder
	for i := 0; i < 10; i++ {
		if i < light {
			bar.WriteString("█")
		} else {
			bar.WriteString("░")
		}
	}

	color := colorStat
	switch band {
	case types.LightDim:
		color = colorLabel
	case types.LightDark:
		color = colorDanger
	case types.LightBlack:
		color = lipgloss.AdaptiveColor{Light: "#333", Dark: "#555"}
	}

	meter := lipgloss.NewStyle().Foreground(color).Render(bar.String())
	line := styleDim.Render("◈ ") + meter + styleDim.Render(" ◈")

	return lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center).Render(line)
}

// renderStatusBar renders character stats and online indicator below the main content.
func renderStatusBar(state *types.GameState, online bool, _ int) string {
	if state.Character == nil {
		return ""
	}
	char := state.Character

	// HP (danger-colored when low)
	hp := fmt.Sprintf("%d/%d", char.HP, char.MaxHP)
	if char.HP <= char.MaxHP/3 {
		hp = styleDanger.Render(hp)
	} else {
		hp = styleStat.Render(hp)
	}

	// Stats line — centered
	statsLine := strings.Join([]string{
		styleStatLabel.Render("HP ") + hp,
		styleStatLabel.Render("STR ") + styleStat.Render(fmt.Sprintf("%d", char.Str)),
		styleStatLabel.Render("DEX ") + styleStat.Render(fmt.Sprintf("%d", char.Dex)),
		styleStatLabel.Render("WIL ") + styleStat.Render(fmt.Sprintf("%d", char.Wil)),
		styleStatLabel.Render("Armor ") + styleStat.Render(fmt.Sprintf("%d", char.Armor)),
	}, "  ")

	// Online indicator
	indicator := styleDim.Render("○ offline")
	if online {
		indicator = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#228B22", Dark: "#66BB6A"}).Render("◉") +
			styleDim.Render(" online")
	}

	centered := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center)

	return strings.Join([]string{
		centered.Render(statsLine),
		centered.Render(indicator),
	}, "\n\n")
}

func renderKeyHints(hints []KeyHint) string {
	var parts []string
	for _, h := range hints {
		parts = append(parts, styleKey.Render(h.Key)+" "+styleDim.Render(h.Desc))
	}
	return strings.Join(parts, "  ")
}
