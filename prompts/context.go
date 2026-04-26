package prompts

import (
	"fmt"
	"os"
	"strings"

	"github.com/li-zeyuan/claude-code-harness/config"
	"github.com/li-zeyuan/claude-code-harness/memory"
	"github.com/li-zeyuan/claude-code-harness/skills"
	"go.uber.org/zap"
)

func buildSkillSection(cwd string, extraSkillDirs []string, extraPluginRoots []string, settings *config.Settings) (string, error) {
	registry, err := skills.LoadSkillRegistry(
		skills.WithCwd(cwd),
		skills.WithExtraSkillDirs(extraSkillDirs...),
		skills.WithExtraPluginRoots(extraPluginRoots...),
		skills.WithSettings(settings),
	)
	if err != nil {
		return "", err
	}

	skills := registry.GetSkills()
	if len(skills) == 0 {
		return "", nil
	}

	lines := []string{
		"# Available Skills",
        "",
        "The following skills are available via the `skill` tool. ",
        "When a user's request matches a skill, invoke it with `skill(name=\"<skill_name>\")` ",
        "to load detailed instructions before proceeding.",
        "",
	}
	for _, skill := range skills {
		lines = append(lines, fmt.Sprintf("- **%s**: %s", skill.Name, skill.Description))
	}

	return strings.Join(lines, "\n"), nil
}

func BuildRuntimeSystemPrompt(
	settings *config.Settings,
	cwd string,
	latestUserPrompt string,
	extraSkillDirs []string,
	extraPluginRoots []string,
) string {

	sections := make([]string, 0)

	skillsSection, _ := buildSkillSection(cwd, extraSkillDirs, extraPluginRoots, settings)
	if len(skillsSection) > 0 {
		// && isCoordinatorMode
		sections = append(sections, skillsSection)
	}

	if settings.Memory.Enable {
		memorySection, err := memory.LoadMemoryPrompt(cwd, settings.Memory.MaxEntrypointLines)
		if err != nil {
			zap.L().Error("load memory prompt", zap.Error(err))
		}
		if memorySection != "" {
			sections = append(sections, memorySection)
		}

		if latestUserPrompt != "" {
			relevantMemories, err := memory.FindRelevantMemories(latestUserPrompt, cwd, settings.Memory.MaxFiles)
			if err != nil {
				zap.L().Error("find relevant memories", zap.Error(err))
			}
			if len(relevantMemories) > 0 {
				lines := make([]string, 0)
				lines = append(lines, "# Relevant Memories")
				for _, memory := range relevantMemories {
					body, err := os.ReadFile(memory.Path)
					if err != nil {
						zap.L().Error("read memory file", zap.String("path", memory.Path), zap.Error(err))
						continue
					}
					lines = append(
						lines,
						"",
						fmt.Sprintf("## %s", memory.Path),
						"```md",
						strings.TrimSpace(string(body))[:8000],
						"```",
					)
				}
				sections = append(sections, strings.Join(lines, "\n"))
			}
		}
	}

	return strings.Join(sections, "\n\n")
}
