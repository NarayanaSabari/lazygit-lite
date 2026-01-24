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

func (l *Layout) Calculate() (leftWidth, leftHeight, rightWidth, rightTopHeight, rightBottomHeight int) {
	leftWidth = int(float64(l.width) * l.splitRatio)
	rightWidth = l.width - leftWidth

	contentHeight := l.height - 2

	leftHeight = contentHeight
	rightTopHeight = contentHeight / 3
	rightBottomHeight = contentHeight - rightTopHeight

	return
}

func (l *Layout) Render(leftPanel, topRightPanel, bottomRightPanel, actionBar string) string {
	rightPanel := lipgloss.JoinVertical(lipgloss.Left, topRightPanel, bottomRightPanel)
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
