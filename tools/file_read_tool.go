package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/li-zeyuan/claude-code-harness/sandbox"
	"github.com/li-zeyuan/claude-code-harness/utils"
)

const (
	defaultOffset = 0
	defaultLimit  = 200
)

type FileReadToolInput struct {
	Path   string `json:"path" validate:"required"`
	Offset int    `json:"offset" validate:"min=0"`
	Limit  int    `json:"limit" validate:"min=1,max=2000"`
}

type FileReadTool struct {
	BaseTool
}

func NewFileReadTool() *FileReadTool {
	return &FileReadTool{
		BaseTool: BaseTool{
			Name:        "read_file",
			Description: "Read a text file from the local repository.",
			InputModel:  map[string]interface{}{},
		},
	}
}

func (frt *FileReadTool) Name() string {
	return frt.BaseTool.Name
}

func (frt *FileReadTool) IsReadOnly() bool {
	return true
}

func (frt *FileReadTool) Execute(input any, ctx *ToolExecutionContext) *ToolResult {
	args, ok := input.(FileReadToolInput)
	if !ok {
		return &ToolResult{Output: "Invalid input", IsError: true}
	}

	resolved := resolvePath(ctx.Cwd, args.Path)

	if sandbox.IsDockerSandboxActive() {
		allowed, reason := sandbox.ValidateSandboxPath(resolved, ctx.Cwd)
		if !allowed {
			return &ToolResult{Output: fmt.Sprintf("Sandbox: %s", reason), IsError: true}
		}
	}

	info, err := os.Stat(resolved)
	if os.IsNotExist(err) {
		return &ToolResult{Output: fmt.Sprintf("File not found: %s", resolved), IsError: true}
	}
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("Error accessing file: %s", err), IsError: true}
	}
	if info.IsDir() {
		return &ToolResult{Output: fmt.Sprintf("Cannot read directory: %s", resolved), IsError: true}
	}

	raw, err := os.ReadFile(resolved)
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("Error reading file: %s", err), IsError: true}
	}

	if bytes.Contains(raw, []byte{0x00}) {
		return &ToolResult{Output: fmt.Sprintf("Binary file cannot be read as text: %s", resolved), IsError: true}
	}

	lines := strings.Split(string(raw), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	end := args.Offset + args.Limit
	if end > len(lines) {
		end = len(lines)
	}
	start := args.Offset
	if start > len(lines) {
		start = len(lines)
	}
	selected := lines[start:end]

	if len(selected) == 0 {
		return &ToolResult{Output: fmt.Sprintf("(no content in selected range for %s)", resolved)}
	}

	numbered := make([]string, len(selected))
	for i, line := range selected {
		numbered[i] = fmt.Sprintf("%6d\t%s", args.Offset+i+1, line)
	}
	return &ToolResult{Output: strings.Join(numbered, "\n")}
}

func (frt *FileReadTool) Validate(input map[string]any) (any, error) {
	args := &FileReadToolInput{
		Offset: defaultOffset,
		Limit:  defaultLimit,
	}

	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	if err := json.Unmarshal(data, args); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if err := utils.Validator.Struct(args); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	return args, nil
}

func resolvePath(base, candidate string) string {
	if filepath.IsAbs(candidate) {
		return filepath.Clean(candidate)
	}
	return filepath.Clean(filepath.Join(base, candidate))
}
