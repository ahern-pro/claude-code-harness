package models

type ContentBlock interface {
}

type ConversationMessage struct {
	Role    string
	Content []ContentBlock
}

func (cm *ConversationMessage) ToolUses() []*ToolUseBlock {
	return nil
}

type ToolUseBlock struct {
	Name  string
	Id    string
	Input map[string]any
}

type ToolResultBlock struct {
	ToolUseID string
	Content string
	IsError bool
}

func FromUserText(text string) *ConversationMessage {
	return &ConversationMessage{}
}
