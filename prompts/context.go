package prompts

import (
	"fmt"
	"os"
	"strings"

	"github.com/li-zeyuan/claude-code-harness/config"
	"github.com/li-zeyuan/claude-code-harness/memory"
	"go.uber.org/zap"
)

func BuildRuntimeSystemPrompt(
	settings *config.Settings,
	cwd string,
	latestUserPrompt string,
) string {

	sections := make([]string, 0)

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
