package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/li-zeyuan/claude-code-harness/sandbox"
	"github.com/li-zeyuan/claude-code-harness/utils"
)

type FileWriteToolInput struct {
	Path             string `json:"path" validate:"required"`
	Content          string `json:"content"`
	CreateDirectories *bool `json:"create_directories,omitempty"`
}

type FileWriteTool struct {
	BaseTool
}

func NewFileWriteTool() *FileWriteTool {
	return &FileWriteTool{
		BaseTool: BaseTool{
			Name:        "write_file",
			Description: "Create or overwrite a text file in the local repository.",
			InputModel:  map[string]interface{}{},
		},
	}
}

func (fwt *FileWriteTool) Name() string {
	return fwt.BaseTool.Name
}

func (fwt *FileWriteTool) IsReadOnly() bool {
	return false
}

func (fwt *FileWriteTool) Validate(input map[string]any) (any, error) {
	args := &FileWriteToolInput{}

	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	if err := json.Unmarshal(data, args); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if args.CreateDirectories == nil {
		defaultCreate := true
		args.CreateDirectories = &defaultCreate
	}

	if err := utils.Validator.Struct(args); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	return args, nil
}

func (fwt *FileWriteTool) Execute(input any, ctx *ToolExecutionContext) *ToolResult {
	args, ok := input.(*FileWriteToolInput)
	if !ok {
		return &ToolResult{Output: fmt.Sprintf("Invalid input: %T", input), IsError: true}
	}

	resolved := resolvePath(ctx.Cwd, args.Path)

	if sandbox.IsDockerSandboxActive() {
		allowed, reason := sandbox.ValidateSandboxPath(resolved, ctx.Cwd)
		if !allowed {
			return &ToolResult{Output: fmt.Sprintf("Sandbox: %s", reason), IsError: true}
		}
	}

	createDirs := true
	if args.CreateDirectories != nil {
		createDirs = *args.CreateDirectories
	}

	if createDirs {
		if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
			return &ToolResult{Output: fmt.Sprintf("Error creating directories: %s", err), IsError: true}
		}
	}

	if err := os.WriteFile(resolved, []byte(args.Content), 0o644); err != nil {
		return &ToolResult{Output: fmt.Sprintf("Error writing file: %s", err), IsError: true}
	}

	return &ToolResult{Output: fmt.Sprintf("Wrote %s", resolved)}
}
