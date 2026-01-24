package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		UI: UIConfig{
			Theme:      "catppuccin-mocha",
			Mouse:      true,
			GraphStyle: "unicode",
			ShowGraph:  true,
			DateFormat: "relative",
		},
		Layout: LayoutConfig{
			SplitRatio: 0.5,
			MinWidth:   80,
		},
		Git: GitConfig{
			AutoFetch:          false,
			AutoFetchInterval:  300,
			PullRebase:         true,
			PushForceWithLease: true,
		},
		Keybindings: KeybindingsConfig{
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
		},
		Commit: CommitConfig{
			SubjectLimit: 50,
			BodyWrap:     72,
			Template:     "",
		},
		Performance: PerformanceConfig{
			MaxCommits:        1000,
			LazyLoadThreshold: 100,
		},
	}
}

func Load() (*Config, error) {
	config := DefaultConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		return config, nil
	}

	configPath := filepath.Join(home, ".config", "lazygit-lite")
	viper.AddConfigPath(configPath)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return config, nil
		}
		return nil, err
	}

	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}
