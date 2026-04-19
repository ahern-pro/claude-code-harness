package config

type Settings struct {
	Memory *MemorySettings
}

func (s *Settings) MergeCliOverrides(settingsOverrides map[string]interface{}) *Settings {
	return s
}

func LoadSettings() *Settings {
	return &Settings{}
}

type MemorySettings struct {
	Enable bool
	MaxFiles int
	MaxEntrypointLines int
	ContextWindowTokens int
	AutoCompactThresholdTokens int
}