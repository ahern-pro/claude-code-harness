package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/li-zeyuan/claude-code-harness/config"
)

const (
	sourceUser       = "user"
	userSkillsSubdir = "skills"
	skillFilename    = "SKILL.md"
)

type LoadOption func(*loadOptions)

type loadOptions struct {
	cwd string
	extraSkillDirs []string
	extraPluginRoots []string
	settings *config.Settings
}

func WithCwd(cwd string) LoadOption {
	return func(o *loadOptions) {
		o.cwd = cwd
	}
}

func WithExtraSkillDirs(dirs ...string) LoadOption {
	return func(o *loadOptions) {
		o.extraSkillDirs = append(o.extraSkillDirs, dirs...)
	}
}

func WithExtraPluginRoots(roots ...string) LoadOption {
	return func(o *loadOptions) {
		o.extraPluginRoots = append(o.extraPluginRoots, roots...)
	}
}

func WithSettings(settings *config.Settings) LoadOption {
	return func(o *loadOptions) {
		o.settings = settings
	}
}

func GetUserSkillsDir() (string, error) {
	cfgDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cfgDir, userSkillsSubdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create user skills dir %q: %w", dir, err)
	}
	return dir, nil
}

func LoadSkillRegistry(opts ...LoadOption) (*SkillRegistry, error) {
	cfg := &loadOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	registry := NewSkillRegistry()
	for _, skill := range GetBundledSkills() {
		registry.Register(skill)
	}

	userSkills, err := LoadUserSkills()
	if err != nil {
		return nil, err
	}
	for _, skill := range userSkills {
		registry.Register(skill)
	}

	if len(cfg.extraSkillDirs) > 0 {
		extras, err := LoadSkillsFromDirs(cfg.extraSkillDirs, sourceUser)
		if err != nil {
			return nil, err
		}
		for _, skill := range extras {
			registry.Register(skill)
		}
	}

	if len(cfg.cwd) > 0 {
		// todo load plugins skills from cwd
	}

	return registry, nil
}

func LoadUserSkills() ([]SkillDefinition, error) {
	dir, err := GetUserSkillsDir()
	if err != nil {
		return nil, err
	}
	return LoadSkillsFromDirs([]string{dir}, sourceUser)
}

func LoadSkillsFromDirs(directories []string, source string) ([]SkillDefinition, error) {
	if source == "" {
		source = sourceUser
	}
	var skills []SkillDefinition
	if len(directories) == 0 {
		return skills, nil
	}

	seen := make(map[string]struct{})
	for _, directory := range directories {
		root, err := expandAndResolve(directory)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(root, 0o755); err != nil {
			return nil, fmt.Errorf("create skills dir %q: %w", root, err)
		}

		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, fmt.Errorf("read skills dir %q: %w", root, err)
		}

		candidates := make([]string, 0, len(entries))
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillPath := filepath.Join(root, entry.Name(), skillFilename)
			info, statErr := os.Stat(skillPath)
			if statErr != nil || info.IsDir() {
				continue
			}
			candidates = append(candidates, skillPath)
		}
		sort.Strings(candidates)

		for _, path := range candidates {
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}

			raw, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read skill %q: %w", path, err)
			}
			content := string(raw)
			defaultName := filepath.Base(filepath.Dir(path))
			name, description := parseSkillDoc(defaultName, content, userFallbackPrefix)
			skills = append(skills, SkillDefinition{
				Name:        name,
				Description: description,
				Content:     content,
				Source:      source,
				Path:        path,
			})
		}
	}
	return skills, nil
}

func expandAndResolve(path string) (string, error) {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		if len(path) == 1 {
			path = home
		} else if path[1] == '/' || path[1] == filepath.Separator {
			path = filepath.Join(home, path[2:])
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", path, err)
	}
	return filepath.Clean(abs), nil
}
