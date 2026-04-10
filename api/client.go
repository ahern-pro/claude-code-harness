package api

import "github.com/li-zeyuan/claude-code-harness/models"

type ApiStreamEvent interface {
}

type ApiTextDeltaEvent struct {
	Text string
}

type ApiRetryEvent struct {
	message      string
	attempt      int
	maxAttempts  int
	delaySeconds float64
}

type ApiMessageCompleteEvent struct {
	Message *models.ConversationMessage
	usage *UsageSnapshot
	stopReason string
}

type ApiMessageRequest struct {
	Model        string
	Messages     []*models.ConversationMessage
	SystemPrompt string
	MaxTokens    int
}

type SupportsStreamingMessages struct {
}

func (c *SupportsStreamingMessages) StreamMessage(request *ApiMessageRequest) <-chan ApiStreamEvent {
	return nil
}
