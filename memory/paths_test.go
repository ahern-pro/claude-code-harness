package memory

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setDataDirEnv points OPENHARNESS_DATA_DIR at a temp dir for the duration of a
// test. It restores any previous value on cleanup.
func setDataDirEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("OPENHARNESS_DATA_DIR", dir)
	// Also isolate the config dir so tests never touch the user home.
	t.Setenv("OPENHARNESS_CONFIG_DIR", t.TempDir())
	return dir
}

func expectedDigest(t *testing.T, absPath string) string {
	t.Helper()
	sum := sha1.Sum([]byte(absPath))
	return hex.EncodeToString(sum[:])[:12]
}

func TestGetProjectMemoryDir_CreatesDeterministicPath(t *testing.T) {
	dataDir := setDataDirEnv(t)

	projectDir := t.TempDir()
	absProject, err := filepath.Abs(projectDir)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if resolved, err := filepath.EvalSymlinks(absProject); err == nil {
		absProject = resolved
	}

	got, err := GetProjectMemoryDir(projectDir)
	if err != nil {
		t.Fatalf("GetProjectMemoryDir: %v", err)
	}

	digest := expectedDigest(t, absProject)
	want := filepath.Join(dataDir, "memory", fmt.Sprintf("%s-%s", filepath.Base(absProject), digest))
	if got != want {
		t.Errorf("path mismatch:\n got:  %s\n want: %s", got, want)
	}

	info, err := os.Stat(got)
	if err != nil {
		t.Fatalf("stat memory dir: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("memory path is not a directory: %s", got)
	}
}

func TestGetProjectMemoryDir_IdempotentForSameCwd(t *testing.T) {
	setDataDirEnv(t)

	projectDir := t.TempDir()

	first, err := GetProjectMemoryDir(projectDir)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	second, err := GetProjectMemoryDir(projectDir)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if first != second {
		t.Errorf("expected identical paths across calls, got %q and %q", first, second)
	}
}

func TestGetProjectMemoryDir_DifferentProjectsGetDifferentDirs(t *testing.T) {
	setDataDirEnv(t)

	a := t.TempDir()
	b := t.TempDir()

	dirA, err := GetProjectMemoryDir(a)
	if err != nil {
		t.Fatalf("project a: %v", err)
	}
	dirB, err := GetProjectMemoryDir(b)
	if err != nil {
		t.Fatalf("project b: %v", err)
	}

	if dirA == dirB {
		t.Errorf("expected distinct memory dirs for distinct projects, both = %s", dirA)
	}
}

func TestGetProjectMemoryDir_ResolvesRelativePath(t *testing.T) {
	setDataDirEnv(t)

	projectDir := t.TempDir()
	parent := filepath.Dir(projectDir)
	base := filepath.Base(projectDir)

	// Change into parent and pass the bare base name as cwd.
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(parent); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	fromRelative, err := GetProjectMemoryDir(base)
	if err != nil {
		t.Fatalf("relative: %v", err)
	}
	fromAbsolute, err := GetProjectMemoryDir(projectDir)
	if err != nil {
		t.Fatalf("absolute: %v", err)
	}

	if fromRelative != fromAbsolute {
		t.Errorf("relative and absolute cwd must resolve to same dir:\n rel: %s\n abs: %s", fromRelative, fromAbsolute)
	}
}

func TestGetProjectMemoryDir_EmptyCwdReturnsError(t *testing.T) {
	setDataDirEnv(t)

	if _, err := GetProjectMemoryDir(""); err == nil {
		t.Fatal("expected error for empty cwd, got nil")
	}
}

func TestGetMemoryEntrypoint_PointsAtMemoryMd(t *testing.T) {
	setDataDirEnv(t)

	projectDir := t.TempDir()

	entry, err := GetMemoryEntrypoint(projectDir)
	if err != nil {
		t.Fatalf("GetMemoryEntrypoint: %v", err)
	}
	if filepath.Base(entry) != "MEMORY.md" {
		t.Errorf("expected entrypoint filename MEMORY.md, got %s", filepath.Base(entry))
	}

	dir, err := GetProjectMemoryDir(projectDir)
	if err != nil {
		t.Fatalf("GetProjectMemoryDir: %v", err)
	}
	if filepath.Dir(entry) != dir {
		t.Errorf("entrypoint should live inside project memory dir:\n entry dir: %s\n memory dir: %s", filepath.Dir(entry), dir)
	}

	// Ensure the parent dir exists even though the file itself is not written.
	if _, err := os.Stat(filepath.Dir(entry)); err != nil {
		t.Errorf("memory dir should exist: %v", err)
	}
	if _, err := os.Stat(entry); !os.IsNotExist(err) {
		t.Errorf("entrypoint file should not be created by GetMemoryEntrypoint, stat err=%v", err)
	}
}

func TestGetProjectMemoryDir_UsesDataDirEnvOverride(t *testing.T) {
	dataDir := setDataDirEnv(t)

	projectDir := t.TempDir()

	got, err := GetProjectMemoryDir(projectDir)
	if err != nil {
		t.Fatalf("GetProjectMemoryDir: %v", err)
	}
	if !strings.HasPrefix(got, dataDir+string(filepath.Separator)) {
		t.Errorf("memory dir should live under OPENHARNESS_DATA_DIR=%s, got %s", dataDir, got)
	}
}
