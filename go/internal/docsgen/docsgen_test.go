package docsgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSkill creates <root>/skills/<name>/SKILL.md with the given frontmatter body.
func writeSkill(t *testing.T, root, name, frontmatter string) {
	t.Helper()
	dir := filepath.Join(root, "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(frontmatter), 0o644); err != nil {
		t.Fatalf("write SKILL.md for %s: %v", name, err)
	}
}

// newFixtureRoot builds a synthetic repo root with a skills/ tree and a catalog
// file that already contains the BEGIN/END markers (empty managed region).
func newFixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	writeSkill(t, root, "beta", "---\nname: beta\ndescription: \"Beta skill. Does B things.\"\ndescription-ja: \"ベータ\"\n---\n\n# Beta\n")
	writeSkill(t, root, "alpha", "---\nname: alpha\ndescription: \"Alpha skill: handles A | also pipes.\"\n---\n\n# Alpha\n")
	// A directory without SKILL.md must be ignored (not a skill).
	if err := os.MkdirAll(filepath.Join(root, "skills", "notaskill"), 0o755); err != nil {
		t.Fatalf("mkdir notaskill: %v", err)
	}
	// Dev/experimental skills are excluded from the distributed catalog.
	writeSkill(t, root, "test-foo", "---\nname: test-foo\ndescription: \"dev only\"\n---\n")
	writeSkill(t, root, "x-promo", "---\nname: x-promo\ndescription: \"dev only\"\n---\n")

	catalogDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	catalog := "# Catalog\n\nIntro prose stays.\n\n" +
		CatalogBeginMarker + "\n" + CatalogEndMarker + "\n\n## Tail prose stays\n"
	if err := os.WriteFile(filepath.Join(catalogDir, "CLAUDE-skill-catalog.md"), []byte(catalog), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
	return root
}

func TestCollectSkills_SortedAndFiltered(t *testing.T) {
	root := newFixtureRoot(t)
	skills, err := CollectSkills(root)
	if err != nil {
		t.Fatalf("CollectSkills: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills (alpha, beta), got %d: %+v", len(skills), skills)
	}
	if skills[0].Name != "alpha" || skills[1].Name != "beta" {
		t.Errorf("skills not sorted by name: %+v", skills)
	}
	if skills[0].Description != "Alpha skill: handles A | also pipes." {
		t.Errorf("alpha description parsed wrong: %q", skills[0].Description)
	}
	if skills[1].Description != "Beta skill. Does B things." {
		t.Errorf("beta description parsed wrong: %q", skills[1].Description)
	}
}

func TestRenderCatalog_EscapesPipesAndIsDeterministic(t *testing.T) {
	skills := []Skill{
		{Name: "alpha", Description: "Has a | pipe"},
		{Name: "beta", Description: "Plain"},
	}
	got1 := RenderCatalog(skills)
	got2 := RenderCatalog(skills)
	if got1 != got2 {
		t.Fatal("RenderCatalog is not deterministic")
	}
	if !strings.Contains(got1, "| alpha | Has a \\| pipe |") {
		t.Errorf("pipe not escaped in table cell:\n%s", got1)
	}
	if !strings.HasPrefix(got1, CatalogBeginMarker) {
		t.Errorf("rendered block must start with begin marker:\n%s", got1)
	}
	if !strings.Contains(got1, CatalogEndMarker) {
		t.Errorf("rendered block must contain end marker:\n%s", got1)
	}
}

func TestReplaceManagedRegion_PreservesSurroundingProse(t *testing.T) {
	content := "HEAD\n" + CatalogBeginMarker + "\nOLD\n" + CatalogEndMarker + "\nTAIL\n"
	block := CatalogBeginMarker + "\nNEW\n" + CatalogEndMarker + "\n"
	out, err := ReplaceManagedRegion(content, block)
	if err != nil {
		t.Fatalf("ReplaceManagedRegion: %v", err)
	}
	want := "HEAD\n" + CatalogBeginMarker + "\nNEW\n" + CatalogEndMarker + "\nTAIL\n"
	if out != want {
		t.Errorf("region replacement wrong:\n got: %q\nwant: %q", out, want)
	}
}

func TestReplaceManagedRegion_MissingMarkerErrors(t *testing.T) {
	if _, err := ReplaceManagedRegion("no markers here", "x"); err == nil {
		t.Error("expected error when begin marker is absent")
	}
	reversed := CatalogEndMarker + "\n" + CatalogBeginMarker + "\n"
	if _, err := ReplaceManagedRegion(reversed, "x"); err == nil {
		t.Error("expected error when end marker precedes begin marker")
	}
}

func TestGenerateWriteCheck_RoundTrip(t *testing.T) {
	root := newFixtureRoot(t)

	// Initially the committed catalog has an empty managed region → out of sync.
	inSync, _, err := Check(root)
	if err != nil {
		t.Fatalf("Check (pre-write): %v", err)
	}
	if inSync {
		t.Fatal("expected catalog to be out of sync before first Write")
	}

	changed, err := Write(root)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !changed {
		t.Fatal("expected Write to report a change on first run")
	}

	// After Write, --check must pass and a second Write must be a no-op.
	inSync, diff, err := Check(root)
	if err != nil {
		t.Fatalf("Check (post-write): %v", err)
	}
	if !inSync {
		t.Fatalf("expected catalog in sync after Write; diff:\n%s", diff)
	}
	changed, err = Write(root)
	if err != nil {
		t.Fatalf("Write (idempotent): %v", err)
	}
	if changed {
		t.Error("expected second Write to be a no-op")
	}

	// The generated catalog must list both real skills and preserve prose.
	data, err := os.ReadFile(filepath.Join(root, "docs", "CLAUDE-skill-catalog.md"))
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}
	text := string(data)
	for _, want := range []string{"Intro prose stays.", "## Tail prose stays", "| alpha |", "| beta |"} {
		if !strings.Contains(text, want) {
			t.Errorf("generated catalog missing %q:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"test-foo", "x-promo", "notaskill"} {
		if strings.Contains(text, unwanted) {
			t.Errorf("generated catalog must not list excluded entry %q", unwanted)
		}
	}
}

func TestCheck_DetectsDrift(t *testing.T) {
	root := newFixtureRoot(t)
	if _, err := Write(root); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// Add a new skill after generation → catalog should now be detected as drifted.
	writeSkill(t, root, "gamma", "---\nname: gamma\ndescription: \"Gamma.\"\n---\n")
	inSync, diff, err := Check(root)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if inSync {
		t.Error("expected drift after adding a new skill")
	}
	if !strings.Contains(diff, "gamma") {
		t.Errorf("drift diff should mention the new skill gamma:\n%s", diff)
	}
}

func TestParseFrontmatter_QuotedAndEscaped(t *testing.T) {
	body := "---\nname: demo\ndescription: \"Says \\\"hi\\\" and a back\\\\slash.\"\nallowed-tools: [\"Read\"]\n---\nbody\n"
	fm, err := parseFrontmatter([]byte(body))
	if err != nil {
		t.Fatalf("parseFrontmatter: %v", err)
	}
	if fm["name"] != "demo" {
		t.Errorf("name parsed wrong: %q", fm["name"])
	}
	if fm["description"] != `Says "hi" and a back\slash.` {
		t.Errorf("description unescaped wrong: %q", fm["description"])
	}
}

func TestParseFrontmatter_NoFrontmatterReturnsEmpty(t *testing.T) {
	fm, err := parseFrontmatter([]byte("# just a heading\n"))
	if err != nil {
		t.Fatalf("parseFrontmatter: %v", err)
	}
	if len(fm) != 0 {
		t.Errorf("expected empty map for file without frontmatter, got %+v", fm)
	}
}

func TestParseFrontmatter_UnterminatedErrors(t *testing.T) {
	if _, err := parseFrontmatter([]byte("---\nname: x\n")); err == nil {
		t.Error("expected error for unterminated frontmatter")
	}
}
