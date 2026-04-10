package compact

import (
	"github.com/li-zeyuan/claude-code-harness/api"
	"github.com/li-zeyuan/claude-code-harness/models"
)

type AutoCompactState struct {
	compacted           bool
	turnCounter         int
	consecutiveFailures int
}

func AutoCompactIfNeeded(
	messages []*models.ConversationMessage,
	model string,
	state *AutoCompactState,
	apiClient *api.SupportsStreamingMessages,
	systemPrompt string,
	preserveRecent int,
) ([]*models.ConversationMessage, bool) {
	if !shouldCompact(messages, model, state) {
		return messages, false
	}

	// 阶段1: microcompact 微型压缩
	messages, tokensFreed := microcompactMessages(messages)
	if tokensFreed > 0 && !shouldCompact(messages, model, state) {
		return messages, true
	}

	// 阶段2: LLMcompact 总结压缩
	result, err := compactConversation(messages, apiClient, model, systemPrompt, preserveRecent)
	if err != nil {
		return messages, false
	}

	return result, true
}

/*
微型压缩：替换历史工具调用结果为：“[Old tool result content cleared]”
*/
func microcompactMessages(messages []*models.ConversationMessage) ([]*models.ConversationMessage, int) {
	return messages, 0
}

func compactConversation(messages []*models.ConversationMessage, apiClient *api.SupportsStreamingMessages, model string, systemPrompt string, preserveRecent int) ([]*models.ConversationMessage, error) {
	return messages, nil
}

func shouldCompact(messages []*models.ConversationMessage, model string, state *AutoCompactState) bool {
	if state.consecutiveFailures >= 3 {
		return false
	}

	tokenCount := estimateMessageTokens(messages)
	threshold := getAutoCompactThreshold(model)
	return tokenCount >= threshold
}

func estimateMessageTokens(messages []*models.ConversationMessage) int {
	return 0
}

func getAutoCompactThreshold(model string) int {
	return 0
}
