package engine

import (
	"github.com/li-zeyuan/claude-code-harness/models"
	"github.com/li-zeyuan/claude-code-harness/tools"
)

type QueryEngine struct {
	systemPrompt string
	messages     []*models.ConversationMessage
	toolRegistry *tools.ToolRegistry
}

func NewQueryEngine(systemPrompt string, toolRegistry *tools.ToolRegistry) *QueryEngine {
	return &QueryEngine{
		systemPrompt: systemPrompt,
		toolRegistry: toolRegistry,
	}
}

func (qe *QueryEngine) SetSystemPrompt(systemPrompt string) {
	qe.systemPrompt = systemPrompt
}

func (qe *QueryEngine) SubmitMessage(prompt string) <-chan StreamEvent {
	queryCtx := NewQueryContext()
	qe.messages = append(qe.messages, models.FromUserText(prompt))

	for event := range RunQuery(queryCtx, qe.messages) {
		qe.messages = append(qe.messages, event.(*models.ConversationMessage))
	}

	return nil
}
