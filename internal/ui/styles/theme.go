package styles

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Background    lipgloss.Color
	Foreground    lipgloss.Color
	Subtext       lipgloss.Color
	Border        lipgloss.Color
	Selection     lipgloss.Color
	BranchMain    lipgloss.Color
	BranchFeature lipgloss.Color
	BranchHotfix  lipgloss.Color
	Tag           lipgloss.Color
	Head          lipgloss.Color
	DiffAdd       lipgloss.Color
	DiffRemove    lipgloss.Color
	DiffContext   lipgloss.Color
	CommitHash    lipgloss.Color
	Graph1        lipgloss.Color
	Graph2        lipgloss.Color
	Graph3        lipgloss.Color
	Graph4        lipgloss.Color
	Graph5        lipgloss.Color
}

func CatppuccinMocha() Theme {
	return Theme{
		Background:    lipgloss.Color("#1e1e2e"),
		Foreground:    lipgloss.Color("#cdd6f4"),
		Subtext:       lipgloss.Color("#a6adc8"),
		Border:        lipgloss.Color("#313244"),
		Selection:     lipgloss.Color("#45475a"),
		BranchMain:    lipgloss.Color("#a6e3a1"),
		BranchFeature: lipgloss.Color("#89b4fa"),
		BranchHotfix:  lipgloss.Color("#f38ba8"),
		Tag:           lipgloss.Color("#f9e2af"),
		Head:          lipgloss.Color("#cba6f7"),
		DiffAdd:       lipgloss.Color("#a6e3a1"),
		DiffRemove:    lipgloss.Color("#f38ba8"),
		DiffContext:   lipgloss.Color("#585b70"),
		CommitHash:    lipgloss.Color("#fab387"),
		Graph1:        lipgloss.Color("#89b4fa"),
		Graph2:        lipgloss.Color("#cba6f7"),
		Graph3:        lipgloss.Color("#94e2d5"),
		Graph4:        lipgloss.Color("#f9e2af"),
		Graph5:        lipgloss.Color("#a6e3a1"),
	}
}

func GetTheme(name string) Theme {
	switch name {
	case "catppuccin-mocha":
		return CatppuccinMocha()
	default:
		return CatppuccinMocha()
	}
}
