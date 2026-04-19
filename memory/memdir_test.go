package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMemoryPrompt_NoEntrypoint(t *testing.T) {
	setDataDirEnv(t)

	projectDir := t.TempDir()

	got, err := LoadMemoryPrompt(projectDir, 0)
	if err != nil {
		t.Fatalf("LoadMemoryPrompt: %v", err)
	}

	memoryDir, err := GetProjectMemoryDir(projectDir)
	if err != nil {
		t.Fatalf("GetProjectMemoryDir: %v", err)
	}

	wantSubstrings := []string{
		"# Memory",
		"- Persistent memory directory: " + memoryDir,
		"- Use this directory to store durable user or project context that should survive future sessions.",
		"- Prefer concise topic files plus an index entry in MEMORY.md.",
		"## MEMORY.md",
		"(not created yet)",
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(got, s) {
			t.Errorf("output missing %q\n---\n%s\n---", s, got)
		}
	}

	if strings.Contains(got, "```md") {
		t.Errorf("expected no fenced block when entrypoint missing, got:\n%s", got)
	}
}

func TestLoadMemoryPrompt_WithEntrypoint(t *testing.T) {
	setDataDirEnv(t)

	projectDir := t.TempDir()

	entry, err := GetMemoryEntrypoint(projectDir)
	if err != nil {
		t.Fatalf("GetMemoryEntrypoint: %v", err)
	}

	body := "# Memory Index\n- [Coding style](coding_style.md)\n- [Deploy process](deploy_process.md)\n"
	if err := os.WriteFile(entry, []byte(body), 0o644); err != nil {
		t.Fatalf("write entrypoint: %v", err)
	}

	got, err := LoadMemoryPrompt(projectDir, 0)
	if err != nil {
		t.Fatalf("LoadMemoryPrompt: %v", err)
	}

	memoryDir := filepath.Dir(entry)
	wantSubstrings := []string{
		"# Memory",
		"- Persistent memory directory: " + memoryDir,
		"## MEMORY.md",
		"```md",
		"# Memory Index",
		"- [Coding style](coding_style.md)",
		"- [Deploy process](deploy_process.md)",
		"```",
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(got, s) {
			t.Errorf("output missing %q\n---\n%s\n---", s, got)
		}
	}

	if strings.Contains(got, "(not created yet)") {
		t.Errorf("placeholder should not appear when entrypoint exists, got:\n%s", got)
	}

	// The fenced block must close after the inlined content, and the trailing
	// empty line from body should have been stripped.
	if !strings.HasSuffix(got, "\n```") {
		t.Errorf("expected prompt to end with closing fence, got:\n%s", got)
	}
}

func TestLoadMemoryPrompt_TruncatesToMaxLines(t *testing.T) {
	setDataDirEnv(t)

	projectDir := t.TempDir()

	entry, err := GetMemoryEntrypoint(projectDir)
	if err != nil {
		t.Fatalf("GetMemoryEntrypoint: %v", err)
	}

	var b strings.Builder
	for i := 0; i < 10; i++ {
		b.WriteString("line")
		b.WriteString(string(rune('0' + i)))
		b.WriteString("\n")
	}
	if err := os.WriteFile(entry, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write entrypoint: %v", err)
	}

	got, err := LoadMemoryPrompt(projectDir, 3)
	if err != nil {
		t.Fatalf("LoadMemoryPrompt: %v", err)
	}

	for i := 0; i < 3; i++ {
		want := "line" + string(rune('0'+i))
		if !strings.Contains(got, want) {
			t.Errorf("expected truncated output to contain %q, got:\n%s", want, got)
		}
	}
	for i := 3; i < 10; i++ {
		notWant := "line" + string(rune('0'+i))
		if strings.Contains(got, notWant) {
			t.Errorf("expected truncated output to omit %q, got:\n%s", notWant, got)
		}
	}
}

func TestLoadMemoryPrompt_EmptyEntrypointOmitsFence(t *testing.T) {
	setDataDirEnv(t)

	projectDir := t.TempDir()

	entry, err := GetMemoryEntrypoint(projectDir)
	if err != nil {
		t.Fatalf("GetMemoryEntrypoint: %v", err)
	}
	if err := os.WriteFile(entry, []byte(""), 0o644); err != nil {
		t.Fatalf("write entrypoint: %v", err)
	}

	got, err := LoadMemoryPrompt(projectDir, 0)
	if err != nil {
		t.Fatalf("LoadMemoryPrompt: %v", err)
	}

	if strings.Contains(got, "```md") {
		t.Errorf("expected no fenced block for empty entrypoint, got:\n%s", got)
	}
	if strings.Contains(got, "(not created yet)") {
		t.Errorf("should not render placeholder when file exists, got:\n%s", got)
	}
	if !strings.HasPrefix(got, "# Memory\n") {
		t.Errorf("expected prompt to start with header, got:\n%s", got)
	}
}

func TestLoadMemoryPrompt_DefaultMaxLinesWhenNonPositive(t *testing.T) {
	setDataDirEnv(t)

	projectDir := t.TempDir()

	entry, err := GetMemoryEntrypoint(projectDir)
	if err != nil {
		t.Fatalf("GetMemoryEntrypoint: %v", err)
	}

	total := DefaultMaxEntrypointLines + 50
	var b strings.Builder
	for i := 0; i < total; i++ {
		b.WriteString("row\n")
	}
	if err := os.WriteFile(entry, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write entrypoint: %v", err)
	}

	got, err := LoadMemoryPrompt(projectDir, -1)
	if err != nil {
		t.Fatalf("LoadMemoryPrompt: %v", err)
	}

	// DefaultMaxEntrypointLines "row" lines should be present, plus the
	// surrounding header/fence metadata.
	if n := strings.Count(got, "row"); n != DefaultMaxEntrypointLines {
		t.Errorf("expected %d row lines under default cap, got %d", DefaultMaxEntrypointLines, n)
	}
}

func TestLoadMemoryPrompt_EmptyCwdReturnsError(t *testing.T) {
	setDataDirEnv(t)

	if _, err := LoadMemoryPrompt("", 0); err == nil {
		t.Fatal("expected error for empty cwd, got nil")
	}
}

func TestSplitLines(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty", in: "", want: nil},
		{name: "only newlines", in: "\n\n", want: []string{"", ""}},
		{name: "trailing newline stripped", in: "a\nb\n", want: []string{"a", "b"}},
		{name: "no trailing newline", in: "a\nb", want: []string{"a", "b"}},
		{name: "crlf normalised", in: "a\r\nb\r\n", want: []string{"a", "b"}},
		{name: "lone cr normalised", in: "a\rb", want: []string{"a", "b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitLines(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("len mismatch: got %v want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("idx %d: got %q want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
