package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const memoryIndexHeader = "# Memory Index\n"

var slugPattern = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func ListMemoryFiles(cwd string) ([]string, error) {
	dir, err := GetProjectMemoryDir(cwd)
	if err != nil {
		return nil, err
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("glob memory dir %q: %w", dir, err)
	}
	sort.Strings(matches)
	return matches, nil
}

// todo 基于文件锁实现
func AddMemoryEntry(cwd, title, content string) (string, error) {
	dir, err := GetProjectMemoryDir(cwd)
	if err != nil {
		return "", err
	}

	slug := slugify(title)
	path := filepath.Join(dir, slug+".md")
	body := strings.TrimSpace(content) + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", fmt.Errorf("write memory file %q: %w", path, err)
	}

	entrypoint, err := GetMemoryEntrypoint(cwd)
	if err != nil {
		return "", err
	}

	existing := memoryIndexHeader
	if data, err := os.ReadFile(entrypoint); err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read memory entrypoint %q: %w", entrypoint, err)
	}

	name := filepath.Base(path)
	if !strings.Contains(existing, name) {
		existing = strings.TrimRight(existing, "\n\r\t ") + fmt.Sprintf("\n- [%s](%s)\n", title, name)
		if err := os.WriteFile(entrypoint, []byte(existing), 0o644); err != nil {
			return "", fmt.Errorf("update memory entrypoint %q: %w", entrypoint, err)
		}
	}
	return path, nil
}

func RemoveMemoryEntry(cwd, name string) (bool, error) {
	dir, err := GetProjectMemoryDir(cwd)
	if err != nil {
		return false, err
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return false, fmt.Errorf("glob memory dir %q: %w", dir, err)
	}
	sort.Strings(matches)

	var target string
	for _, path := range matches {
		base := filepath.Base(path)
		stem := strings.TrimSuffix(base, filepath.Ext(base))
		if stem == name || base == name {
			target = path
			break
		}
	}
	if target == "" {
		return false, nil
	}

	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("remove memory file %q: %w", target, err)
	}

	entrypoint, err := GetMemoryEntrypoint(cwd)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(entrypoint)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("read memory entrypoint %q: %w", entrypoint, err)
	}

	targetName := filepath.Base(target)
	lines := strings.Split(string(data), "\n")
	kept := lines[:0]
	for _, line := range lines {
		if !strings.Contains(line, targetName) {
			kept = append(kept, line)
		}
	}
	updated := strings.TrimRight(strings.Join(kept, "\n"), "\n\r\t ") + "\n"
	if err := os.WriteFile(entrypoint, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("update memory entrypoint %q: %w", entrypoint, err)
	}
	return true, nil
}

func slugify(title string) string {
	trimmed := strings.ToLower(strings.TrimSpace(title))
	slug := slugPattern.ReplaceAllString(trimmed, "_")
	slug = strings.Trim(slug, "_")
	if slug == "" {
		return "memory"
	}
	return slug
}
