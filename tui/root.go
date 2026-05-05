package tui

import (
	"strings"

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
	m.ActiveView = m.viewForPhase(state.Phase)
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

	case PackSelectedMsg:
		m.Pack = msg.Pack
		m.Content = content.ContentFromPack(msg.Pack)
		newState := engine.Dispatch(&m.State, types.NewGame(msg.Seed), m.Content)
		m.State = *newState
		m.ActiveView = m.viewForPhase(m.State.Phase)
		initCmd := m.ActiveView.Init()
		return m, initCmd

	case SwapItemMsg:
		// Drop the old item, then take the loot item
		s1 := engine.Dispatch(&m.State, types.DropItem(msg.DropSlot), m.Content)
		m.State = *s1
		s2 := engine.Dispatch(&m.State, types.TakeItem(msg.TakeIndex), m.Content)
		oldPhase := m.State.Phase
		m.State = *s2
		if m.State.Phase != oldPhase {
			m.ActiveView = m.viewForPhase(m.State.Phase)
			initCmd := m.ActiveView.Init()
			if initCmd != nil {
				return m, initCmd
			}
		} else {
			m.refreshViewState()
		}

	case GameAction:
		oldPhase := m.State.Phase
		newState := engine.Dispatch(&m.State, msg.Action, m.Content)
		m.State = *newState
		if m.State.Phase != oldPhase {
			// Phase changed — create new view
			m.ActiveView = m.viewForPhase(m.State.Phase)
			initCmd := m.ActiveView.Init()
			if initCmd != nil {
				return m, initCmd
			}
		} else {
			// Same phase — update state pointer in existing view, preserve UI state
			m.refreshViewState()
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

	inGame := m.State.Phase == types.PhaseExploring || m.State.Phase == types.PhaseCombat ||
		m.State.Phase == types.PhaseLooting

	var column []string

	// Light meter at the top during gameplay
	if inGame {
		column = append(column, renderLightHeader(m.State.Light))
		column = append(column, "") // breathing room
	}

	// Main content
	if m.ActiveView != nil {
		column = append(column, m.ActiveView.View())
	}

	// Stats + online below content during gameplay
	if inGame {
		column = append(column, "") // breathing room
		bar := renderStatusBar(&m.State, m.Online, m.Width)
		if bar != "" {
			column = append(column, bar)
		}
	}

	// Key hints
	if m.ActiveView != nil {
		hints := m.ActiveView.KeyHints()
		if len(hints) > 0 {
			column = append(column, "") // breathing room
			column = append(column, lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center).
				Render(renderKeyHints(hints)))
		}
	}

	block := strings.Join(column, "\n")
	return lipgloss.NewStyle().Width(m.Width).Align(lipgloss.Center).Render(block)
}

// refreshViewState updates the state pointer in the current view without
// recreating it. This preserves local UI state (cursor, inventory overlay, etc.)
// across same-phase GameActions like drop_item or use_item.
func (m *RootModel) refreshViewState() {
	switch v := m.ActiveView.(type) {
	case *RoomView:
		v.state = &m.State
		// Rebuild inventory view's character reference if open
		if v.inventory != nil {
			v.inventory.char = m.State.Character
			v.inventory.items = v.inventory.buildEntries()
		}
	case *CombatView:
		v.state = &m.State
	case *DeathView:
		v.state = &m.State
	case *CreationView:
		v.char = m.State.Character
	}
}

func (m RootModel) viewForPhase(phase types.GamePhase) PhaseView {
	switch phase {
	case types.PhaseTitle:
		// Offer quick play with the current/default pack
		var quickPack *types.ContentPack
		if len(m.Packs) > 0 {
			p := m.Pack
			quickPack = &p
		}
		return NewTitleView(m.Online, quickPack)
	case types.PhaseGameSetup:
		return NewSetupView(m.Packs, m.State.GameType)
	case types.PhaseCharacterCreation:
		return NewCreationView(m.State.Character)
	case types.PhaseExploring, types.PhaseLooting:
		return NewRoomView(&m.State, m.Content, m.KittySupport)
	case types.PhaseCombat:
		return NewCombatView(&m.State, m.Content, m.Client, m.Pack, m.KittySupport)
	case types.PhaseDead, types.PhaseVictory:
		return NewDeathView(&m.State, m.Client, m.Online)
	case types.PhaseOssuary:
		return NewOssuaryView(m.Client, m.Online)
	default:
		return newPlaceholderView(phase)
	}
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
