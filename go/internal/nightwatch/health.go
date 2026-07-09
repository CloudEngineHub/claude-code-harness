// Package nightwatch implements Night Watch patrol health (Phase 99.1):
// tri-state active watching for unresolved loops / stale tasks / open decisions.
package nightwatch

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"time"
)

const (
	ReasonNotConfigured     = "not-configured"
	ReasonDaemonUnreachable = "daemon-unreachable"
	ReasonCorrupted         = "corrupted"

	SchemaRelPath = "templates/schemas/night-watch-report.v1.json"
	ConfigRelPath = "templates/night-watch-config.yaml"
)

// Result is the JSON output of night-watch health checks.
type Result struct {
	Healthy bool   `json:"healthy"`
	Reason  string `json:"reason"`
}

type nightWatchConfig struct {
	Enabled bool `json:"enabled"`
}

var (
	socketProbeFn = probeSocket
)

// Check evaluates Night Watch opt-in state and bridge mailbox reachability.
func Check() Result {
	result, _ := CheckWithExit()
	return result
}

// CheckWithExit returns health result and CLI exit code (0 = healthy/not-configured).
func CheckWithExit() (Result, int) {
	if !isEnabled() {
		return Result{Healthy: true, Reason: ReasonNotConfigured}, 0
	}

	home, err := bridgeHome()
	if err != nil {
		return Result{Healthy: true, Reason: ReasonNotConfigured}, 0
	}

	cfgPath := filepath.Join(home, "channels.json")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return Result{Healthy: true, Reason: ReasonNotConfigured}, 0
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return Result{Healthy: false, Reason: ReasonCorrupted}, 1
	}

	var bridgeCfg struct {
		SocketPath string `json:"socket_path"`
		MailboxDB  string `json:"mailbox_db"`
	}
	if err := json.Unmarshal(data, &bridgeCfg); err != nil {
		return Result{Healthy: false, Reason: ReasonCorrupted}, 1
	}
	if bridgeCfg.SocketPath == "" || bridgeCfg.MailboxDB == "" {
		return Result{Healthy: false, Reason: ReasonCorrupted}, 1
	}

	if err := socketProbeFn(bridgeCfg.SocketPath); err != nil {
		return Result{Healthy: false, Reason: ReasonDaemonUnreachable}, 1
	}

	if _, err := os.Stat(bridgeCfg.MailboxDB); err != nil {
		return Result{Healthy: false, Reason: ReasonCorrupted}, 1
	}

	return Result{Healthy: true, Reason: ""}, 0
}

func isEnabled() bool {
	if os.Getenv("NIGHT_WATCH_ENABLED") == "true" {
		return true
	}
	home, err := nightWatchHome()
	if err != nil {
		return false
	}
	cfgPath := filepath.Join(home, "night-watch.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return false
	}
	var cfg nightWatchConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return false
	}
	return cfg.Enabled
}

func nightWatchHome() (string, error) {
	if v := os.Getenv("HARNESS_NIGHT_WATCH_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".harness-night-watch"), nil
}

func bridgeHome() (string, error) {
	if v := os.Getenv("HARNESS_BRIDGE_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".harness-bridge"), nil
}

func probeSocket(socketPath string) error {
	conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err != nil {
		return err
	}
	return conn.Close()
}

// WithSocketProbe replaces the socket probe for tests.
func WithSocketProbe(fn func(string) error) func() {
	prev := socketProbeFn
	socketProbeFn = fn
	return func() { socketProbeFn = prev }
}
