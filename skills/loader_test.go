package skills

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// writeSkill creates <root>/<dir>/SKILL.md with the given body.
func writeSkill(t *testing.T, root, dir, body string) string {
	t.Helper()
	skillDir := filepath.Join(root, dir)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", skillDir, err)
	}
	path := filepath.Join(skillDir, skillFilename)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
	return path
}

func TestGetUserSkillsDir_CreatesUnderConfigDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OPENHARNESS_CONFIG_DIR", tmp)

	dir, err := GetUserSkillsDir()
	if err != nil {
		t.Fatalf("GetUserSkillsDir: %v", err)
	}
	want := filepath.Join(tmp, userSkillsSubdir)
	if dir != want {
		t.Errorf("dir: got %q, want %q", dir, want)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat %q: %v", dir, err)
	}
	if !info.IsDir() {
		t.Errorf("%q is not a directory", dir)
	}
}

func TestLoadSkillsFromDirs_EmptyInputReturnsNil(t *testing.T) {
	skills, err := LoadSkillsFromDirs(nil, sourceUser)
	if err != nil {
		t.Fatalf("LoadSkillsFromDirs(nil): %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected no skills, got %d", len(skills))
	}

	skills, err = LoadSkillsFromDirs([]string{}, sourceUser)
	if err != nil {
		t.Fatalf("LoadSkillsFromDirs([]): %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected no skills, got %d", len(skills))
	}
}

func TestLoadSkillsFromDirs_DefaultsSourceToUser(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "alpha", "# alpha\n\nThe alpha skill.\n")

	skills, err := LoadSkillsFromDirs([]string{root}, "")
	if err != nil {
		t.Fatalf("LoadSkillsFromDirs: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Source != sourceUser {
		t.Errorf("source: got %q, want %q", skills[0].Source, sourceUser)
	}
}

func TestLoadSkillsFromDirs_SkillMdLayout(t *testing.T) {
	root := t.TempDir()

	writeSkill(t, root, "alpha", strings.Join([]string{
		"---",
		"name: alpha-skill",
		"description: Alpha description",
		"---",
		"",
		"# ignored",
		"",
		"Body.",
	}, "\n"))
	writeSkill(t, root, "beta", "# beta\n\nBeta paragraph.\n")

	// Non-skill dirs should be ignored.
	if err := os.MkdirAll(filepath.Join(root, "no-skill"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Loose top-level .md should be ignored (only SKILL.md under a subdir counts).
	if err := os.WriteFile(filepath.Join(root, "stray.md"), []byte("# stray\n\nstray."), 0o644); err != nil {
		t.Fatal(err)
	}

	skills, err := LoadSkillsFromDirs([]string{root}, "custom")
	if err != nil {
		t.Fatalf("LoadSkillsFromDirs: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d: %+v", len(skills), skills)
	}

	byName := make(map[string]SkillDefinition, len(skills))
	for _, s := range skills {
		byName[s.Name] = s
	}

	alpha, ok := byName["alpha-skill"]
	if !ok {
		t.Fatalf("missing alpha-skill, got names: %v", names(skills))
	}
	if alpha.Description != "Alpha description" {
		t.Errorf("alpha description: got %q", alpha.Description)
	}
	if alpha.Source != "custom" {
		t.Errorf("alpha source: got %q, want %q", alpha.Source, "custom")
	}
	if !strings.HasSuffix(alpha.Path, filepath.Join("alpha", skillFilename)) {
		t.Errorf("alpha path: got %q", alpha.Path)
	}

	beta, ok := byName["beta"]
	if !ok {
		t.Fatalf("missing beta, got names: %v", names(skills))
	}
	if beta.Description != "Beta paragraph." {
		t.Errorf("beta description: got %q", beta.Description)
	}
}

func TestLoadSkillsFromDirs_SortedByPath(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "zeta", "# zeta\n\nz.\n")
	writeSkill(t, root, "alpha", "# alpha\n\na.\n")
	writeSkill(t, root, "middle", "# middle\n\nm.\n")

	skills, err := LoadSkillsFromDirs([]string{root}, sourceUser)
	if err != nil {
		t.Fatalf("LoadSkillsFromDirs: %v", err)
	}
	paths := make([]string, len(skills))
	for i, s := range skills {
		paths[i] = s.Path
	}
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	for i := range paths {
		if paths[i] != sorted[i] {
			t.Errorf("skills not sorted by path: got %v", paths)
			break
		}
	}
}

func TestLoadSkillsFromDirs_DeduplicatesAcrossDirs(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "alpha", "# alpha\n\nbody.\n")

	skills, err := LoadSkillsFromDirs([]string{root, root}, sourceUser)
	if err != nil {
		t.Fatalf("LoadSkillsFromDirs: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill after dedup, got %d", len(skills))
	}
}

func TestLoadSkillsFromDirs_MergesMultipleRoots(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	writeSkill(t, rootA, "alpha", "# alpha\n\na body.\n")
	writeSkill(t, rootB, "beta", "# beta\n\nb body.\n")

	skills, err := LoadSkillsFromDirs([]string{rootA, rootB}, sourceUser)
	if err != nil {
		t.Fatalf("LoadSkillsFromDirs: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}

	got := names(skills)
	sort.Strings(got)
	want := []string{"alpha", "beta"}
	if !equalStrings(got, want) {
		t.Errorf("names: got %v, want %v", got, want)
	}
}

func TestLoadSkillsFromDirs_CreatesMissingDirectory(t *testing.T) {
	base := t.TempDir()
	missing := filepath.Join(base, "does-not-exist")

	skills, err := LoadSkillsFromDirs([]string{missing}, sourceUser)
	if err != nil {
		t.Fatalf("LoadSkillsFromDirs: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}
	info, err := os.Stat(missing)
	if err != nil {
		t.Fatalf("stat %q: %v", missing, err)
	}
	if !info.IsDir() {
		t.Errorf("%q was not created as dir", missing)
	}
}

func TestLoadSkillsFromDirs_UserFallbackDescription(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "lonely", "")

	skills, err := LoadSkillsFromDirs([]string{root}, sourceUser)
	if err != nil {
		t.Fatalf("LoadSkillsFromDirs: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	want := userFallbackPrefix + "lonely"
	if skills[0].Description != want {
		t.Errorf("description: got %q, want %q", skills[0].Description, want)
	}
}

func TestLoadUserSkills_ReadsUserDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OPENHARNESS_CONFIG_DIR", tmp)

	userDir := filepath.Join(tmp, userSkillsSubdir)
	writeSkill(t, userDir, "my-skill", "# my-skill\n\nA user skill.\n")

	skills, err := LoadUserSkills()
	if err != nil {
		t.Fatalf("LoadUserSkills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 user skill, got %d", len(skills))
	}
	s := skills[0]
	if s.Name != "my-skill" {
		t.Errorf("name: got %q, want %q", s.Name, "my-skill")
	}
	if s.Source != sourceUser {
		t.Errorf("source: got %q, want %q", s.Source, sourceUser)
	}
	if s.Description != "A user skill." {
		t.Errorf("description: got %q", s.Description)
	}
}

func TestLoadUserSkills_EmptyWhenNoSkills(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OPENHARNESS_CONFIG_DIR", tmp)

	skills, err := LoadUserSkills()
	if err != nil {
		t.Fatalf("LoadUserSkills: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected no skills, got %d", len(skills))
	}
}

func TestLoadSkillRegistry_IncludesBundledAndUserSkills(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OPENHARNESS_CONFIG_DIR", tmp)

	userDir := filepath.Join(tmp, userSkillsSubdir)
	writeSkill(t, userDir, "user-only-skill", "# user-only-skill\n\nFrom user dir.\n")

	registry, err := LoadSkillRegistry()
	if err != nil {
		t.Fatalf("LoadSkillRegistry: %v", err)
	}

	bundled := GetBundledSkills()
	if len(bundled) == 0 {
		t.Fatal("no bundled skills found; cannot verify registry")
	}
	for _, b := range bundled {
		if got := registry.SkillDefinition(b.Name); got.Name == "" {
			t.Errorf("bundled skill %q missing from registry", b.Name)
		}
	}

	if got := registry.SkillDefinition("user-only-skill"); got.Name != "user-only-skill" {
		t.Errorf("user-only-skill missing from registry: got %+v", got)
	}
}

func TestLoadSkillRegistry_UserSkillOverridesBundled(t *testing.T) {
	bundled := GetBundledSkills()
	if len(bundled) == 0 {
		t.Skip("no bundled skills available for override test")
	}
	target := bundled[0]

	tmp := t.TempDir()
	t.Setenv("OPENHARNESS_CONFIG_DIR", tmp)

	userDir := filepath.Join(tmp, userSkillsSubdir)
	overrideBody := strings.Join([]string{
		"---",
		"name: " + target.Name,
		"description: Overridden by user",
		"---",
		"",
		"User-authored override body.",
	}, "\n")
	writeSkill(t, userDir, "override-dir", overrideBody)

	registry, err := LoadSkillRegistry()
	if err != nil {
		t.Fatalf("LoadSkillRegistry: %v", err)
	}
	got := registry.SkillDefinition(target.Name)
	if got.Source != sourceUser {
		t.Errorf("source: got %q, want %q", got.Source, sourceUser)
	}
	if got.Description != "Overridden by user" {
		t.Errorf("description: got %q, want %q", got.Description, "Overridden by user")
	}
}

func TestLoadSkillRegistry_WithExtraSkillDirs(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OPENHARNESS_CONFIG_DIR", tmp)

	extra := t.TempDir()
	writeSkill(t, extra, "plugin-skill", "# plugin-skill\n\nExtra directory skill.\n")

	registry, err := LoadSkillRegistry(WithExtraSkillDirs(extra))
	if err != nil {
		t.Fatalf("LoadSkillRegistry: %v", err)
	}

	got := registry.SkillDefinition("plugin-skill")
	if got.Name != "plugin-skill" {
		t.Fatalf("plugin-skill missing from registry")
	}
	if got.Description != "Extra directory skill." {
		t.Errorf("description: got %q", got.Description)
	}
	if got.Source != sourceUser {
		t.Errorf("source: got %q, want %q", got.Source, sourceUser)
	}
}

func TestLoadSkillRegistry_EachCallReturnsFreshRegistry(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OPENHARNESS_CONFIG_DIR", tmp)

	first, err := LoadSkillRegistry()
	if err != nil {
		t.Fatalf("first LoadSkillRegistry: %v", err)
	}
	second, err := LoadSkillRegistry()
	if err != nil {
		t.Fatalf("second LoadSkillRegistry: %v", err)
	}
	if first == second {
		t.Error("expected a fresh *SkillRegistry on each call")
	}
}

func TestExpandAndResolve_TildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot resolve home dir: %v", err)
	}

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"bare tilde", "~", home},
		{"tilde slash", "~/foo", filepath.Join(home, "foo")},
	}
	if runtime.GOOS == "windows" {
		t.Skip("tilde handling is POSIX-specific in these cases")
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := expandAndResolve(tc.in)
			if err != nil {
				t.Fatalf("expandAndResolve(%q): %v", tc.in, err)
			}
			wantAbs, _ := filepath.Abs(tc.want)
			if got != filepath.Clean(wantAbs) {
				t.Errorf("got %q, want %q", got, wantAbs)
			}
		})
	}
}

func TestExpandAndResolve_RelativePath(t *testing.T) {
	got, err := expandAndResolve("./relative/path")
	if err != nil {
		t.Fatalf("expandAndResolve: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("expected absolute path, got %q", got)
	}
}

// --- helpers ---

func names(skills []SkillDefinition) []string {
	out := make([]string, len(skills))
	for i, s := range skills {
		out[i] = s.Name
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
