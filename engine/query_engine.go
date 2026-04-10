package engine

import "github.com/li-zeyuan/claude-code-harness/models"

type QueryEngine struct {
	systemPrompt string
	messages     []*models.ConversationMessage
}

func NewQueryEngine(systemPrompt string) *QueryEngine {
	return &QueryEngine{
		systemPrompt: systemPrompt,
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
