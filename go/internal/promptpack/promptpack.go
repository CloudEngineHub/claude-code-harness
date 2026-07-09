// Package promptpack embeds the verb prompt contracts that the host LLM
// (Claude / Codex / Cursor) follows when running a harness verb.
//
// These prompts are the single source of truth for each verb's irreducible
// contract. They are embedded into the binary so `harness work|plan|review|
// release` can assemble a host-facing prompt without reading external files.
package promptpack

import (
	"embed"
	"fmt"
)

//go:embed prompts/*.md
var fs embed.FS

// Get returns the embedded prompt for verb ("work"|"plan"|"review"|"release").
func Get(verb string) (string, error) {
	b, err := fs.ReadFile("prompts/" + verb + ".md")
	if err != nil {
		return "", fmt.Errorf("promptpack: unknown verb %q", verb)
	}
	return string(b), nil
}

// Verbs lists the available prompt verbs.
func Verbs() []string { return []string{"work", "plan", "review", "release"} }
