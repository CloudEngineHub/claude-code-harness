// Package judgmentledger provides append-only JSONL storage for human judgment
// decisions with project-scoped search/recall (Phase 98.1).
package judgmentledger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Chachamaru127/claude-code-harness/go/internal/gitport"
	"github.com/google/uuid"
)

const (
	SchemaRelPath        = "templates/schemas/judgment-ledger.v1.json"
	DefaultLedgerRelPath = ".claude/state/judgment-ledger.jsonl"
)

// Record is one judgment-ledger.v1 JSONL line.
type Record struct {
	ID        string   `json:"id"`
	Project   string   `json:"project"`
	DecidedAt string   `json:"decided_at"`
	Question  string   `json:"question"`
	Answer    string   `json:"answer"`
	Rationale string   `json:"rationale"`
	CardRef   string   `json:"card_ref"`
	Tags      []string `json:"tags"`
}

// AppendOpts configures a ledger append.
type AppendOpts struct {
	LedgerPath string
	SchemaPath string
	Record     Record
}

// Append validates and appends one record. Returns error on schema or I/O failure.
func Append(opts AppendOpts) error {
	if opts.SchemaPath == "" {
		return fmt.Errorf("schema path required")
	}
	rec := opts.Record
	if rec.ID == "" {
		rec.ID = uuid.NewString()
	}
	if rec.DecidedAt == "" {
		rec.DecidedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if rec.Tags == nil {
		rec.Tags = []string{}
	}
	if err := ValidateRecord(rec, opts.SchemaPath); err != nil {
		return err
	}
	path := opts.LedgerPath
	if path == "" {
		return fmt.Errorf("ledger path required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

// AppendFailOpen appends one record without propagating errors (fail-open contract).
func AppendFailOpen(opts AppendOpts) {
	_ = Append(opts)
}

// LoadAll reads every valid record from a JSONL ledger file.
func LoadAll(path string) ([]Record, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []Record
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec Record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		out = append(out, rec)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// LoadByProject returns records scoped to project (exact match).
func LoadByProject(path, project string) ([]Record, error) {
	all, err := LoadAll(path)
	if err != nil {
		return nil, err
	}
	project = strings.TrimSpace(project)
	var out []Record
	for _, rec := range all {
		if rec.Project == project {
			out = append(out, rec)
		}
	}
	return out, nil
}

// DefaultLedgerPath resolves the canonical ledger path under repoRoot.
func DefaultLedgerPath(repoRoot string) string {
	if v := strings.TrimSpace(os.Getenv("HARNESS_JUDGMENT_LEDGER")); v != "" {
		return v
	}
	root := repoRoot
	if root == "" {
		root = resolveRepoRoot()
	}
	return filepath.Join(root, DefaultLedgerRelPath)
}

// DefaultSchemaPath returns the canonical schema path under repoRoot.
func DefaultSchemaPath(repoRoot string) string {
	root := repoRoot
	if root == "" {
		root = resolveRepoRoot()
	}
	return filepath.Join(root, SchemaRelPath)
}

func resolveRepoRoot() string {
	if v := os.Getenv("HARNESS_PROJECT_ROOT"); v != "" {
		return v
	}
	if v := os.Getenv("PROJECT_ROOT"); v != "" {
		return v
	}
	if out, err := gitport.Output("", "rev-parse", "--show-toplevel"); err == nil {
		if root := strings.TrimSpace(out); root != "" {
			return root
		}
	}
	cwd, _ := os.Getwd()
	return cwd
}
