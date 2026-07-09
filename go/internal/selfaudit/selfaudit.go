// Package selfaudit は settings.local.json の hooks 注入を検知する。
// CCH 自身が書く delivery hook（92.6.2 の inbox check / inbox monitor）は
// 既知 fingerprint として allowlist し、その他の command 型 hook は injection
// として report する。
package selfaudit

import (
	"encoding/json"
	"strings"
)

type HookEntry struct {
	Event   string // "Stop" / "SessionStart" 等
	Type    string // "command" / "subagent" 等
	Command string
}

type Report struct {
	Known        []HookEntry // CCH 既知 hook
	Unknown      []HookEntry // 未知（注入の可能性）
	WarningCount int         // = len(Unknown)
}

// CCHKnownHooks は本パッケージが allowlist 対象として認識する command パターン。
// command 文字列の prefix match で判定する（settings.local.json の placeholder
// 展開後の文字列に対して。具体例: "bin/harness inbox check"）。
//
// 設計原則: 広すぎる pattern を入れない。"bin/harness" 単独 prefix は禁止
// （他の harness subcommand を混入させてしまう）。具体 subcommand 名まで含める。
var CCHKnownHooks = []string{
	"bin/harness inbox check",
	"bin/harness inbox monitor",
}

// Audit は settings.local.json の bytes を入力に取り、hooks 注入を分類する。
// 不正 JSON / hooks フィールド欠落は Warning 0 件・空 Report で返す（fail-open）。
func Audit(settingsLocalJSON []byte) (Report, error) {
	entries := extractCommandHooks(settingsLocalJSON)
	if entries == nil {
		return Report{}, nil
	}

	var known, unknown []HookEntry
	for _, entry := range entries {
		if IsKnown(entry.Command) {
			known = append(known, entry)
		} else {
			unknown = append(unknown, entry)
		}
	}

	return Report{
		Known:        known,
		Unknown:      unknown,
		WarningCount: len(unknown),
	}, nil
}

// IsKnown はある command 文字列が CCHKnownHooks のいずれかと prefix match するか。
func IsKnown(command string) bool {
	trimmed := strings.TrimSpace(command)
	for _, prefix := range CCHKnownHooks {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

func extractCommandHooks(data []byte) []HookEntry {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil
	}
	hooksRaw, ok := root["hooks"]
	if !ok {
		return nil
	}
	var hooks map[string]json.RawMessage
	if err := json.Unmarshal(hooksRaw, &hooks); err != nil {
		return nil
	}

	var entries []HookEntry
	for event, eventRaw := range hooks {
		var items []json.RawMessage
		if err := json.Unmarshal(eventRaw, &items); err != nil {
			continue
		}
		for _, item := range items {
			collectCommandHooks(event, item, &entries)
		}
	}
	return entries
}

func collectCommandHooks(event string, raw json.RawMessage, entries *[]HookEntry) {
	var node struct {
		Matcher string `json:"matcher"`
		Hooks   []struct {
			Type    string `json:"type"`
			Command string `json:"command"`
		} `json:"hooks"`
		Type    string `json:"type"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal(raw, &node); err != nil {
		return
	}

	if len(node.Hooks) > 0 {
		for _, hook := range node.Hooks {
			if hook.Type == "command" && strings.TrimSpace(hook.Command) != "" {
				*entries = append(*entries, HookEntry{
					Event:   event,
					Type:    hook.Type,
					Command: hook.Command,
				})
			}
		}
		return
	}

	if node.Type == "command" && strings.TrimSpace(node.Command) != "" {
		*entries = append(*entries, HookEntry{
			Event:   event,
			Type:    node.Type,
			Command: node.Command,
		})
	}
}
