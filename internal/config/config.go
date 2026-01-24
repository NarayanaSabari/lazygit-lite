package config

type Config struct {
	UI          UIConfig          `yaml:"ui"`
	Layout      LayoutConfig      `yaml:"layout"`
	Git         GitConfig         `yaml:"git"`
	Keybindings KeybindingsConfig `yaml:"keybindings"`
	Commit      CommitConfig      `yaml:"commit"`
	Performance PerformanceConfig `yaml:"performance"`
}

type UIConfig struct {
	Theme      string `yaml:"theme"`
	Mouse      bool   `yaml:"mouse"`
	GraphStyle string `yaml:"graph_style"`
	ShowGraph  bool   `yaml:"show_graph"`
	DateFormat string `yaml:"date_format"`
}

type LayoutConfig struct {
	SplitRatio float64 `yaml:"split_ratio"`
	MinWidth   int     `yaml:"min_width"`
}

type GitConfig struct {
	AutoFetch          bool `yaml:"auto_fetch"`
	AutoFetchInterval  int  `yaml:"auto_fetch_interval"`
	PullRebase         bool `yaml:"pull_rebase"`
	PushForceWithLease bool `yaml:"push_force_with_lease"`
}

type KeybindingsConfig struct {
	Quit     []string `yaml:"quit"`
	Help     []string `yaml:"help"`
	Commit   []string `yaml:"commit"`
	Push     []string `yaml:"push"`
	Pull     []string `yaml:"pull"`
	Fetch    []string `yaml:"fetch"`
	Branch   []string `yaml:"branch"`
	Up       []string `yaml:"up"`
	Down     []string `yaml:"down"`
	Left     []string `yaml:"left"`
	Right    []string `yaml:"right"`
	Top      []string `yaml:"top"`
	Bottom   []string `yaml:"bottom"`
	PageUp   []string `yaml:"page_up"`
	PageDown []string `yaml:"page_down"`
}

type CommitConfig struct {
	SubjectLimit int    `yaml:"subject_limit"`
	BodyWrap     int    `yaml:"body_wrap"`
	Template     string `yaml:"template"`
}

type PerformanceConfig struct {
	MaxCommits        int `yaml:"max_commits"`
	LazyLoadThreshold int `yaml:"lazy_load_threshold"`
}
