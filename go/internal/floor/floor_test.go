package floor

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// fakeRunner is an injected ScriptRunner that returns a per-script (exitCode,
// output) without shelling out. Scripts not in fail are treated as passing
// (exit 0).
type fakeRunner struct {
	// fail maps a script path to the exit code it should return (non-zero).
	fail map[string]int
	// calls records the scripts that were run, in order.
	calls []string
}

func (r *fakeRunner) Run(_ string, script string, _ ...string) (int, string) {
	r.calls = append(r.calls, script)
	if code, ok := r.fail[script]; ok {
		return code, fmt.Sprintf("simulated failure of %s", script)
	}
	return 0, "ok"
}

// allScripts lists the contract script paths the gate must run, for assertions.
func allScripts() []string {
	out := make([]string, 0, len(contractScripts))
	for _, cs := range contractScripts {
		out = append(out, cs.script)
	}
	return out
}

// contractStepNames lists the step NAMES the gate records for the contract
// scripts (these differ from the script paths: the step is the bare script
// basename constant, the path is repo-relative).
func contractStepNames() []string {
	out := make([]string, 0, len(contractScripts))
	for _, cs := range contractScripts {
		out = append(out, cs.step)
	}
	return out
}

// stepNameForScript returns the step name the gate uses for a given script path.
func stepNameForScript(script string) string {
	for _, cs := range contractScripts {
		if cs.script == script {
			return cs.step
		}
	}
	return script
}

func stepByName(rep Report, name string) (StepResult, bool) {
	for _, s := range rep.Steps {
		if s.Name == name {
			return s, true
		}
	}
	return StepResult{}, false
}

// TestGate_AllPass: clean changed files + every script passing ⇒ Passed=true,
// and every expected step is present and passing.
func TestGate_AllPass(t *testing.T) {
	runner := &fakeRunner{}
	rep := Gate("/repo", []string{"go/internal/foo.go", "docs/readme.md"}, runner)

	if !rep.Passed {
		t.Fatalf("expected Passed=true, got false; steps=%+v", rep.Steps)
	}
	for _, s := range rep.Steps {
		if !s.Passed {
			t.Errorf("step %q failed unexpectedly: %s", s.Name, s.Detail)
		}
	}

	// All three contract scripts were actually invoked, in order.
	if strings.Join(runner.calls, ",") != strings.Join(allScripts(), ",") {
		t.Errorf("scripts run = %v, want %v", runner.calls, allScripts())
	}

	// The expected step set is present (chain + policy + 3 scripts = 5).
	wantSteps := append([]string{StepDenySurface, StepPolicyCheck}, contractStepNames()...)
	if len(rep.Steps) != len(wantSteps) {
		t.Fatalf("got %d steps, want %d (%v)", len(rep.Steps), len(wantSteps), wantSteps)
	}
	for _, name := range wantSteps {
		if _, ok := stepByName(rep, name); !ok {
			t.Errorf("missing expected step %q", name)
		}
	}
}

// TestGate_AnySingleScriptFailureFailsGate: looping over each of the 3 scripts,
// failing exactly that one ⇒ Passed=false, and ONLY that step fails. This proves
// no single contract grep can be skipped.
func TestGate_AnySingleScriptFailureFailsGate(t *testing.T) {
	for _, target := range allScripts() {
		t.Run(target, func(t *testing.T) {
			runner := &fakeRunner{fail: map[string]int{target: 1}}
			rep := Gate("/repo", []string{"go/internal/foo.go"}, runner)

			if rep.Passed {
				t.Fatalf("gate passed despite %s failing", target)
			}
			stepName := stepNameForScript(target)
			s, ok := stepByName(rep, stepName)
			if !ok {
				t.Fatalf("failing script %s (step %q) has no step", target, stepName)
			}
			if s.Passed {
				t.Errorf("step %q should be failed", stepName)
			}
			// Every OTHER step passed (only the targeted script fails).
			for _, other := range rep.Steps {
				if other.Name == stepName {
					continue
				}
				if !other.Passed {
					t.Errorf("unrelated step %q failed: %s", other.Name, other.Detail)
				}
			}
		})
	}
}

// TestGate_WeakenedDenySurfaceFailsEvenIfScriptsPass: the bypass-closure test.
// Even with all scripts passing and clean files, a weakened deny surface
// (injected via the verifyDenySurface seam) ⇒ Passed=false, with the failure
// pinned to the deny-surface step.
func TestGate_WeakenedDenySurfaceFailsEvenIfScriptsPass(t *testing.T) {
	orig := verifyDenySurface
	verifyDenySurface = func() error {
		return errors.New("deny surface weakened: R06:no-force-push missing")
	}
	defer func() { verifyDenySurface = orig }()

	runner := &fakeRunner{} // all scripts pass
	rep := Gate("/repo", []string{"go/internal/foo.go"}, runner)

	if rep.Passed {
		t.Fatal("gate passed with a weakened deny surface — bypass NOT closed")
	}
	s, ok := stepByName(rep, StepDenySurface)
	if !ok || s.Passed {
		t.Fatalf("deny-surface step should be present and failed; got %+v ok=%v", s, ok)
	}
	if !strings.Contains(s.Detail, "weakened") {
		t.Errorf("deny-surface failure detail = %q, want it to mention the weakening", s.Detail)
	}
	// The scripts still ran and still passed; the gate fails purely on the chain.
	for _, name := range contractStepNames() {
		if st, _ := stepByName(rep, name); !st.Passed {
			t.Errorf("script step %q should still pass, got failed: %s", name, st.Detail)
		}
	}
}

// TestGate_DenyTriggeringChangeFailsPolicyStep: a changed file whose write the
// R01-R13 engine denies (a protected secret path, .env) ⇒ the policy-check step
// fails ⇒ Passed=false, even though every script passes. This proves the gate
// re-evaluates the candidate changes through the real engine.
func TestGate_DenyTriggeringChangeFailsPolicyStep(t *testing.T) {
	runner := &fakeRunner{} // scripts pass
	rep := Gate("/repo", []string{".env"}, runner)

	if rep.Passed {
		t.Fatal("gate passed despite a deny-triggering change to .env")
	}
	s, ok := stepByName(rep, StepPolicyCheck)
	if !ok || s.Passed {
		t.Fatalf("policy-check step should be present and failed; got %+v ok=%v", s, ok)
	}
	if !strings.Contains(s.Detail, ".env") {
		t.Errorf("policy-check failure detail = %q, want it to name the denied file", s.Detail)
	}
}

// TestGate_CleanChangesPassPolicyStep is the negative control for the policy
// step: an ordinary source file is not denied, so the policy step passes.
func TestGate_CleanChangesPassPolicyStep(t *testing.T) {
	runner := &fakeRunner{}
	rep := Gate("/repo", []string{"go/internal/floor/floor.go"}, runner)
	s, ok := stepByName(rep, StepPolicyCheck)
	if !ok {
		t.Fatal("missing policy-check step")
	}
	if !s.Passed {
		t.Errorf("clean source change should pass policy step, got: %s", s.Detail)
	}
}

// TestGate_NilRunnerDoesNotPanic ensures Gate tolerates a nil runner by falling
// back to the default (which would shell out). We only assert it does not panic
// and produces the full step set; the default runner's script results depend on
// the environment, so we do not assert Passed here.
func TestGate_NilRunnerDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Gate panicked with nil runner: %v", r)
		}
	}()
	// Point at a non-existent repo root so the default runner's scripts just
	// fail (exit 127) rather than executing anything real.
	rep := Gate(t.TempDir(), []string{"go/internal/foo.go"}, nil)
	if len(rep.Steps) == 0 {
		t.Error("expected steps even with nil runner")
	}
}
