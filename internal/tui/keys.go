package tui

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	NextPane   key.Binding
	PrevPane   key.Binding
	ToggleHelp key.Binding
	ToggleLive key.Binding
	ToggleDone key.Binding
	SelectPane key.Binding
	Quit       key.Binding
	Filter     key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		NextPane: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next pane"),
		),
		PrevPane: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev pane"),
		),
		ToggleHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		ToggleLive: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "toggle follow"),
		),
		ToggleDone: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "toggle closed"),
		),
		SelectPane: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "drill in"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter focused list"),
		),
	}
}
