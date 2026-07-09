package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestImpactScoreCommandMatchesGoImplementation(t *testing.T) {
	exe := buildHarnessForImpactScoreTest(t)

	cmd := exec.Command(exe, "impact-score", "--files-changed", "5", "--lines-changed", "200")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("impact-score command failed: %v", err)
	}

	var got impactScoreOutput
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode impact-score output: %v; output=%q", err, stdout.String())
	}
	if got.ImpactScore != 45 {
		t.Fatalf("ImpactScore = %d, want 45", got.ImpactScore)
	}
	if got.HardStop {
		t.Fatal("HardStop = true, want false")
	}
}

func TestImpactScoreCommandFloorExitsTwo(t *testing.T) {
	exe := buildHarnessForImpactScoreTest(t)

	cmd := exec.Command(exe, "impact-score", "--files-changed", "5", "--lines-changed", "200", "--floor-category", "egress")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err == nil {
		t.Fatal("impact-score floor command exit = 0, want 2")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("impact-score floor command error = %T %v, want ExitError", err, err)
	}
	if exitErr.ExitCode() != 2 {
		t.Fatalf("impact-score floor exit = %d, want 2", exitErr.ExitCode())
	}

	var got impactScoreOutput
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode impact-score floor output: %v; output=%q", err, stdout.String())
	}
	if got.ImpactScore != 100 || !got.HardStop {
		t.Fatalf("floor output = %+v, want score=100 hard_stop=true", got)
	}
}

func buildHarnessForImpactScoreTest(t *testing.T) string {
	t.Helper()

	exe := filepath.Join(t.TempDir(), "harness-test")
	cmd := exec.Command("go", "build", "-o", exe, ".")
	cmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "gocache"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build harness test binary failed: %v\n%s", err, string(out))
	}
	return exe
}
