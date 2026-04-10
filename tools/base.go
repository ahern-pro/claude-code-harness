package tools

type ToolExecutionContext struct {
	Cwd string
	Metadata map[string]any
}

type ToolResult struct {
	Output string
	IsError bool
	Metadata map[string]any
}

type BaseTool struct {
	Name string
}

type ToolRegistry struct {
	tools map[string]*BaseTool
}

func (tr *ToolRegistry) Register(tool *BaseTool) {
	tr.tools[tool.Name] = tool
}

func (tr *ToolRegistry) Get(name string) *BaseTool {
	return tr.tools[name]
}

func (bt *BaseTool) Validate(input map[string]any) (map[string]any, error) {
	return input, nil
}

func (bt *BaseTool) Execute(input map[string]any, ctx *ToolExecutionContext) (*ToolResult, error) {
	return nil, nil
}