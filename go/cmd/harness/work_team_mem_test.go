package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/breezingmem"
)

type fakeBreezingMem struct {
	mu     sync.Mutex
	events []string
}

func (f *fakeBreezingMem) RecordEvent(_ context.Context, eventType, _, _, _ string) {
	f.mu.Lock()
	f.events = append(f.events, eventType)
	f.mu.Unlock()
}

func (f *fakeBreezingMem) IngestBrief(_ context.Context, _, _ string, _ []byte) {}

func (f *fakeBreezingMem) snapshot() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.events...)
}

func TestRunTeam_EmitsBreezingMemEvents(t *testing.T) {
	passingFloorGate(t)

	fake := &fakeBreezingMem{}
	origMem := breezingMemClient
	breezingMemClient = fake
	defer func() { breezingMemClient = origMem }()

	indexOf := map[string]int{"t1": 0, "t2": 1}
	var mu sync.Mutex
	var calls []string
	orig := teamWorkerFactory
	teamWorkerFactory = recordingFactory(indexOf, &calls, &mu)
	defer func() { teamWorkerFactory = orig }()

	root := t.TempDir()
	t.Setenv("HARNESS_PROJECT_ROOT", root)
	t.Setenv("HARNESS_BREEZING_SESSION_ID", "sess-team-mem")

	if _, err := runTeam([]string{"t1", "t2"}, "codex", 2); err != nil {
		t.Fatalf("runTeam: %v", err)
	}

	got := fake.snapshot()
	want := []string{
		breezingmem.EventRunStarted,
		breezingmem.EventWorkerResult,
		breezingmem.EventWorkerResult,
		breezingmem.EventAggregationDone,
	}
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events[%d] = %q, want %q (all=%v)", i, got[i], want[i], got)
		}
	}
}

func TestWorkTeamMem_NoSignalAPIReferences(t *testing.T) {
	data, err := os.ReadFile("work_team.go")
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(string(data))
	for _, needle := range []string{"/v1/signals", "signal_send", "signal_read", "signal_ack"} {
		if strings.Contains(lower, needle) {
			t.Fatalf("work_team.go must not reference signal API %q", needle)
		}
	}
}

func TestMemRecordBreezingEvent_BriefConfirmed(t *testing.T) {
	orig := breezingMemClient
	defer func() { breezingMemClient = orig }()

	fake := &fakeBreezingMem{}
	breezingMemClient = fake

	var stdout, stderr bytes.Buffer
	code := runMemRecordBreezingEvent([]string{
		"--type", "brief-confirmed",
		"--project", "/tmp/proj",
		"--session", "sess-brief",
		"--content", `{"approved":true}`,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}

	got := fake.snapshot()
	if len(got) != 1 || got[0] != breezingmem.EventBriefConfirmed {
		t.Fatalf("events = %v, want [%q]", got, breezingmem.EventBriefConfirmed)
	}
}
