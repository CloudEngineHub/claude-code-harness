package harnessmem

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setInvocationTestHooks(t *testing.T, goos string, lookPath func(string) (string, error)) {
	t.Helper()
	origGOOS := goosForInvocation
	origLookPath := lookPathForInvocation
	if goos != "" {
		goosForInvocation = goos
	}
	if lookPath != nil {
		lookPathForInvocation = lookPath
	}
	t.Cleanup(func() {
		goosForInvocation = origGOOS
		lookPathForInvocation = origLookPath
	})
}

func fakeLookPath(t *testing.T) func(string) (string, error) {
	t.Helper()
	return func(bin string) (string, error) {
		switch bin {
		case "node":
			return "/fake/bin/node", nil
		case "bun":
			return "/fake/bin/bun", nil
		}
		return "", errors.New("not found: " + bin)
	}
}

func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
	os.Unsetenv(key)
}

const (
	nodeShebangScript = "#!/usr/bin/env node\nconsole.log(1)\n"
	bunShebangScript  = "#!/usr/bin/env bun\nconsole.log(1)\n"
	bashShebangScript = "#!/bin/bash\necho hi\n"
)

func writeScript(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func runtimeRootCandidate(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	unsetEnvForTest(t, "HARNESS_MEM_CLI")
	return filepath.Join(home, ".harness-mem", "runtime", "harness-mem", "scripts", "harness-mem")
}

func TestResolveInvocation_WrapsJSExtensionWithNode(t *testing.T) {
	setInvocationTestHooks(t, "", fakeLookPath(t))

	script := filepath.Join(t.TempDir(), "harness-mem.js")
	writeScript(t, script, nodeShebangScript)
	t.Setenv("HARNESS_MEM_CLI", script)

	inv, ok := ResolveInvocation(false)
	if !ok {
		t.Fatal("expected invocation to resolve")
	}
	if inv.Name != "/fake/bin/node" {
		t.Errorf("Name = %q, want node runtime", inv.Name)
	}
	if len(inv.ArgPrefix) != 1 || inv.ArgPrefix[0] != script {
		t.Errorf("ArgPrefix = %v, want [%s]", inv.ArgPrefix, script)
	}
}

func TestResolveInvocation_BunShebangJSPrefersBun(t *testing.T) {
	setInvocationTestHooks(t, "", fakeLookPath(t))

	script := filepath.Join(t.TempDir(), "harness-mem.js")
	writeScript(t, script, bunShebangScript)
	t.Setenv("HARNESS_MEM_CLI", script)

	inv, ok := ResolveInvocation(false)
	if !ok {
		t.Fatal("expected invocation to resolve")
	}
	if inv.Name != "/fake/bin/bun" {
		t.Errorf("Name = %q, want bun runtime for bun shebang", inv.Name)
	}
}

func TestResolveInvocation_NoJSRuntimeKeepsOriginal(t *testing.T) {
	setInvocationTestHooks(t, "", func(bin string) (string, error) {
		return "", errors.New("not found: " + bin)
	})

	script := filepath.Join(t.TempDir(), "harness-mem.js")
	writeScript(t, script, nodeShebangScript)
	t.Setenv("HARNESS_MEM_CLI", script)

	inv, ok := ResolveInvocation(false)
	if !ok {
		t.Fatal("expected invocation to resolve")
	}
	if inv.Name != script {
		t.Errorf("Name = %q, want original script %q", inv.Name, script)
	}
}

func TestResolveInvocation_UnixExtensionlessNotWrapped(t *testing.T) {
	setInvocationTestHooks(t, "linux", fakeLookPath(t))

	candidate := runtimeRootCandidate(t)
	writeScript(t, candidate, nodeShebangScript)

	inv, ok := ResolveInvocation(false)
	if !ok {
		t.Fatal("expected invocation to resolve")
	}
	if inv.Name != candidate {
		t.Errorf("Name = %q, want unwrapped candidate %q", inv.Name, candidate)
	}
}

func TestResolveInvocation_WindowsPrefersJSSiblingOverBashWrapper(t *testing.T) {
	setInvocationTestHooks(t, "windows", fakeLookPath(t))

	candidate := runtimeRootCandidate(t)
	writeScript(t, candidate, bashShebangScript)
	jsCandidate := candidate + ".js"
	writeScript(t, jsCandidate, nodeShebangScript)

	inv, ok := ResolveInvocation(false)
	if !ok {
		t.Fatal("expected invocation to resolve")
	}
	if inv.Name != "/fake/bin/node" {
		t.Errorf("Name = %q, want node runtime", inv.Name)
	}
	if len(inv.ArgPrefix) != 1 || inv.ArgPrefix[0] != jsCandidate {
		t.Errorf("ArgPrefix = %v, want [%s]", inv.ArgPrefix, jsCandidate)
	}
}

func TestResolveInvocation_WindowsBashOnlyCandidateNotWrapped(t *testing.T) {
	setInvocationTestHooks(t, "windows", fakeLookPath(t))

	candidate := runtimeRootCandidate(t)
	writeScript(t, candidate, bashShebangScript)
	t.Setenv("HARNESS_MEM_DISABLE_PATH_LOOKUP", "1")

	inv, ok := ResolveInvocation(false)
	if !ok {
		t.Fatal("expected invocation to resolve")
	}
	if inv.Name != candidate {
		t.Errorf("Name = %q, want unwrapped bash candidate %q", inv.Name, candidate)
	}
}

func TestResolveInvocation_WindowsNodeShebangExtensionlessWrapped(t *testing.T) {
	setInvocationTestHooks(t, "windows", fakeLookPath(t))

	candidate := runtimeRootCandidate(t)
	writeScript(t, candidate, nodeShebangScript)

	inv, ok := ResolveInvocation(false)
	if !ok {
		t.Fatal("expected invocation to resolve")
	}
	if inv.Name != "/fake/bin/node" {
		t.Errorf("Name = %q, want node runtime", inv.Name)
	}
	if len(inv.ArgPrefix) != 1 || inv.ArgPrefix[0] != candidate {
		t.Errorf("ArgPrefix = %v, want [%s]", inv.ArgPrefix, candidate)
	}
}

func TestResolveInvocation_WindowsCmdShimNotWrapped(t *testing.T) {
	setInvocationTestHooks(t, "windows", func(bin string) (string, error) {
		if bin == "harness-mem" {
			return `C:\Users\test\AppData\Roaming\npm\harness-mem.cmd`, nil
		}
		return "", errors.New("not found: " + bin)
	})

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	unsetEnvForTest(t, "HARNESS_MEM_CLI")
	unsetEnvForTest(t, "HARNESS_MEM_DISABLE_PATH_LOOKUP")

	inv, ok := ResolveInvocation(false)
	if !ok {
		t.Fatal("expected invocation to resolve via PATH shim")
	}
	if !strings.HasSuffix(inv.Name, "harness-mem.cmd") {
		t.Errorf("Name = %q, want .cmd shim unwrapped", inv.Name)
	}
	if len(inv.ArgPrefix) != 0 {
		t.Errorf("ArgPrefix = %v, want empty", inv.ArgPrefix)
	}
}

func TestResolveInvocation_WindowsPathLookupJSWrapped(t *testing.T) {
	jsOnPath := `C:\Users\test\.harness-mem\runtime\harness-mem\scripts\harness-mem.js`
	setInvocationTestHooks(t, "windows", func(bin string) (string, error) {
		switch bin {
		case "harness-mem":
			return jsOnPath, nil
		case "node":
			return `C:\Program Files\nodejs\node.exe`, nil
		}
		return "", errors.New("not found: " + bin)
	})

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	unsetEnvForTest(t, "HARNESS_MEM_CLI")
	unsetEnvForTest(t, "HARNESS_MEM_DISABLE_PATH_LOOKUP")

	inv, ok := ResolveInvocation(false)
	if !ok {
		t.Fatal("expected invocation to resolve")
	}
	if inv.Name != `C:\Program Files\nodejs\node.exe` {
		t.Errorf("Name = %q, want node runtime", inv.Name)
	}
	if len(inv.ArgPrefix) != 1 || inv.ArgPrefix[0] != jsOnPath {
		t.Errorf("ArgPrefix = %v, want [%s]", inv.ArgPrefix, jsOnPath)
	}
}

func TestScriptRuntimePreference_WindowsExtensionlessByShebang(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantNeeds bool
		wantFirst string
	}{
		{"node shebang", nodeShebangScript, true, "node"},
		{"bun shebang", bunShebangScript, true, "bun"},
		{"env -S node", "#!/usr/bin/env -S node --no-warnings\n", true, "node"},
		{"bash shebang", bashShebangScript, false, ""},
		{"no shebang", "plain text\n", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setInvocationTestHooks(t, "windows", nil)
			script := filepath.Join(t.TempDir(), "harness-mem")
			writeScript(t, script, tt.content)

			needs, order := scriptRuntimePreference(script)
			if needs != tt.wantNeeds {
				t.Fatalf("needs = %v, want %v", needs, tt.wantNeeds)
			}
			if tt.wantNeeds && (len(order) == 0 || order[0] != tt.wantFirst) {
				t.Errorf("runtime order = %v, want first %q", order, tt.wantFirst)
			}
		})
	}
}
