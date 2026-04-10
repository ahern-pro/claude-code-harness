package config

type Settings struct {
}

func (s *Settings) MergeCliOverrides(settingsOverrides map[string]interface{}) *Settings {
	return s
}

func LoadSettings() *Settings {
	return &Settings{}
}