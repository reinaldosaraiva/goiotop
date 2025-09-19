package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines all keyboard shortcuts
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding

	// Sorting
	SortCPU    key.Binding
	SortMemory key.Binding
	SortIO     key.Binding
	SortPID    key.Binding
	SortName   key.Binding

	// Filtering
	Filter     key.Binding
	Search     key.Binding
	ClearFilter key.Binding

	// Views
	Tab          key.Binding
	ProcessView  key.Binding
	SummaryView  key.Binding
	HelpView     key.Binding

	// Actions
	Refresh      key.Binding
	Details      key.Binding
	Kill         key.Binding
	ForceKill    key.Binding

	// General
	Help         key.Binding
	Quit         key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Navigation
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("PgUp", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("PgDown", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home"),
			key.WithHelp("Home", "go to top"),
		),
		End: key.NewBinding(
			key.WithKeys("end"),
			key.WithHelp("End", "go to bottom"),
		),

		// Sorting
		SortCPU: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "sort by CPU"),
		),
		SortMemory: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "sort by memory"),
		),
		SortIO: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "sort by I/O"),
		),
		SortPID: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "sort by PID"),
		),
		SortName: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "sort by name"),
		),

		// Filtering
		Filter: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "filter processes"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		ClearFilter: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("ESC", "clear filter"),
		),

		// Views
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "switch view"),
		),
		ProcessView: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "process list"),
		),
		SummaryView: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "system summary"),
		),
		HelpView: key.NewBinding(
			key.WithKeys("3", "?"),
			key.WithHelp("3/?", "help"),
		),

		// Actions
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Details: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "show details"),
		),
		Kill: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "kill process"),
		),
		ForceKill: key.NewBinding(
			key.WithKeys("X"),
			key.WithHelp("X", "force kill"),
		),

		// General
		Help: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns a short help string
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Tab,
		k.Help,
		k.Quit,
	}
}

// FullHelp returns the full help keybindings
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown}, // Navigation
		{k.SortCPU, k.SortMemory, k.SortIO, k.SortPID, k.SortName}, // Sorting
		{k.Filter, k.Search, k.ClearFilter}, // Filtering
		{k.Tab, k.ProcessView, k.SummaryView, k.HelpView}, // Views
		{k.Refresh, k.Details, k.Kill}, // Actions
		{k.Help, k.Quit}, // General
	}
}