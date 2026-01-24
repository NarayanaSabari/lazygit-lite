# lazygit-lite

> A lightweight, fast, and beautiful terminal-based Git visualization tool

**lazygit-lite** is a simple terminal UI for Git that focuses on visualization and common operations. Built with Go, Bubble Tea, and Lip Gloss.

## Features

- **Beautiful commit graph** - Unicode-based Git graph with branch decorations
- **Mouse support** - Click to select, scroll to navigate
- **Vim keybindings** - Navigate with j/k/h/l
- **Essential operations** - commit, push, pull, fetch
- **Fast** - Sub-second startup for most repositories
- **Catppuccin Mocha theme** - Modern, beautiful colors

## Installation

### Build from Source

```bash
git clone https://github.com/yourusername/lazygit-lite.git
cd lazygit-lite
make build
sudo make install
```

### Using Go

```bash
go install github.com/yourusername/lazygit-lite/cmd/lazygit-lite@latest
```

## Usage

Run `lazygit-lite` in any Git repository:

```bash
cd your-git-repo
lazygit-lite
```

## Keybindings

### Navigation
- `j` / `↓` - Move down
- `k` / `↑` - Move up
- `h` / `←` - Focus left panel
- `l` / `→` - Focus right panel
- `g` / `Home` - Go to top
- `G` / `End` - Go to bottom
- `Ctrl+D` - Page down
- `Ctrl+U` - Page up

### Actions
- `c` - Commit
- `p` - Push
- `P` - Pull
- `f` - Fetch
- `b` - Branch picker
- `Enter` - View commit details

### General
- `?` - Toggle help
- `q` / `Ctrl+C` - Quit

## Configuration

Configuration file location: `~/.config/lazygit-lite/config.yaml`

Example configuration:

```yaml
ui:
  theme: "catppuccin-mocha"
  mouse: true
  graph_style: "unicode"
  show_graph: true
  date_format: "relative"

layout:
  split_ratio: 0.5
  min_width: 80

git:
  auto_fetch: false
  pull_rebase: true
  push_force_with_lease: true
```

## Requirements

- Go 1.21+
- Git 2.30+
- Terminal with 256 colors or true color support

## Development

```bash
make build
make test
make run
```

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Inspired by [lazygit](https://github.com/jesseduffield/lazygit), [lazydocker](https://github.com/jesseduffield/lazydocker), and [k9s](https://github.com/derailed/k9s)
- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- Theme from [Catppuccin](https://github.com/catppuccin/catppuccin)
