package memory

import (
	"fmt"
	"os"
	"strings"
)

const DefaultMaxEntrypointLines = 200

func LoadMemoryPrompt(cwd string, maxEntrypointLines int) (string, error) {
	if maxEntrypointLines <= 0 {
		maxEntrypointLines = DefaultMaxEntrypointLines
	}

	memoryDir, err := GetProjectMemoryDir(cwd)
	if err != nil {
		return "", fmt.Errorf("resolve project memory dir: %w", err)
	}
	entrypoint, err := GetMemoryEntrypoint(cwd)
	if err != nil {
		return "", fmt.Errorf("resolve memory entrypoint: %w", err)
	}

	lines := []string{
		"# Memory",
		fmt.Sprintf("- Persistent memory directory: %s", memoryDir),
		"- Use this directory to store durable user or project context that should survive future sessions.",
		"- Prefer concise topic files plus an index entry in MEMORY.md.",
	}

	data, readErr := os.ReadFile(entrypoint)
	switch {
	case readErr == nil:
		contentLines := splitLines(string(data))
		if len(contentLines) > maxEntrypointLines {
			contentLines = contentLines[:maxEntrypointLines]
		}
		if len(contentLines) > 0 {
			lines = append(lines, "", "## MEMORY.md", "```md")
			lines = append(lines, contentLines...)
			lines = append(lines, "```")
		}
	case os.IsNotExist(readErr):
		lines = append(lines, "", "## MEMORY.md", "(not created yet)")
	default:
		return "", fmt.Errorf("read memory entrypoint %q: %w", entrypoint, readErr)
	}

	return strings.Join(lines, "\n"), nil
}

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	normalized = strings.TrimSuffix(normalized, "\n")
	return strings.Split(normalized, "\n")
}
