package skills

import (
	"strings"
	"testing"
)

func TestParseFrontmatter_YAMLWithNameAndDescription(t *testing.T) {
	content := strings.Join([]string{
		"---",
		"name: custom-name",
		"description: A custom description",
		"---",
		"",
		"# heading",
		"",
		"Some body text.",
	}, "\n")

	name, description := parseFrontmatter("default", content)
	if name != "custom-name" {
		t.Errorf("name: got %q, want %q", name, "custom-name")
	}
	if description != "A custom description" {
		t.Errorf("description: got %q, want %q", description, "A custom description")
	}
}

func TestParseFrontmatter_YAMLStripsQuotes(t *testing.T) {
	content := strings.Join([]string{
		"---",
		"name: \"quoted-name\"",
		"description: 'quoted description'",
		"---",
		"",
	}, "\n")

	name, description := parseFrontmatter("default", content)
	if name != "quoted-name" {
		t.Errorf("name: got %q, want %q", name, "quoted-name")
	}
	if description != "quoted description" {
		t.Errorf("description: got %q, want %q", description, "quoted description")
	}
}

func TestParseFrontmatter_YAMLMissingDescriptionFallsBack(t *testing.T) {
	content := strings.Join([]string{
		"---",
		"name: from-frontmatter",
		"---",
		"",
		"# heading-name",
		"",
		"Paragraph body serves as description.",
	}, "\n")

	name, description := parseFrontmatter("default", content)
	if name != "heading-name" {
		t.Errorf("name: got %q, want %q (fallback should overwrite frontmatter name)", name, "heading-name")
	}
	if description != "Paragraph body serves as description." {
		t.Errorf("description: got %q", description)
	}
}

func TestParseFrontmatter_HeadingAndParagraph(t *testing.T) {
	content := strings.Join([]string{
		"# my-skill",
		"",
		"This is the first paragraph used as description.",
		"",
		"## Next section",
		"",
		"More content here.",
	}, "\n")

	name, description := parseFrontmatter("default", content)
	if name != "my-skill" {
		t.Errorf("name: got %q, want %q", name, "my-skill")
	}
	expected := "This is the first paragraph used as description."
	if description != expected {
		t.Errorf("description: got %q, want %q", description, expected)
	}
}

func TestParseFrontmatter_EmptyHeadingUsesDefault(t *testing.T) {
	content := strings.Join([]string{
		"# ",
		"",
		"Body text.",
	}, "\n")

	name, _ := parseFrontmatter("default-name", content)
	if name != "default-name" {
		t.Errorf("name: got %q, want %q", name, "default-name")
	}
}

func TestParseFrontmatter_NoHeadingNoParagraph(t *testing.T) {
	name, description := parseFrontmatter("lonely", "")
	if name != "lonely" {
		t.Errorf("name: got %q, want %q", name, "lonely")
	}
	if description != "Bundled skill: lonely" {
		t.Errorf("description: got %q, want %q", description, "Bundled skill: lonely")
	}
}

func TestParseFrontmatter_TruncatesLongParagraph(t *testing.T) {
	long := strings.Repeat("a", 250)
	content := "# name\n\n" + long

	_, description := parseFrontmatter("default", content)
	if len(description) != descriptionMaxLen {
		t.Errorf("description length: got %d, want %d", len(description), descriptionMaxLen)
	}
	if description != strings.Repeat("a", descriptionMaxLen) {
		t.Errorf("description content mismatch")
	}
}

func TestParseFrontmatter_SkipsHeadingsInFallback(t *testing.T) {
	content := strings.Join([]string{
		"# name",
		"",
		"## sub-heading",
		"",
		"First real paragraph.",
	}, "\n")

	_, description := parseFrontmatter("default", content)
	if description != "First real paragraph." {
		t.Errorf("description: got %q, want %q", description, "First real paragraph.")
	}
}

func TestGetBundledSkills_LoadsContentDirectory(t *testing.T) {
	result := GetBundledSkills()
	if len(result) == 0 {
		t.Fatal("expected at least one bundled skill, got zero")
	}

	seen := make(map[string]struct{})
	for _, skill := range result {
		if skill.Name == "" {
			t.Errorf("skill with empty name: %+v", skill)
		}
		if skill.Description == "" {
			t.Errorf("skill %q has empty description", skill.Name)
		}
		if skill.Content == "" {
			t.Errorf("skill %q has empty content", skill.Name)
		}
		if skill.Source != sourceBundled {
			t.Errorf("skill %q has source %q, want %q", skill.Name, skill.Source, sourceBundled)
		}
		if !strings.HasPrefix(skill.Path, contentDir+"/") {
			t.Errorf("skill %q has unexpected path %q", skill.Name, skill.Path)
		}
		if !strings.HasSuffix(skill.Path, ".md") {
			t.Errorf("skill %q path does not end in .md: %q", skill.Name, skill.Path)
		}
		if _, dup := seen[skill.Path]; dup {
			t.Errorf("duplicate skill path: %q", skill.Path)
		}
		seen[skill.Path] = struct{}{}
	}
}

func TestGetBundledSkills_ReturnsSortedByPath(t *testing.T) {
	result := GetBundledSkills()
	if len(result) < 2 {
		t.Skip("need at least two bundled skills to test ordering")
	}
	for i := 1; i < len(result); i++ {
		if result[i-1].Path >= result[i].Path {
			t.Errorf("skills not sorted: %q should precede %q", result[i].Path, result[i-1].Path)
		}
	}
}

func TestGetBundledSkills_DescriptionsWithinLimit(t *testing.T) {
	for _, skill := range GetBundledSkills() {
		if len(skill.Description) > descriptionMaxLen {
			t.Errorf("skill %q description exceeds %d chars: got %d",
				skill.Name, descriptionMaxLen, len(skill.Description))
		}
	}
}

func TestStripQuotes(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`"hello"`, "hello"},
		{`'hello'`, "hello"},
		{`hello`, "hello"},
		{`"`, `"`},
		{``, ``},
		{`"mismatch'`, `"mismatch'`},
	}
	for _, tc := range cases {
		if got := stripQuotes(tc.in); got != tc.want {
			t.Errorf("stripQuotes(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
