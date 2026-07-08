package selfaudit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

const denyBaselineVersion = "deny-baseline.v1"

// DenyBaseline は .claude-plugin/settings.json の deny エントリ集合を hash で固定する。
//
// 設計:
//   - deny エントリは順序非依存（sorted + canonical JSON で hash 化）
//   - 起動時に現 settings.json から ComputeDenyHash → recorded baseline と比較
//   - 「同じ・追加」は通過、「減少」は起動拒否
//   - baseline は repo 内 templates/security/deny-baseline.json として固定（テキスト）
type DenyBaseline struct {
	Version         string   `json:"version"`
	CanonicalSHA256 string   `json:"canonical_sha256"`
	Entries         []string `json:"entries"`
}

// ComputeDenyHash は settings.json の permissions.deny を sorted unique + JSON
// に正規化し SHA-256 hex を返す。不正 JSON / deny フィールド欠落は ""（fail-loud
// 寸前の sentinel） + error。
func ComputeDenyHash(settingsJSON []byte) (canonicalSHA256 string, entries []string, err error) {
	entries, err = extractDenyEntries(settingsJSON)
	if err != nil {
		return "", nil, err
	}
	hash, err := hashDenyEntries(entries)
	if err != nil {
		return "", nil, err
	}
	return hash, entries, nil
}

// VerifyDenyNotRegressed は recorded baseline と current settings.json を比較
// し、deny エントリが「減少」していないかを返す。エラー or 減少なら false +
// 詳細メッセージ、それ以外（同一・追加）は true + 空文字。
func VerifyDenyNotRegressed(baseline DenyBaseline, currentSettingsJSON []byte) (ok bool, reason string, err error) {
	_, currentEntries, err := ComputeDenyHash(currentSettingsJSON)
	if err != nil {
		return false, "", err
	}

	currentSet := make(map[string]struct{}, len(currentEntries))
	for _, entry := range currentEntries {
		currentSet[entry] = struct{}{}
	}

	var missing []string
	for _, entry := range baseline.Entries {
		if _, found := currentSet[entry]; !found {
			missing = append(missing, entry)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return false, fmt.Sprintf("deny entry regression: removed %d baseline entr(y/ies): %s", len(missing), strings.Join(missing, ", ")), nil
	}
	return true, "", nil
}

// LoadBaseline は repo の deny-baseline.json を読む。ファイル不在は
// DenyBaseline{}, false, nil（fail-open、最初の bootstrap 想定）。
func LoadBaseline(path string) (DenyBaseline, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DenyBaseline{}, false, nil
		}
		return DenyBaseline{}, false, err
	}
	var baseline DenyBaseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return DenyBaseline{}, false, err
	}
	return baseline, true, nil
}

func extractDenyEntries(settingsJSON []byte) (entries []string, err error) {
	var root struct {
		Permissions struct {
			Deny []string `json:"deny"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(settingsJSON, &root); err != nil {
		return nil, fmt.Errorf("parse settings.json: %w", err)
	}
	if root.Permissions.Deny == nil {
		return nil, fmt.Errorf("permissions.deny field missing")
	}

	seen := make(map[string]struct{}, len(root.Permissions.Deny))
	for _, entry := range root.Permissions.Deny {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}
		seen[trimmed] = struct{}{}
	}
	entries = make([]string, 0, len(seen))
	for entry := range seen {
		entries = append(entries, entry)
	}
	sort.Strings(entries)
	return entries, nil
}

func hashDenyEntries(entries []string) (string, error) {
	canonical, err := json.Marshal(entries)
	if err != nil {
		return "", fmt.Errorf("marshal deny entries: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}
