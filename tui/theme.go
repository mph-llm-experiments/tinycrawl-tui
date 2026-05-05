package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Content width for the main game panel — fixed to prevent resizing.
const contentWidth = 50

var (
	colorName   = lipgloss.AdaptiveColor{Light: "#8B0000", Dark: "#FF6B6B"}
	colorLabel  = lipgloss.AdaptiveColor{Light: "#6B4423", Dark: "#D4A76A"}
	colorStat   = lipgloss.AdaptiveColor{Light: "#4A3520", Dark: "#E8D5B7"}
	colorDim    = lipgloss.AdaptiveColor{Light: "#8B7355", Dark: "#8B7355"}
	colorBorder = lipgloss.AdaptiveColor{Light: "#8B7355", Dark: "#6B5335"}
	colorDanger = lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#FF4444"}
	colorLoot   = lipgloss.AdaptiveColor{Light: "#DAA520", Dark: "#FFD700"}
	colorSystem = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#999999"}

	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorName)
	styleLabel = lipgloss.NewStyle().Foreground(colorLabel).Italic(true)
	styleStat  = lipgloss.NewStyle().Foreground(colorStat)
	styleDim   = lipgloss.NewStyle().Foreground(colorDim)
	styleBox   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1).
			Width(contentWidth)
	styleDanger = lipgloss.NewStyle().Foreground(colorDanger)
	styleKey    = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#333", Dark: "#EEE"}).
			Background(lipgloss.AdaptiveColor{Light: "#DDD", Dark: "#444"}).
			Padding(0, 1)

	// Section divider within a panel
	styleDivider = lipgloss.NewStyle().Foreground(colorBorder)
)

func divider() string {
	return styleDivider.Render(strings.Repeat("─", contentWidth-4))
}

// Exported style accessors for use by phase views in other packages.

func StyleTitle() lipgloss.Style  { return styleTitle }
func StyleLabel() lipgloss.Style  { return styleLabel }
func StyleStat() lipgloss.Style   { return styleStat }
func StyleDim() lipgloss.Style    { return styleDim }
func StyleBox() lipgloss.Style    { return styleBox }
func StyleDanger() lipgloss.Style { return styleDanger }
func StyleKey() lipgloss.Style    { return styleKey }

func ColorName() lipgloss.AdaptiveColor   { return colorName }
func ColorLabel() lipgloss.AdaptiveColor   { return colorLabel }
func ColorStat() lipgloss.AdaptiveColor    { return colorStat }
func ColorDim() lipgloss.AdaptiveColor     { return colorDim }
func ColorBorder() lipgloss.AdaptiveColor  { return colorBorder }
func ColorDanger() lipgloss.AdaptiveColor  { return colorDanger }
func ColorLoot() lipgloss.AdaptiveColor    { return colorLoot }
func ColorSystem() lipgloss.AdaptiveColor  { return colorSystem }
