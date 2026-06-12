// Package selfaudit は .claude/settings.local.json の hooks 注入を検知する。
// CCH 自身が書く delivery hook（92.6.2 の inbox check / inbox monitor）は
// 既知 fingerprint として allowlist し、その他の command 型 hook は injection
// として report する。
package selfaudit

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
	return Report{}, nil
}

// IsKnown はある command 文字列が CCHKnownHooks のいずれかと prefix match するか。
func IsKnown(command string) bool {
	return false
}
