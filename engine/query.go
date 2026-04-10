package engine

import (
	"fmt"
	"sync"

	"github.com/li-zeyuan/claude-code-harness/api"
	"github.com/li-zeyuan/claude-code-harness/hooks"
	"github.com/li-zeyuan/claude-code-harness/models"
	"github.com/li-zeyuan/claude-code-harness/services/compact"
	"github.com/li-zeyuan/claude-code-harness/tools"
)

type QueryContext struct {
	cwd            string
	maxTurns       int
	apiClient      *api.SupportsStreamingMessages
	systemPrompt   string
	preserveRecent int
	askUserPrompt  string
	model          string
	state          *compact.AutoCompactState
	maxTokens      int
	hookExecutor   *hooks.HooksExecutor
	toolRegistry   *tools.ToolRegistry
}

func NewQueryContext() *QueryContext {
	return &QueryContext{}
}

func RunQuery(queryCtx *QueryContext, messages []*models.ConversationMessage) <-chan StreamEvent {
	for turnCount := 0; turnCount < queryCtx.maxTurns; turnCount++ {
		// 1. Auto-Compact 检查
		messages, _ := compact.AutoCompactIfNeeded(messages, queryCtx.model, queryCtx.state, queryCtx.apiClient, queryCtx.systemPrompt, queryCtx.preserveRecent)

		var finalMessage *models.ConversationMessage

		// 2. 调用 LLM API
		for event := range queryCtx.apiClient.StreamMessage(&api.ApiMessageRequest{
			Model:        queryCtx.model,
			Messages:     messages,
			SystemPrompt: queryCtx.systemPrompt,
			MaxTokens:    queryCtx.maxTokens,
		}) {
			switch e := event.(type) {
			case *api.ApiTextDeltaEvent:
			case *api.ApiRetryEvent:

			case *api.ApiMessageCompleteEvent:
				finalMessage = e.Message
			default:
			}

			messages = append(messages, finalMessage)

			toolCalls := finalMessage.ToolUses()
			if len(toolCalls) == 0 {
				continue
			}

			toolResults := make([]*models.ToolResultBlock, 0)
			wg := sync.WaitGroup{}

			// 3. todo 调用工具
			for _, tc := range toolCalls {
				wg.Add(1)
				go func(tc *models.ToolUseBlock) {
					defer wg.Done()
					results := executeToolCall(queryCtx, tc)
					toolResults = append(toolResults, results)
				}(tc)
			}
			wg.Wait()

			// 4. 将 tool_result 作为 user 消息追加，继续循环
			messages = append(messages, &models.ConversationMessage{
				Role:    "user",
				Content: nil,
			})
		}
	}

	return nil
}

func executeToolCall(queryCtx *QueryContext, toolCall *models.ToolUseBlock) *models.ToolResultBlock {
	// 1. pre-hook 检查
	preHooks := queryCtx.hookExecutor.Execute(hooks.PRE_TOOL_USE, map[string]any{
		"tool_name":  toolCall.Name,
		"tool_input": toolCall.Input,
		"event_type": hooks.PRE_TOOL_USE,
	})
	if preHooks.Blocked() {
		return &models.ToolResultBlock{
			ToolUseID: toolCall.Id,
			Content:   fmt.Sprintf("Tool %s is blocked by pre-hook: %s", toolCall.Name, preHooks.Reason()),
			IsError:   true,
		}
	}
	// 2. 查找工具
	tool := queryCtx.toolRegistry.Get(toolCall.Name)
	if tool == nil {
		return &models.ToolResultBlock{
			ToolUseID: toolCall.Id,
			Content:   fmt.Sprintf("Tool %s not found", toolCall.Name),
			IsError:   true,
		}
	}
	// 3. 参数校验
	parsedInput, err := tool.Validate(toolCall.Input)
	if err != nil {
		return &models.ToolResultBlock{
			ToolUseID: toolCall.Id,
			Content:   fmt.Sprintf("Tool %s validation failed: %s", toolCall.Name, err.Error()),
			IsError:   true,
		}
	}
	// 4. 权限校验
	// 5. 执行工具
	result, err := tool.Execute(parsedInput, &tools.ToolExecutionContext{
		Cwd: queryCtx.cwd,
		Metadata: map[string]any{
			"tool_registry":   queryCtx.toolRegistry,
			"ask_user_prompt": queryCtx.askUserPrompt,
		},
	})
	if err != nil {
		return &models.ToolResultBlock{
			ToolUseID: toolCall.Id,
			Content:   fmt.Sprintf("Tool %s execution failed: %s", toolCall.Name, err.Error()),
			IsError:   true,
		}
	}
	toolResult := &models.ToolResultBlock{
		ToolUseID: toolCall.Id,
		Content:   result.Output,
		IsError:   result.IsError,
	}
	// 6. post-hook通知

	return toolResult
}
