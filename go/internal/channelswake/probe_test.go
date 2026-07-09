package channelswake

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestChannelsWakeProbeScript(t *testing.T) {
	scriptPath := probeScriptPath(t)
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "AUTO_APPROVE_DEFAULT=false") {
		t.Fatal("probe script must contain literal AUTO_APPROVE_DEFAULT=false")
	}
	if !strings.Contains(body, "HARNESS_CHANNELS_WAKE_OPT_IN") {
		t.Fatal("probe script must gate wake on opt-in env")
	}
	if !strings.Contains(body, "channels-wake check") {
		t.Fatal("probe script must invoke harness channels-wake check")
	}
}

func probeScriptPath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	return filepath.Join(repoRoot, "scripts", "channels-wake-probe.sh")
}
