package skills

import (
	"embed"
	"sort"
	"strings"
)

//go:embed content/*.md
var contentFS embed.FS

const (
	contentDir            = "content"
	sourceBundled         = "bundled"
	descriptionMaxLen     = 200
	frontmatterDelim      = "---"
	nameKey               = "name:"
	descriptionKey        = "description:"
	bundledFallbackPrefix = "Bundled skill: "
	userFallbackPrefix    = "Skill: "
)

func GetBundledSkills() []SkillDefinition {
	entries, err := contentFS.ReadDir(contentDir)
	if err != nil {
		return nil
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	result := make([]SkillDefinition, 0, len(names))
	for _, filename := range names {
		path := contentDir + "/" + filename
		raw, err := contentFS.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(raw)
		stem := strings.TrimSuffix(filename, ".md")
		name, description := parseFrontmatter(stem, content)
		result = append(result, SkillDefinition{
			Name:        name,
			Description: description,
			Content:     content,
			Source:      sourceBundled,
			Path:        path,
		})
	}
	return result
}

func parseFrontmatter(defaultName, content string) (string, string) {
	return parseSkillDoc(defaultName, content, bundledFallbackPrefix)
}

func parseSkillDoc(defaultName, content, fallbackPrefix string) (string, string) {
	name := defaultName
	description := ""
	lines := strings.Split(content, "\n")

	bodyStart := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == frontmatterDelim {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == frontmatterDelim {
				for _, fmLine := range lines[1:i] {
					fm := strings.TrimSpace(fmLine)
					switch {
					case strings.HasPrefix(fm, nameKey):
						if val := stripQuotes(strings.TrimSpace(fm[len(nameKey):])); val != "" {
							name = val
						}
					case strings.HasPrefix(fm, descriptionKey):
						if val := stripQuotes(strings.TrimSpace(fm[len(descriptionKey):])); val != "" {
							description = val
						}
					}
				}
				bodyStart = i + 1
				break
			}
		}
		if description != "" {
			return name, description
		}
	}

	for _, line := range lines[bodyStart:] {
		stripped := strings.TrimSpace(line)
		if strings.HasPrefix(stripped, "# ") {
			if heading := strings.TrimSpace(stripped[2:]); heading != "" {
				name = heading
			} else {
				name = defaultName
			}
			continue
		}
		if stripped == "" || strings.HasPrefix(stripped, frontmatterDelim) || strings.HasPrefix(stripped, "#") {
			continue
		}
		if len(stripped) > descriptionMaxLen {
			description = stripped[:descriptionMaxLen]
		} else {
			description = stripped
		}
		break
	}

	if description == "" {
		description = fallbackPrefix + name
	}
	return name, description
}

// stripQuotes removes a single pair of surrounding single or double quotes.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '\'' && last == '\'') || (first == '"' && last == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
