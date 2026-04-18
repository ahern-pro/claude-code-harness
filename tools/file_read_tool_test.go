package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileReadTool_Metadata(t *testing.T) {
	tool := NewFileReadTool()

	if got := tool.Name(); got != "read_file" {
		t.Errorf("Name() = %q, want %q", got, "read_file")
	}
	if !tool.IsReadOnly() {
		t.Error("IsReadOnly() = false, want true")
	}
}

func TestFileReadTool_Validate(t *testing.T) {
	tool := NewFileReadTool()

	tests := []struct {
		name       string
		input      map[string]any
		wantErr    bool
		wantPath   string
		wantOffset int
		wantLimit  int
	}{
		{
			name:       "valid with defaults",
			input:      map[string]any{"path": "foo.txt"},
			wantPath:   "foo.txt",
			wantOffset: defaultOffset,
			wantLimit:  defaultLimit,
		},
		{
			name:       "valid with explicit offset and limit",
			input:      map[string]any{"path": "foo.txt", "offset": 10, "limit": 50},
			wantPath:   "foo.txt",
			wantOffset: 10,
			wantLimit:  50,
		},
		{
			name:       "offset and limit as float (json numeric)",
			input:      map[string]any{"path": "foo.txt", "offset": float64(5), "limit": float64(20)},
			wantPath:   "foo.txt",
			wantOffset: 5,
			wantLimit:  20,
		},
		{
			name:    "missing path",
			input:   map[string]any{},
			wantErr: true,
		},
		{
			name:    "empty path",
			input:   map[string]any{"path": ""},
			wantErr: true,
		},
		{
			name:    "negative offset",
			input:   map[string]any{"path": "foo.txt", "offset": -1},
			wantErr: true,
		},
		{
			name:    "limit below minimum",
			input:   map[string]any{"path": "foo.txt", "limit": 0},
			wantErr: true,
		},
		{
			name:    "limit above maximum",
			input:   map[string]any{"path": "foo.txt", "limit": 3000},
			wantErr: true,
		},
		{
			name:    "invalid path type",
			input:   map[string]any{"path": 123},
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
			args, ok := got.(*FileReadToolInput)
			if !ok {
				t.Fatalf("Validate returned %T, want *FileReadToolInput", got)
			}
			if args.Path != tc.wantPath {
				t.Errorf("Path = %q, want %q", args.Path, tc.wantPath)
			}
			if args.Offset != tc.wantOffset {
				t.Errorf("Offset = %d, want %d", args.Offset, tc.wantOffset)
			}
			if args.Limit != tc.wantLimit {
				t.Errorf("Limit = %d, want %d", args.Limit, tc.wantLimit)
			}
		})
	}
}

func TestResolvePath(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		candidate string
		want      string
	}{
		{
			name:      "absolute candidate",
			base:      "/tmp/work",
			candidate: "/etc/hosts",
			want:      "/etc/hosts",
		},
		{
			name:      "relative candidate joined with base",
			base:      "/tmp/work",
			candidate: "a/b.txt",
			want:      "/tmp/work/a/b.txt",
		},
		{
			name:      "relative with dot segments",
			base:      "/tmp/work",
			candidate: "./sub/../a.txt",
			want:      "/tmp/work/a.txt",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolvePath(tc.base, tc.candidate); got != tc.want {
				t.Errorf("resolvePath(%q, %q) = %q, want %q", tc.base, tc.candidate, got, tc.want)
			}
		})
	}
}

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
}

func TestFileReadTool_Execute_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	writeFile(t, path, []byte("alpha\nbeta\ngamma\n"))

	tool := NewFileReadTool()
	ctx := &ToolExecutionContext{Cwd: dir}

	result := tool.Execute(FileReadToolInput{Path: "sample.txt", Limit: defaultLimit}, ctx)
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}

	want := strings.Join([]string{
		"     1\talpha",
		"     2\tbeta",
		"     3\tgamma",
	}, "\n")
	if result.Output != want {
		t.Errorf("Output = %q, want %q", result.Output, want)
	}
}

func TestFileReadTool_Execute_OffsetLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.txt")
	writeFile(t, path, []byte("l1\nl2\nl3\nl4\nl5\n"))

	tool := NewFileReadTool()
	ctx := &ToolExecutionContext{Cwd: dir}

	result := tool.Execute(FileReadToolInput{
		Path:   "multi.txt",
		Offset: 1,
		Limit:  2,
	}, ctx)
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}

	want := "     2\tl2\n     3\tl3"
	if result.Output != want {
		t.Errorf("Output = %q, want %q", result.Output, want)
	}
}

func TestFileReadTool_Execute_OffsetBeyondEnd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "short.txt")
	writeFile(t, path, []byte("only-line\n"))

	tool := NewFileReadTool()
	ctx := &ToolExecutionContext{Cwd: dir}

	result := tool.Execute(FileReadToolInput{
		Path:   "short.txt",
		Offset: 100,
		Limit:  10,
	}, ctx)
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}
	if !strings.Contains(result.Output, "no content in selected range") {
		t.Errorf("expected empty-range message, got %q", result.Output)
	}
}

func TestFileReadTool_Execute_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	tool := NewFileReadTool()
	ctx := &ToolExecutionContext{Cwd: dir}

	result := tool.Execute(FileReadToolInput{Path: "does-not-exist.txt", Limit: defaultLimit}, ctx)
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	if !strings.Contains(result.Output, "File not found") {
		t.Errorf("expected 'File not found' message, got %q", result.Output)
	}
}

func TestFileReadTool_Execute_IsDirectory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tool := NewFileReadTool()
	ctx := &ToolExecutionContext{Cwd: dir}

	result := tool.Execute(FileReadToolInput{Path: "subdir", Limit: defaultLimit}, ctx)
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	if !strings.Contains(result.Output, "Cannot read directory") {
		t.Errorf("expected 'Cannot read directory' message, got %q", result.Output)
	}
}

func TestFileReadTool_Execute_BinaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "binary.bin")
	writeFile(t, path, []byte{'a', 'b', 0x00, 'c'})

	tool := NewFileReadTool()
	ctx := &ToolExecutionContext{Cwd: dir}

	result := tool.Execute(FileReadToolInput{Path: "binary.bin", Limit: defaultLimit}, ctx)
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	if !strings.Contains(result.Output, "Binary file cannot be read as text") {
		t.Errorf("expected 'Binary file' message, got %q", result.Output)
	}
}

func TestFileReadTool_Execute_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "abs.txt")
	writeFile(t, path, []byte("hello\n"))

	tool := NewFileReadTool()
	ctx := &ToolExecutionContext{Cwd: "/unused"}

	result := tool.Execute(FileReadToolInput{Path: path, Limit: defaultLimit}, ctx)
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}
	want := "     1\thello"
	if result.Output != want {
		t.Errorf("Output = %q, want %q", result.Output, want)
	}
}

func TestFileReadTool_Execute_InvalidInputType(t *testing.T) {
	tool := NewFileReadTool()
	ctx := &ToolExecutionContext{Cwd: t.TempDir()}

	result := tool.Execute("not-a-struct", ctx)
	if !result.IsError {
		t.Fatal("expected IsError=true for invalid input type")
	}
	if !strings.Contains(result.Output, "Invalid input") {
		t.Errorf("expected 'Invalid input' message, got %q", result.Output)
	}
}

func TestFileReadTool_Execute_FileWithoutTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no-nl.txt")
	writeFile(t, path, []byte("one\ntwo"))

	tool := NewFileReadTool()
	ctx := &ToolExecutionContext{Cwd: dir}

	result := tool.Execute(FileReadToolInput{Path: "no-nl.txt", Limit: defaultLimit}, ctx)
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}
	want := "     1\tone\n     2\ttwo"
	if result.Output != want {
		t.Errorf("Output = %q, want %q", result.Output, want)
	}
}
