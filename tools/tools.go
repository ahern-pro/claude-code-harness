package tools

type Tool interface {
	Name() string
	Validate(input map[string]any) (any, error)
	Execute(input any, ctx *ToolExecutionContext) *ToolResult
}
func CreateDefaultToolRegistry(mcpManager interface{}) *ToolRegistry {
	registry := ToolRegistry{
		tools: make(map[string]Tool),
	}
	registry.Register(NewFileReadTool())
	registry.Register(NewFileWriteTool())
	registry.Register(NewWebSearchTool())
	return &registry
}
