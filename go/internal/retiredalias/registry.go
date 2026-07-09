package retiredalias

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	RegistryRelPath = "templates/registry/retired-aliases.v1.yaml"
	SchemaRelPath   = "templates/schemas/retired-alias.v1.json"
)

// Kind classifies a retired alias entry.
type Kind string

const (
	KindPath    Kind = "path"
	KindConcept Kind = "concept"
	KindCommand Kind = "command"
	KindSkill   Kind = "skill"
)

// Entry is one retired alias registry row.
type Entry struct {
	ID        string   `yaml:"id" json:"id"`
	Kind      Kind     `yaml:"kind" json:"kind"`
	Pattern   string   `yaml:"pattern" json:"pattern"`
	RemovedIn string   `yaml:"removed_in,omitempty" json:"removed_in,omitempty"`
	Reason    string   `yaml:"reason,omitempty" json:"reason,omitempty"`
	Allowlist []string `yaml:"allowlist,omitempty" json:"allowlist,omitempty"`
}

// Registry is the retired-aliases.v1 document.
type Registry struct {
	Version int     `yaml:"version" json:"version"`
	Entries []Entry `yaml:"entries" json:"entries"`
}

// Hit is one pattern match reported by Scan.
type Hit struct {
	EntryID string
	Kind    Kind
	Pattern string
	File    string
	Line    int
	Snippet string
}

func (h Hit) String() string {
	return fmt.Sprintf("%s (%s) %s:%d — %q", h.EntryID, h.Kind, h.File, h.Line, h.Snippet)
}

// DefaultRegistryPath returns the canonical registry path under repoRoot.
func DefaultRegistryPath(repoRoot string) string {
	return filepath.Join(repoRoot, RegistryRelPath)
}

// DefaultSchemaPath returns the canonical schema path under repoRoot.
func DefaultSchemaPath(repoRoot string) string {
	return filepath.Join(repoRoot, SchemaRelPath)
}

// LoadRegistry reads and parses a retired-aliases YAML file.
func LoadRegistry(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read registry %s: %w", path, err)
	}
	reg, err := ParseRegistryYAML(data)
	if err != nil {
		return nil, err
	}
	if reg.Version != 1 {
		return nil, fmt.Errorf("unsupported registry version %d", reg.Version)
	}
	if len(reg.Entries) == 0 {
		return nil, fmt.Errorf("registry has no entries")
	}
	return reg, nil
}
