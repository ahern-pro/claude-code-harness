package prompts

import "github.com/li-zeyuan/claude-code-harness/config"

func BuildRuntimeSystemPrompt(
	settings *config.Settings,
	cwd string,
	latestuserPrompt string,
) string {
	return "system prompt"
}
