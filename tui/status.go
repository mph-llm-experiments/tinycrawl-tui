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

func renderStatusBar(state *types.GameState, online bool, width int) string {
	if state.Character == nil {
		return ""
	}
	char := state.Character

	// Stats
	stats := fmt.Sprintf("HP %d/%d  STR %d  DEX %d  WIL %d  Armor %d",
		char.HP, char.MaxHP, char.Str, char.Dex, char.Wil, char.Armor)

	// Light meter (10 segments)
	lightBar := renderLightBar(state.Light)

	// Online indicator
	indicator := "○ offline"
	if online {
		indicator = "◉ online"
	}

	left := styleStat.Render(stats)
	middle := lightBar
	right := styleDim.Render(indicator)

	// Arrange with spacing
	gap := width - lipgloss.Width(left) - lipgloss.Width(middle) - lipgloss.Width(right)
	if gap < 2 {
		gap = 2
	}
	halfGap := gap / 2

	return left + strings.Repeat(" ", halfGap) + middle + strings.Repeat(" ", gap-halfGap) + right
}

func renderLightBar(light int) string {
	var bar strings.Builder
	bar.WriteString("Light ")
	for i := 0; i < 10; i++ {
		if i < light {
			bar.WriteString("█")
		} else {
			bar.WriteString("░")
		}
	}
	band := types.GetLightBand(light)
	color := colorStat
	switch band {
	case types.LightDim:
		color = colorLabel
	case types.LightDark:
		color = colorDanger
	case types.LightBlack:
		color = lipgloss.AdaptiveColor{Light: "#333", Dark: "#555"}
	}
	return lipgloss.NewStyle().Foreground(color).Render(bar.String())
}

func renderKeyHints(hints []KeyHint) string {
	var parts []string
	for _, h := range hints {
		parts = append(parts, styleKey.Render(h.Key)+" "+styleDim.Render(h.Desc))
	}
	return strings.Join(parts, "  ")
}
