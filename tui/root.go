package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/client"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/content"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/engine"
	"github.com/mph-llm-experiments/tinycrawl-tui/internal/types"
)

// GameAction wraps an engine Action as a tea.Msg.
type GameAction struct {
	Action types.Action
}

// PhaseView is a sub-model for a game phase.
type PhaseView interface {
	tea.Model
	KeyHints() []KeyHint
}

// RootModel is the top-level Bubbletea model that coordinates game state and phase views.
type RootModel struct {
	State        types.GameState
	Content      types.ContentData
	Pack         types.ContentPack
	Packs        []types.ContentPack
	Client       *client.Client
	Online       bool
	KittySupport bool
	Width        int
	Height       int
	ActiveView   PhaseView
}

// NewRootModel creates a RootModel with initial state.
func NewRootModel(cfg client.Config, packs []types.ContentPack, kitty bool) RootModel {
	c := client.New(cfg)
	online := c.Ping()

	state := engine.CreateInitialState(0)

	// Default pack
	var pack types.ContentPack
	if len(packs) > 0 {
		pack = packs[0]
	}

	m := RootModel{
		State:        state,
		Content:      content.ContentFromPack(pack),
		Pack:         pack,
		Packs:        packs,
		Client:       c,
		Online:       online,
		KittySupport: kitty,
	}
	m.ActiveView = newPlaceholderView(state.Phase)
	return m
}

func (m RootModel) Init() tea.Cmd {
	return nil
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case GameAction:
		oldPhase := m.State.Phase
		newState := engine.Dispatch(&m.State, msg.Action, m.Content)
		m.State = *newState
		if m.State.Phase != oldPhase {
			m.ActiveView = m.viewForPhase(m.State.Phase)
		}
	}

	if m.ActiveView != nil {
		var cmd tea.Cmd
		updated, cmd := m.ActiveView.Update(msg)
		m.ActiveView = updated.(PhaseView)
		return m, cmd
	}
	return m, nil
}

func (m RootModel) View() string {
	if m.Width == 0 {
		return ""
	}

	// Main content
	var mainContent string
	if m.ActiveView != nil {
		mainContent = m.ActiveView.View()
	}

	// Status bar (only during gameplay phases)
	var statusBar string
	var keyHints string
	if m.State.Phase == types.PhaseExploring || m.State.Phase == types.PhaseCombat ||
		m.State.Phase == types.PhaseLooting {
		statusBar = renderStatusBar(&m.State, m.Online, m.Width)
	}
	if m.ActiveView != nil {
		hints := m.ActiveView.KeyHints()
		if len(hints) > 0 {
			keyHints = renderKeyHints(hints)
		}
	}

	// Layout: content centered, status bar and hints at bottom
	contentHeight := m.Height - 2 // reserve for status + hints
	if statusBar != "" {
		contentHeight--
	}
	if keyHints != "" {
		contentHeight--
	}
	_ = contentHeight // will be used for viewport clipping in later tasks

	// Center content horizontally
	centered := lipgloss.NewStyle().Width(m.Width).Align(lipgloss.Center).Render(mainContent)

	// Build final layout
	result := centered
	if statusBar != "" {
		result += "\n" + statusBar
	}
	if keyHints != "" {
		hintsCentered := lipgloss.NewStyle().Width(m.Width).Align(lipgloss.Center).Render(keyHints)
		result += "\n" + hintsCentered
	}

	return result
}

func (m RootModel) viewForPhase(phase types.GamePhase) PhaseView {
	// Placeholder — will be replaced as we implement each phase view
	return newPlaceholderView(phase)
}

// placeholderView is a temporary stand-in for unimplemented phases.
type placeholderView struct {
	phase types.GamePhase
}

func newPlaceholderView(phase types.GamePhase) *placeholderView {
	return &placeholderView{phase: phase}
}

func (v *placeholderView) Init() tea.Cmd                         { return nil }
func (v *placeholderView) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return v, nil }
func (v *placeholderView) View() string {
	return styleTitle.Render("CAIRN CRAWLER") + "\n\n" +
		styleDim.Render("Phase: "+string(v.phase)) + "\n\n" +
		styleDim.Render("Press ctrl+c to quit")
}
func (v *placeholderView) KeyHints() []KeyHint {
	return []KeyHint{{Key: "ctrl+c", Desc: "quit"}}
}
