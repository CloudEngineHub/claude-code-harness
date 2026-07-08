package plans

import (
	"os"
	"path/filepath"
	"testing"
)

// samplePlans exercises every status marker, an escaped pipe inside a DoD cell,
// the header row, and a separator row. ParseMarkdown must return all four task
// rows (including the Done row) and skip the header + separator.
const samplePlans = "" +
	"| Task | 内容 | DoD | Depends | Status |\n" +
	"|---|---|---|---|---|\n" +
	"| 1 | Feature X | build passes | none | `cc:WIP` |\n" +
	"| 2 | Write tests | tests green | 1 | cc:TODO |\n" +
	"| 3 | Ship it | released | 2 | cc:完了 |\n" +
	"| 4 | Migrate DB | run `a \\| b` filter | 3 | cc:blocked |\n"

func TestParseMarkdown_CountAndFields(t *testing.T) {
	tasks := ParseMarkdown(samplePlans)

	// 4 task rows; header + separator skipped.
	if len(tasks) != 4 {
		t.Fatalf("expected 4 task rows, got %d: %+v", len(tasks), tasks)
	}

	// Row 1 field extraction.
	got := tasks[0]
	if got.TaskID != "1" {
		t.Errorf("row0 TaskID: want 1, got %q", got.TaskID)
	}
	if got.Title != "Feature X" {
		t.Errorf("row0 Title: want %q, got %q", "Feature X", got.Title)
	}
	if got.DoD != "build passes" {
		t.Errorf("row0 DoD: want %q, got %q", "build passes", got.DoD)
	}
	if got.Depends != "none" {
		t.Errorf("row0 Depends: want %q, got %q", "none", got.Depends)
	}
	if got.Status != "`cc:WIP`" {
		t.Errorf("row0 Status: want %q, got %q", "`cc:WIP`", got.Status)
	}

	// Row 2 (TODO) Depends/Title extraction.
	if tasks[1].Depends != "1" {
		t.Errorf("row1 Depends: want %q, got %q", "1", tasks[1].Depends)
	}
	if tasks[1].Title != "Write tests" {
		t.Errorf("row1 Title: want %q, got %q", "Write tests", tasks[1].Title)
	}
}

func TestParseMarkdown_Tags(t *testing.T) {
	tasks := ParseMarkdown(samplePlans)
	if len(tasks) != 4 {
		t.Fatalf("expected 4 task rows, got %d", len(tasks))
	}

	cases := []struct {
		idx  int
		want Tags
	}{
		{0, Tags{Wip: true}},
		{1, Tags{Todo: true}},
		{2, Tags{Done: true}},
		{3, Tags{Blocked: true}},
	}
	for _, c := range cases {
		if got := tasks[c.idx].Tags; got != c.want {
			t.Errorf("row%d (%s) Tags: want %+v, got %+v",
				c.idx, tasks[c.idx].Status, c.want, got)
		}
	}

	// Explicitly assert the Done row is retained (unlike getPlanRows which drops it).
	if !tasks[2].Tags.Done {
		t.Error("expected Done row (row2) to be retained with Tags.Done=true")
	}
}

func TestParseMarkdown_EscapedPipeInDoD(t *testing.T) {
	tasks := ParseMarkdown(samplePlans)
	if len(tasks) != 4 {
		t.Fatalf("expected 4 task rows, got %d", len(tasks))
	}

	// The escaped `\|` must be un-escaped to a literal `|` inside the DoD cell,
	// and must NOT split the row into extra columns.
	want := "run `a | b` filter"
	if got := tasks[3].DoD; got != want {
		t.Errorf("row3 DoD un-escape: want %q, got %q", want, got)
	}
	if tasks[3].TaskID != "4" || tasks[3].Status != "cc:blocked" {
		t.Errorf("row3 boundaries shifted: TaskID=%q Status=%q", tasks[3].TaskID, tasks[3].Status)
	}
}

func TestParseMarkdown_SkipsHeaderAndSeparator(t *testing.T) {
	// A document that is ONLY a header + separator must yield zero task rows.
	content := "" +
		"| Task | 内容 | DoD | Depends | Status |\n" +
		"|---|---|---|---|---|\n"
	tasks := ParseMarkdown(content)
	if len(tasks) != 0 {
		t.Fatalf("expected 0 task rows from header+separator only, got %d: %+v", len(tasks), tasks)
	}

	// Lines without a pipe and rows with too few cells are skipped.
	mixed := "" +
		"Just some prose with no table.\n" +
		"| only | three | cells |\n" +
		"| 9 | Real | dod | none | cc:TODO |\n"
	tasks = ParseMarkdown(mixed)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task row from mixed content, got %d: %+v", len(tasks), tasks)
	}
	if tasks[0].TaskID != "9" {
		t.Errorf("expected TaskID=9, got %q", tasks[0].TaskID)
	}
}

func TestFind(t *testing.T) {
	tasks := ParseMarkdown(samplePlans)

	got := Find(tasks, "3")
	if got == nil {
		t.Fatal("Find(\"3\") returned nil")
	}
	if got.Title != "Ship it" {
		t.Errorf("Find(\"3\").Title: want %q, got %q", "Ship it", got.Title)
	}
	if !got.Tags.Done {
		t.Errorf("Find(\"3\") should be the Done row, got Tags %+v", got.Tags)
	}

	if missing := Find(tasks, "does-not-exist"); missing != nil {
		t.Errorf("Find(missing): want nil, got %+v", missing)
	}
}

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Plans.md")
	if err := os.WriteFile(path, []byte(samplePlans), 0o600); err != nil {
		t.Fatal(err)
	}

	tasks, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	if len(tasks) != 4 {
		t.Fatalf("ParseFile: expected 4 task rows, got %d", len(tasks))
	}

	// Unreadable path returns (nil, err).
	tasks, err = ParseFile(filepath.Join(dir, "nope.md"))
	if err == nil {
		t.Error("ParseFile(missing): expected error, got nil")
	}
	if tasks != nil {
		t.Errorf("ParseFile(missing): expected nil tasks, got %+v", tasks)
	}
}

func TestSplitPipeRow(t *testing.T) {
	// Outer pipes produce leading/trailing empty cells that must be trimmed.
	cells := SplitPipeRow("| a | b | c |")
	if len(cells) != 3 {
		t.Fatalf("expected 3 cells, got %d: %q", len(cells), cells)
	}

	// Escaped pipe stays inside a single cell as a literal '|'.
	cells = SplitPipeRow(`| a | b \| c | d |`)
	if len(cells) != 3 {
		t.Fatalf("escaped pipe: expected 3 cells, got %d: %q", len(cells), cells)
	}
	if cells[1] != " b | c " {
		t.Errorf("escaped pipe cell: want %q, got %q", " b | c ", cells[1])
	}
}
