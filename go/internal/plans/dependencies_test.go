package plans

import (
	"strings"
	"testing"
)

func TestCheckDoneDependencies_AllClosed(t *testing.T) {
	content := "" +
		"| Task | 内容 | DoD | Depends | Status |\n" +
		"|---|---|---|---|---|\n" +
		"| 1 | Base | done | - | cc:done |\n" +
		"| 2 | Withdrawn | no longer needed | - | cc:withdrawn |\n" +
		"| 3 | Final | done | 1, 2 | cc:完了 |\n"

	violations := CheckDoneDependencies(ParseMarkdown(content))
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %+v", violations)
	}
}

func TestCheckDoneDependencies_Violation(t *testing.T) {
	content := "" +
		"| Task | 内容 | DoD | Depends | Status |\n" +
		"|---|---|---|---|---|\n" +
		"| 1 | Base | todo | - | cc:todo |\n" +
		"| 2 | Final | done | 1 | cc:done |\n"

	violations := CheckDoneDependencies(ParseMarkdown(content))
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %+v", violations)
	}
	if violations[0].TaskID != "2" || violations[0].DependencyID != "1" {
		t.Fatalf("unexpected violation: %+v", violations[0])
	}
	if !strings.Contains(FormatDependencyViolations(violations), "task 2 is done but depends on 1") {
		t.Fatalf("formatted violation missing task/dependency: %q", FormatDependencyViolations(violations))
	}
}
