package styles

type Styles struct {
	Theme Theme
}

func NewStyles(theme Theme) *Styles {
	return &Styles{
		Theme: theme,
	}
}
