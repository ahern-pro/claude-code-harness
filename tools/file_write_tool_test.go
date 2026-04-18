package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileWriteTool_Metadata(t *testing.T) {
	tool := NewFileWriteTool()

	if got := tool.Name(); got != "write_file" {
		t.Errorf("Name() = %q, want %q", got, "write_file")
	}
	if tool.IsReadOnly() {
		t.Error("IsReadOnly() = true, want false")
	}
}

func TestFileWriteTool_Validate(t *testing.T) {
	tool := NewFileWriteTool()

	tests := []struct {
		name              string
		input             map[string]any
		wantErr           bool
		wantPath          string
		wantContent       string
		wantCreateDirsVal bool
	}{
		{
			name:              "valid with defaults (create_directories defaults to true)",
			input:             map[string]any{"path": "foo.txt", "content": "hello"},
			wantPath:          "foo.txt",
			wantContent:       "hello",
			wantCreateDirsVal: true,
		},
		{
			name:              "valid with explicit create_directories=false",
			input:             map[string]any{"path": "foo.txt", "content": "hi", "create_directories": false},
			wantPath:          "foo.txt",
			wantContent:       "hi",
			wantCreateDirsVal: false,
		},
		{
			name:              "valid with explicit create_directories=true",
			input:             map[string]any{"path": "foo.txt", "content": "", "create_directories": true},
			wantPath:          "foo.txt",
			wantContent:       "",
			wantCreateDirsVal: true,
		},
		{
			name:    "missing path",
			input:   map[string]any{"content": "hello"},
			wantErr: true,
		},
		{
			name:    "empty path",
			input:   map[string]any{"path": "", "content": "hello"},
			wantErr: true,
		},
		{
			name:    "invalid path type",
			input:   map[string]any{"path": 123, "content": "hello"},
			wantErr: true,
		},
		{
			name:    "invalid content type",
			input:   map[string]any{"path": "foo.txt", "content": 123},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tool.Validate(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (args=%+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			args, ok := got.(*FileWriteToolInput)
			if !ok {
				t.Fatalf("Validate returned %T, want *FileWriteToolInput", got)
			}
			if args.Path != tc.wantPath {
				t.Errorf("Path = %q, want %q", args.Path, tc.wantPath)
			}
			if args.Content != tc.wantContent {
				t.Errorf("Content = %q, want %q", args.Content, tc.wantContent)
			}
			if args.CreateDirectories == nil {
				t.Fatal("CreateDirectories = nil, want non-nil")
			}
			if *args.CreateDirectories != tc.wantCreateDirsVal {
				t.Errorf("CreateDirectories = %v, want %v", *args.CreateDirectories, tc.wantCreateDirsVal)
			}
		})
	}
}

func TestFileWriteTool_Execute_Success(t *testing.T) {
	dir := t.TempDir()
	tool := NewFileWriteTool()
	ctx := &ToolExecutionContext{Cwd: dir}

	result := tool.Execute(FileWriteToolInput{
		Path:    "out.txt",
		Content: "hello\nworld\n",
	}, ctx)
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}

	wrote := filepath.Join(dir, "out.txt")
	if !strings.Contains(result.Output, "Wrote ") {
		t.Errorf("expected 'Wrote' prefix in output, got %q", result.Output)
	}
	if !strings.Contains(result.Output, wrote) {
		t.Errorf("expected resolved path %q in output, got %q", wrote, result.Output)
	}

	data, err := os.ReadFile(wrote)
	if err != nil {
		t.Fatalf("read back error: %v", err)
	}
	if string(data) != "hello\nworld\n" {
		t.Errorf("file content = %q, want %q", string(data), "hello\nworld\n")
	}
}

func TestFileWriteTool_Execute_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "overwrite.txt")
	if err := os.WriteFile(path, []byte("old-content"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	tool := NewFileWriteTool()
	ctx := &ToolExecutionContext{Cwd: dir}

	result := tool.Execute(FileWriteToolInput{
		Path:    "overwrite.txt",
		Content: "new-content",
	}, ctx)
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back error: %v", err)
	}
	if string(data) != "new-content" {
		t.Errorf("file content = %q, want %q", string(data), "new-content")
	}
}

func TestFileWriteTool_Execute_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	tool := NewFileWriteTool()
	ctx := &ToolExecutionContext{Cwd: dir}

	result := tool.Execute(FileWriteToolInput{
		Path:    "nested/a/b/c.txt",
		Content: "deep",
	}, ctx)
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}

	got, err := os.ReadFile(filepath.Join(dir, "nested", "a", "b", "c.txt"))
	if err != nil {
		t.Fatalf("read back error: %v", err)
	}
	if string(got) != "deep" {
		t.Errorf("file content = %q, want %q", string(got), "deep")
	}
}

func TestFileWriteTool_Execute_NoCreateDirs_FailsWhenParentMissing(t *testing.T) {
	dir := t.TempDir()
	tool := NewFileWriteTool()
	ctx := &ToolExecutionContext{Cwd: dir}

	createDirs := false
	result := tool.Execute(FileWriteToolInput{
		Path:              "missing/sub/c.txt",
		Content:           "x",
		CreateDirectories: &createDirs,
	}, ctx)

	if !result.IsError {
		t.Fatal("expected IsError=true when parent dir missing and create_directories=false")
	}
	if !strings.Contains(result.Output, "Error writing file") {
		t.Errorf("expected 'Error writing file' message, got %q", result.Output)
	}
}

func TestFileWriteTool_Execute_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, "abs.txt")

	tool := NewFileWriteTool()
	ctx := &ToolExecutionContext{Cwd: "/unused"}

	result := tool.Execute(FileWriteToolInput{
		Path:    abs,
		Content: "abs-content",
	}, ctx)
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read back error: %v", err)
	}
	if string(data) != "abs-content" {
		t.Errorf("file content = %q, want %q", string(data), "abs-content")
	}
}

func TestFileWriteTool_Execute_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	tool := NewFileWriteTool()
	ctx := &ToolExecutionContext{Cwd: dir}

	result := tool.Execute(FileWriteToolInput{
		Path:    "empty.txt",
		Content: "",
	}, ctx)
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}

	data, err := os.ReadFile(filepath.Join(dir, "empty.txt"))
	if err != nil {
		t.Fatalf("read back error: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("file content length = %d, want 0", len(data))
	}
}

func TestFileWriteTool_Execute_InvalidInputType(t *testing.T) {
	tool := NewFileWriteTool()
	ctx := &ToolExecutionContext{Cwd: t.TempDir()}

	result := tool.Execute("not-a-struct", ctx)
	if !result.IsError {
		t.Fatal("expected IsError=true for invalid input type")
	}
	if !strings.Contains(result.Output, "Invalid input") {
		t.Errorf("expected 'Invalid input' message, got %q", result.Output)
	}
}

func TestFileWriteTool_Execute_AcceptsPointerInput(t *testing.T) {
	dir := t.TempDir()
	tool := NewFileWriteTool()
	ctx := &ToolExecutionContext{Cwd: dir}

	result := tool.Execute(&FileWriteToolInput{
		Path:    "ptr.txt",
		Content: "ptr",
	}, ctx)
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}

	data, err := os.ReadFile(filepath.Join(dir, "ptr.txt"))
	if err != nil {
		t.Fatalf("read back error: %v", err)
	}
	if string(data) != "ptr" {
		t.Errorf("file content = %q, want %q", string(data), "ptr")
	}
}

func TestFileWriteTool_Execute_NilPointerInput(t *testing.T) {
	tool := NewFileWriteTool()
	ctx := &ToolExecutionContext{Cwd: t.TempDir()}

	var nilInput *FileWriteToolInput
	result := tool.Execute(nilInput, ctx)
	if !result.IsError {
		t.Fatal("expected IsError=true for nil *FileWriteToolInput input")
	}
}
