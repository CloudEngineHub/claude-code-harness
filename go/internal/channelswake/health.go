// Package channelswake monitors Bridge Daemon communication channels
// (live-messaging / delivery / inbox hooks) and reports tri-state health for
// Session Monitor integration. Wake triggers are opt-in only (default OFF).
package channelswake

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/Chachamaru127/claude-code-harness/go/internal/eventstore"

	_ "modernc.org/sqlite"
)

const (
	ReasonNotConfigured     = "not-configured"
	ReasonDaemonUnreachable = "daemon-unreachable"
	ReasonCorrupted         = "corrupted"

	defaultStaleAfter = 24 * time.Hour
	socketDialTimeout = 500 * time.Millisecond
)

// Result is the JSON output of channels-wake health checks.
type Result struct {
	Healthy bool   `json:"healthy"`
	Reason  string `json:"reason"`
}

type channelsConfig struct {
	SocketPath        string `json:"socket_path"`
	MailboxDB         string `json:"mailbox_db"`
	StaleAfterSeconds int64  `json:"stale_after_seconds"`
}

var (
	socketProbeFn = probeSocket
	nowFn         = time.Now
)

// Check evaluates bridge socket reachability and mailbox freshness.
// not-configured is healthy (monitor exclusion); unreachable/corrupted are unhealthy.
func Check() Result {
	result, _ := CheckWithExit()
	return result
}

// CheckWithExit returns the health result and CLI exit code (0 = healthy/not-configured).
func CheckWithExit() (Result, int) {
	home, err := bridgeHome()
	if err != nil {
		return Result{Healthy: true, Reason: ReasonNotConfigured}, 0
	}

	cfgPath := filepath.Join(home, "channels.json")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return Result{Healthy: true, Reason: ReasonNotConfigured}, 0
	}

	cfg, err := loadChannelsConfig(cfgPath)
	if err != nil {
		return Result{Healthy: false, Reason: ReasonCorrupted}, 1
	}

	if cfg.SocketPath == "" || cfg.MailboxDB == "" {
		return Result{Healthy: false, Reason: ReasonCorrupted}, 1
	}

	if err := socketProbeFn(cfg.SocketPath); err != nil {
		return Result{Healthy: false, Reason: ReasonDaemonUnreachable}, 1
	}

	staleAfter := defaultStaleAfter
	if cfg.StaleAfterSeconds > 0 {
		staleAfter = time.Duration(cfg.StaleAfterSeconds) * time.Second
	}
	if stale, err := mailboxStale(cfg.MailboxDB, staleAfter, nowFn()); err != nil {
		return Result{Healthy: false, Reason: ReasonCorrupted}, 1
	} else if stale {
		return Result{Healthy: false, Reason: ReasonCorrupted}, 1
	}

	return Result{Healthy: true, Reason: ""}, 0
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

func loadChannelsConfig(path string) (channelsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return channelsConfig{}, err
	}
	var cfg channelsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return channelsConfig{}, err
	}
	return cfg, nil
}

func probeSocket(socketPath string) error {
	conn, err := net.DialTimeout("unix", socketPath, socketDialTimeout)
	if err != nil {
		return err
	}
	return conn.Close()
}

func mailboxStale(dbPath string, staleAfter time.Duration, now time.Time) (bool, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return false, fmt.Errorf("mailbox db missing")
	}

	db, err := eventstore.Open(dbPath)
	if err != nil {
		return false, err
	}
	defer db.Close()

	maxTS, err := eventstore.MaxTimestamp(db)
	if err != nil {
		return false, err
	}
	if !maxTS.Valid {
		return true, nil
	}

	last := time.Unix(0, maxTS.Int64)
	if now.Sub(last) > staleAfter {
		return true, nil
	}
	return false, nil
}

// WithSocketProbe replaces the socket probe for tests. The returned func restores it.
func WithSocketProbe(fn func(string) error) func() {
	prev := socketProbeFn
	socketProbeFn = fn
	return func() { socketProbeFn = prev }
}

// WithNow replaces the clock for tests. The returned func restores it.
func WithNow(fn func() time.Time) func() {
	prev := nowFn
	nowFn = fn
	return func() { nowFn = prev }
}
