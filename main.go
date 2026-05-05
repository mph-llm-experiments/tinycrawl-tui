package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/client"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/content"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/kitty"
	"github.com/mph-llm-experiments/tinycrawl-tui/tui"
)

func main() {
	cfg, _ := client.LoadConfig()
	packs, _ := content.LoadEmbeddedPacks()
	localPacks, _ := content.LoadLocalPacks(filepath.Join(client.ConfigDir(), "packs"))
	packs = append(packs, localPacks...)
	var kittySupport bool
	switch cfg.ImageMode {
	case "kitty":
		kittySupport = true
	case "off":
		kittySupport = false
	default:
		kittySupport = kitty.Detect()
	}

	m := tui.NewRootModel(cfg, packs, kittySupport)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
