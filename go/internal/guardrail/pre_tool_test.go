package guardrail

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/pkg/hookproto"
)

func TestEvaluatePreTool_R03ProtectedPathAskListFromHarnessTOML(t *testing.T) {
	projectRoot := t.TempDir()
	tomlPath := filepath.Join(projectRoot, "harness.toml")
	data := []byte(`
[[safety.guardrail.protectedPathAskList]]
path = ".env"
reason = "customer deploy env update"
`)
	if err := os.WriteFile(tomlPath, data, 0o600); err != nil {
		t.Fatalf("write harness.toml: %v", err)
	}

	result := EvaluatePreTool(hookproto.HookInput{
		CWD:      projectRoot,
		ToolName: "Bash",
		ToolInput: map[string]interface{}{
			"command": "printf 'SECRET=foo\n' > .env",
		},
	})

	if result.Decision != hookproto.DecisionAsk {
		t.Fatalf("expected ask, got %s", result.Decision)
	}
	if !strings.Contains(result.Reason, "R03") ||
		!strings.Contains(result.Reason, ".env") ||
		!strings.Contains(result.Reason, tomlPath) ||
		!strings.Contains(result.Reason, "customer deploy env update") {
		t.Fatalf("ask reason missing audit details: %q", result.Reason)
	}
	if strings.Contains(result.Reason, "SECRET=foo") {
		t.Fatalf("ask reason echoed command content: %q", result.Reason)
	}
}

func TestEvaluatePreTool_R03ProtectedPathAskListIgnoresPluginRootTOML(t *testing.T) {
	projectRoot := t.TempDir()
	pluginRoot := t.TempDir()
	pluginTomlPath := filepath.Join(pluginRoot, "harness.toml")
	data := []byte(`
[[safety.guardrail.protectedPathAskList]]
path = ".env"
reason = "plugin global config must not relax project policy"
`)
	if err := os.WriteFile(pluginTomlPath, data, 0o600); err != nil {
		t.Fatalf("write plugin harness.toml: %v", err)
	}

	result := EvaluatePreTool(hookproto.HookInput{
		CWD:        projectRoot,
		PluginRoot: pluginRoot,
		ToolName:   "Bash",
		ToolInput: map[string]interface{}{
			"command": "printf 'SECRET=foo\n' > .env",
		},
	})

	if result.Decision != hookproto.DecisionDeny {
		t.Fatalf("expected project-local default deny, got %s", result.Decision)
	}
	if strings.Contains(result.Reason, "plugin global config") {
		t.Fatalf("deny reason should not use plugin-root config: %q", result.Reason)
	}
}

func TestEvaluatePreTool_RuntimeFloorHardStopBeforeRules(t *testing.T) {
	projectRoot := t.TempDir()
	dangerousCmd := "gh release create v9.9.9"

	envVars := map[string]string{
		"HARNESS_AUTO_APPROVE":    "on",
		"HARNESS_RUNTIME_FLOOR":   "off",
		"HARNESS_DISABLE_GUARDRAIL": "1",
	}
	for key, value := range envVars {
		t.Setenv(key, value)
	}

	result := EvaluatePreTool(hookproto.HookInput{
		CWD:      projectRoot,
		ToolName: "Bash",
		ToolInput: map[string]interface{}{
			"command": dangerousCmd,
		},
	})

	if result.Decision != hookproto.DecisionAsk {
		t.Fatalf("expected ask from runtime floor, got %s (reason=%q)", result.Decision, result.Reason)
	}
	if !strings.HasPrefix(result.Reason, "RUNTIME_FLOOR:") {
		t.Fatalf("expected RUNTIME_FLOOR reason prefix, got %q", result.Reason)
	}
	if !strings.Contains(result.Reason, "prod-deploy") {
		t.Fatalf("expected prod-deploy category in reason, got %q", result.Reason)
	}
}
