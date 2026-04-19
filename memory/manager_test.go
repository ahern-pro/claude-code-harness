package memory

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestListMemoryFiles_EmptyWhenNothingWritten(t *testing.T) {
	setDataDirEnv(t)
	project := t.TempDir()

	got, err := ListMemoryFiles(project)
	if err != nil {
		t.Fatalf("ListMemoryFiles: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no memory files, got %v", got)
	}
}

func TestListMemoryFiles_ReturnsSortedMarkdownFiles(t *testing.T) {
	setDataDirEnv(t)
	project := t.TempDir()

	dir, err := GetProjectMemoryDir(project)
	if err != nil {
		t.Fatalf("GetProjectMemoryDir: %v", err)
	}
	// Write files in a deliberately unsorted order and include a non-md file
	// that should be ignored.
	names := []string{"zeta.md", "alpha.md", "mike.md"}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatalf("seed %s: %v", n, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seed ignore: %v", err)
	}

	got, err := ListMemoryFiles(project)
	if err != nil {
		t.Fatalf("ListMemoryFiles: %v", err)
	}
	want := []string{
		filepath.Join(dir, "alpha.md"),
		filepath.Join(dir, "mike.md"),
		filepath.Join(dir, "zeta.md"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected listing:\n got:  %v\n want: %v", got, want)
	}
}

func TestAddMemoryEntry_WritesFileAndIndex(t *testing.T) {
	setDataDirEnv(t)
	project := t.TempDir()

	path, err := AddMemoryEntry(project, "Project Context", "  first note  ")
	if err != nil {
		t.Fatalf("AddMemoryEntry: %v", err)
	}
	if filepath.Base(path) != "project_context.md" {
		t.Errorf("expected slugified filename project_context.md, got %s", filepath.Base(path))
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read memory file: %v", err)
	}
	if string(body) != "first note\n" {
		t.Errorf("unexpected memory body: %q", string(body))
	}

	entrypoint, err := GetMemoryEntrypoint(project)
	if err != nil {
		t.Fatalf("GetMemoryEntrypoint: %v", err)
	}
	index, err := os.ReadFile(entrypoint)
	if err != nil {
		t.Fatalf("read entrypoint: %v", err)
	}
	got := string(index)
	if !strings.HasPrefix(got, "# Memory Index") {
		t.Errorf("expected index to start with header, got %q", got)
	}
	wantLink := "- [Project Context](project_context.md)"
	if !strings.Contains(got, wantLink) {
		t.Errorf("expected index to contain %q, got %q", wantLink, got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("expected index to end with newline, got %q", got)
	}
}

func TestAddMemoryEntry_IsIdempotentForSameTitle(t *testing.T) {
	setDataDirEnv(t)
	project := t.TempDir()

	first, err := AddMemoryEntry(project, "Notes", "one")
	if err != nil {
		t.Fatalf("first add: %v", err)
	}
	second, err := AddMemoryEntry(project, "Notes", "two")
	if err != nil {
		t.Fatalf("second add: %v", err)
	}
	if first != second {
		t.Errorf("expected same path across calls, got %q and %q", first, second)
	}

	body, err := os.ReadFile(second)
	if err != nil {
		t.Fatalf("read memory file: %v", err)
	}
	if string(body) != "two\n" {
		t.Errorf("expected second write to overwrite body, got %q", string(body))
	}

	entrypoint, err := GetMemoryEntrypoint(project)
	if err != nil {
		t.Fatalf("GetMemoryEntrypoint: %v", err)
	}
	index, err := os.ReadFile(entrypoint)
	if err != nil {
		t.Fatalf("read entrypoint: %v", err)
	}
	if occurrences := strings.Count(string(index), "notes.md"); occurrences != 1 {
		t.Errorf("expected index to reference notes.md exactly once, got %d\nindex:\n%s", occurrences, index)
	}
}

func TestAddMemoryEntry_EmptyTitleFallsBackToMemory(t *testing.T) {
	setDataDirEnv(t)
	project := t.TempDir()

	path, err := AddMemoryEntry(project, "   ", "body")
	if err != nil {
		t.Fatalf("AddMemoryEntry: %v", err)
	}
	if filepath.Base(path) != "memory.md" {
		t.Errorf("expected fallback filename memory.md, got %s", filepath.Base(path))
	}
}

func TestAddMemoryEntry_SlugifiesSymbolsAndCase(t *testing.T) {
	setDataDirEnv(t)
	project := t.TempDir()

	path, err := AddMemoryEntry(project, "--Hello, World!--", "note")
	if err != nil {
		t.Fatalf("AddMemoryEntry: %v", err)
	}
	if got, want := filepath.Base(path), "hello_world.md"; got != want {
		t.Errorf("slug mismatch: got %s, want %s", got, want)
	}
}

func TestRemoveMemoryEntry_ByStemAndFilename(t *testing.T) {
	setDataDirEnv(t)
	project := t.TempDir()

	if _, err := AddMemoryEntry(project, "Alpha", "a"); err != nil {
		t.Fatalf("seed alpha: %v", err)
	}
	if _, err := AddMemoryEntry(project, "Bravo", "b"); err != nil {
		t.Fatalf("seed bravo: %v", err)
	}

	removed, err := RemoveMemoryEntry(project, "alpha")
	if err != nil {
		t.Fatalf("remove by stem: %v", err)
	}
	if !removed {
		t.Fatal("expected removal by stem to succeed")
	}

	removed, err = RemoveMemoryEntry(project, "bravo.md")
	if err != nil {
		t.Fatalf("remove by filename: %v", err)
	}
	if !removed {
		t.Fatal("expected removal by filename to succeed")
	}

	files, err := ListMemoryFiles(project)
	if err != nil {
		t.Fatalf("ListMemoryFiles: %v", err)
	}
	// Python parity: MEMORY.md itself is a .md file so remains in the listing.
	// What matters is that the per-entry files are gone.
	for _, f := range files {
		base := filepath.Base(f)
		if base != "MEMORY.md" {
			t.Errorf("unexpected leftover memory file after removals: %s", f)
		}
	}

	entrypoint, err := GetMemoryEntrypoint(project)
	if err != nil {
		t.Fatalf("GetMemoryEntrypoint: %v", err)
	}
	index, err := os.ReadFile(entrypoint)
	if err != nil {
		t.Fatalf("read entrypoint: %v", err)
	}
	if strings.Contains(string(index), "alpha.md") || strings.Contains(string(index), "bravo.md") {
		t.Errorf("index still references removed files:\n%s", index)
	}
}

func TestRemoveMemoryEntry_MissingReturnsFalse(t *testing.T) {
	setDataDirEnv(t)
	project := t.TempDir()

	removed, err := RemoveMemoryEntry(project, "ghost")
	if err != nil {
		t.Fatalf("RemoveMemoryEntry: %v", err)
	}
	if removed {
		t.Fatal("expected false when no matching file exists")
	}
}

func TestRemoveMemoryEntry_LeavesUnrelatedIndexLinesIntact(t *testing.T) {
	setDataDirEnv(t)
	project := t.TempDir()

	if _, err := AddMemoryEntry(project, "Keep", "k"); err != nil {
		t.Fatalf("seed keep: %v", err)
	}
	if _, err := AddMemoryEntry(project, "Drop", "d"); err != nil {
		t.Fatalf("seed drop: %v", err)
	}

	if _, err := RemoveMemoryEntry(project, "drop"); err != nil {
		t.Fatalf("RemoveMemoryEntry: %v", err)
	}

	entrypoint, err := GetMemoryEntrypoint(project)
	if err != nil {
		t.Fatalf("GetMemoryEntrypoint: %v", err)
	}
	index, err := os.ReadFile(entrypoint)
	if err != nil {
		t.Fatalf("read entrypoint: %v", err)
	}
	got := string(index)
	if !strings.Contains(got, "- [Keep](keep.md)") {
		t.Errorf("expected unrelated index line to remain, got %q", got)
	}
	if strings.Contains(got, "drop.md") {
		t.Errorf("expected dropped entry to be removed, got %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("expected trailing newline, got %q", got)
	}
}
