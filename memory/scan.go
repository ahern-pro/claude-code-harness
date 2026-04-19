package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.uber.org/zap"
)

const (
	defaultScanMaxFiles  = 50
	descriptionMaxLen    = 200
	bodyPreviewMaxLen    = 300
	descriptionScanLines = 10
)

func ScanMemoryFiles(cwd string, maxFiles int) ([]*MemoryHeader, error) {
	if maxFiles <= 0 {
		maxFiles = defaultScanMaxFiles
	}

	dir, err := GetProjectMemoryDir(cwd)
	if err != nil {
		return nil, err
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("glob memory dir %q: %w", dir, err)
	}

	headers := make([]*MemoryHeader, 0, len(matches))
	for _, path := range matches {
		if filepath.Base(path) == memoryEntrypointFileName {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			zap.L().Error("read memory file", zap.String("path", path), zap.Error(err))
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			zap.L().Error("stat memory file", zap.String("path", path), zap.Error(err))
			continue
		}
		header := parseMemoryFile(path, string(data))
		header.ModifiedAt = info.ModTime().Unix()
		headers = append(headers, header)
	}

	sort.SliceStable(headers, func(i, j int) bool {
		return headers[i].ModifiedAt > headers[j].ModifiedAt
	})
	if len(headers) > maxFiles {
		headers = headers[:maxFiles]
	}
	return headers, nil
}

func parseMemoryFile(path, content string) *MemoryHeader {
	lines := strings.Split(content, "\n")

	base := filepath.Base(path)
	title := strings.TrimSuffix(base, filepath.Ext(base))
	description := ""
	memoryType := ""
	bodyStart := 0

	// Parse YAML frontmatter (--- ... ---)
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				for _, fmLine := range lines[1:i] {
					key, value, ok := strings.Cut(fmLine, ":")
					if !ok {
						continue
					}
					key = strings.TrimSpace(key)
					value = strings.Trim(strings.TrimSpace(value), "'\"")
					if value == "" {
						continue
					}
					switch key {
					case "name":
						title = value
					case "description":
						description = value
					case "type":
						memoryType = value
					}
				}
				bodyStart = i + 1
				break
			}
		}
	}

	// Fallback: first non-empty, non-frontmatter, non-heading line as description.
	descLineIdx := -1
	if description == "" {
		end := bodyStart + descriptionScanLines
		if end > len(lines) {
			end = len(lines)
		}
		for idx := bodyStart; idx < end; idx++ {
			stripped := strings.TrimSpace(lines[idx])
			if stripped == "" || stripped == "---" || strings.HasPrefix(stripped, "#") {
				continue
			}
			if len(stripped) > descriptionMaxLen {
				stripped = stripped[:descriptionMaxLen]
			}
			description = stripped
			descLineIdx = idx
			break
		}
	}

	var bodyParts []string
	for idx := bodyStart; idx < len(lines); idx++ {
		stripped := strings.TrimSpace(lines[idx])
		if stripped == "" || strings.HasPrefix(stripped, "#") || idx == descLineIdx {
			continue
		}
		bodyParts = append(bodyParts, stripped)
	}
	bodyPreview := strings.Join(bodyParts, " ")
	if len(bodyPreview) > bodyPreviewMaxLen {
		bodyPreview = bodyPreview[:bodyPreviewMaxLen]
	}

	return &MemoryHeader{
		Path:        path,
		Title:       title,
		Description: description,
		MemoryType:  memoryType,
		BodyPreview: bodyPreview,
	}
}
