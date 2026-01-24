package keys

import tea "github.com/charmbracelet/bubbletea"

type KeyMap struct {
	Quit     []string
	Help     []string
	Commit   []string
	Push     []string
	Pull     []string
	Fetch    []string
	Branch   []string
	Up       []string
	Down     []string
	Left     []string
	Right    []string
	Top      []string
	Bottom   []string
	PageUp   []string
	PageDown []string
	Enter    []string
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:     []string{"q", "ctrl+c"},
		Help:     []string{"?"},
		Commit:   []string{"c"},
		Push:     []string{"p"},
		Pull:     []string{"P"},
		Fetch:    []string{"f"},
		Branch:   []string{"b"},
		Up:       []string{"k", "up"},
		Down:     []string{"j", "down"},
		Left:     []string{"h", "left"},
		Right:    []string{"l", "right"},
		Top:      []string{"g", "home"},
		Bottom:   []string{"G", "end"},
		PageUp:   []string{"ctrl+u"},
		PageDown: []string{"ctrl+d"},
		Enter:    []string{"enter"},
	}
}

func MatchesKey(msg tea.KeyMsg, keys []string) bool {
	for _, key := range keys {
		if msg.String() == key {
			return true
		}
	}
	return false
}
