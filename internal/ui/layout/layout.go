package layout

import (
	"github.com/charmbracelet/lipgloss"
)

type Layout struct {
	width      int
	height     int
	splitRatio float64
}

func New(width, height int, splitRatio float64) *Layout {
	return &Layout{
		width:      width,
		height:     height,
		splitRatio: splitRatio,
	}
}

func (l *Layout) Calculate() (leftWidth, leftHeight, rightWidth, rightHeight int) {
	leftWidth = int(float64(l.width) * l.splitRatio)
	rightWidth = l.width - leftWidth

	contentHeight := l.height - 2

	leftHeight = contentHeight
	rightHeight = contentHeight

	return
}

func (l *Layout) Render(leftPanel, rightPanel, actionBar string) string {
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	return lipgloss.JoinVertical(lipgloss.Left,
		content,
		actionBar,
	)
}

func (l *Layout) SetSize(width, height int) {
	l.width = width
	l.height = height
}
