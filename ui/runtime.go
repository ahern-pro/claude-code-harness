package ui

import (
	"log"
	"os"

	"github.com/li-zeyuan/claude-code-harness/config"
	"github.com/li-zeyuan/claude-code-harness/engine"
	"github.com/li-zeyuan/claude-code-harness/prompts"
)

type RuntimeBundle struct {
	engine *engine.QueryEngine
}

func BuildRuntime(prompt string) *RuntimeBundle {
	settings := config.LoadSettings().MergeCliOverrides(nil)
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
		return nil
	}

	engine := engine.NewQueryEngine(
		prompts.BuildRuntimeSystemPrompt(settings, cwd, prompt),
	)
	return &RuntimeBundle{
		engine: engine,
	}
}

func HandleLine(bundle *RuntimeBundle, line string, renderEvent interface{}) {
	cwd, _ := os.Getwd()
	systemPrompt := prompts.BuildRuntimeSystemPrompt(config.LoadSettings(), cwd, line)
	bundle.engine.SetSystemPrompt(systemPrompt)

	for _ = range bundle.engine.SubmitMessage(line) {
		// renderEvent(event)
	}
}