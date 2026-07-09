package blastradius

import (
	"strings"
	"testing"
)

func TestBlastRadius_RecursiveDelete_Stops(t *testing.T) {
	esc, reason := Detect("rm -rf build/", 0, 0)
	if !esc || !strings.HasPrefix(reason, AxisDelete) {
		t.Fatalf("recursive delete should escalate on delete axis, got (%v, %q)", esc, reason)
	}
}

func TestBlastRadius_FindDelete_Stops(t *testing.T) {
	esc, reason := Detect("find . -name '*.tmp' -delete", 0, 0)
	if !esc || !strings.HasPrefix(reason, AxisDelete) {
		t.Fatalf("find -delete should escalate on delete axis, got (%v, %q)", esc, reason)
	}
}

func TestBlastRadius_ForcePushTriggers(t *testing.T) {
	for _, c := range []string{
		"git push --force origin main",
		"git push  origin  main  --force-with-lease",
		"git push -f origin feature",
	} {
		esc, reason := Detect(c, 0, 0)
		if !esc || !strings.HasPrefix(reason, AxisIrreversible) {
			t.Fatalf("force-push %q should escalate (irreversible), got (%v, %q)", c, esc, reason)
		}
	}
}

func TestBlastRadius_ResetHardTriggers(t *testing.T) {
	esc, reason := Detect("git reset --hard HEAD~3", 0, 0)
	if !esc || !strings.HasPrefix(reason, AxisIrreversible) {
		t.Fatalf("reset --hard should escalate, got (%v, %q)", esc, reason)
	}
}

func TestBlastRadius_TagPushTriggers(t *testing.T) {
	for _, c := range []string{"git push origin --tags", "git push origin v1.2.3"} {
		esc, reason := Detect(c, 0, 0)
		if !esc || !strings.HasPrefix(reason, AxisCrossRepo) {
			t.Fatalf("tag/version push %q should escalate (cross-repo), got (%v, %q)", c, esc, reason)
		}
	}
}

func TestBlastRadius_FileCountOverThreshold(t *testing.T) {
	esc, reason := Detect("echo hello", 50, 20)
	if !esc || !strings.HasPrefix(reason, AxisFileCount) {
		t.Fatalf("file count over threshold should escalate, got (%v, %q)", esc, reason)
	}
	// Under threshold: no escalation.
	if esc, _ := Detect("echo hello", 5, 20); esc {
		t.Fatalf("file count under threshold must not escalate")
	}
	// Threshold disabled (<=0): no escalation by count.
	if esc, _ := Detect("echo hello", 9999, 0); esc {
		t.Fatalf("disabled threshold must not escalate by count")
	}
}

func TestBlastRadius_NormalCommandPasses(t *testing.T) {
	for _, c := range []string{
		"go test ./...",
		"git push origin main", // ordinary push, no force/tags
		"git commit -m 'feat: x'",
		"rm file.txt", // single-file rm, no -r
		"ls -la",
	} {
		if esc, reason := Detect(c, 0, 0); esc {
			t.Fatalf("benign command %q must not escalate, got reason %q", c, reason)
		}
	}
}
