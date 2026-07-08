package hookhandler

// task_completed_escalation.go - テスト失敗エスカレーション・Fix Proposal 管理

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// qualityGateEntry は task-quality-gate.json の各タスクエントリ。
type qualityGateEntry struct {
	FailureCount int    `json:"failure_count"`
	LastAction   string `json:"last_action"`
	UpdatedAt    string `json:"updated_at"`
}

// fixProposal は pending-fix-proposals.jsonl の1行分エントリ。
type fixProposal struct {
	SourceTaskID      string `json:"source_task_id"`
	FixTaskID         string `json:"fix_task_id"`
	TaskSubject       string `json:"task_subject"`
	ProposalSubject   string `json:"proposal_subject"`
	FailureCategory   string `json:"failure_category"`
	RecommendedAction string `json:"recommended_action"`
	DoD               string `json:"dod"`
	Depends           string `json:"depends"`
	CreatedAt         string `json:"created_at"`
	Status            string `json:"status"`
}

// testResultFile は .claude/state/test-result.json のスキーマ。
type testResultFile struct {
	Status  string `json:"status"`
	Command string `json:"command"`
	Output  string `json:"output"`
}

// checkTestResultAndEscalate はテスト結果を確認し、失敗カウントを管理する。
// testOK=false の場合は failCount も返す。
func (h *taskCompletedHandler) checkTestResultAndEscalate(taskID, taskSubject, teammateName, ts string) (testOK bool, failCount int) {
	resultFile := h.stateDir + "/test-result.json"

	// 結果ファイルがない場合は成功扱い（テスト不要なプロジェクト）
	if _, err := os.Stat(resultFile); err != nil {
		return true, 0
	}

	data, err := os.ReadFile(resultFile)
	if err != nil {
		return true, 0
	}
	var result testResultFile
	if err := json.Unmarshal(data, &result); err != nil {
		return true, 0
	}

	if result.Status != "failed" {
		// 成功またはタイムアウト: 失敗カウントをリセット
		h.updateFailureCount(taskID, "reset", ts)
		return true, 0
	}

	// テスト失敗
	failCount = h.updateFailureCount(taskID, "increment", ts)

	// 失敗をタイムラインに記録
	h.appendTimeline(timelineEntry{
		Event:        "test_result_failed",
		Teammate:     teammateName,
		TaskID:       taskID,
		Subject:      taskSubject,
		Timestamp:    ts,
		FailureCount: fmt.Sprintf("%d", failCount),
	})

	return false, failCount
}

// updateFailureCount は quality-gate.json のタスク別失敗カウントを更新する。
// 戻り値は新しいカウント値。
func (h *taskCompletedHandler) updateFailureCount(taskID, action, ts string) int {
	gatePath := h.stateDir + "/task-quality-gate.json"

	// 既存データを読み込む
	existing := make(map[string]qualityGateEntry)
	if data, err := os.ReadFile(gatePath); err == nil {
		// シンボルリンクチェック
		if info, err := os.Lstat(gatePath); err == nil && info.Mode()&os.ModeSymlink == 0 {
			_ = json.Unmarshal(data, &existing)
		}
	}

	entry := existing[taskID]
	if action == "increment" {
		entry.FailureCount++
	} else {
		entry.FailureCount = 0
	}
	entry.LastAction = action
	entry.UpdatedAt = ts
	existing[taskID] = entry

	// ファイルに書き戻す（シンボルリンクを拒否）
	if info, err := os.Lstat(gatePath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return entry.FailureCount
	}

	data, err := json.MarshalIndent(existing, "", "  ")
	if err == nil {
		tmpPath := gatePath + ".tmp"
		if err := os.WriteFile(tmpPath, append(data, '\n'), 0o644); err == nil {
			os.Rename(tmpPath, gatePath) //nolint:errcheck
		}
	}

	return entry.FailureCount
}

// buildFixTaskID は元タスク ID から fix タスク ID を生成する。
// 例: "26.1" → "26.1.fix", "26.1.fix" → "26.1.fix2", "26.1.fix2" → "26.1.fix3"
func buildFixTaskID(sourceTaskID string) string {
	// .fix{N} パターン
	if idx := strings.LastIndex(sourceTaskID, ".fix"); idx >= 0 {
		suffix := sourceTaskID[idx+4:]
		base := sourceTaskID[:idx]
		if suffix == "" {
			return base + ".fix2"
		}
		var n int
		if _, err := fmt.Sscanf(suffix, "%d", &n); err == nil {
			return fmt.Sprintf("%s.fix%d", base, n+1)
		}
	}
	return sourceTaskID + ".fix"
}

// classifyFailure はテスト出力から失敗カテゴリと推奨アクションを分類する。
func classifyFailure(output string) (category, action string) {
	lower := strings.ToLower(output)
	switch {
	case containsAny(lower, "syntax", "syntaxerror", "parse error", "unexpected token"):
		return "syntax_error", localizedHarnessMessage("ja", "Fix the syntax error. Check the code grammar.", "構文エラーを修正してください。コードの文法を確認してください。")
	case containsAny(lower, "cannot find module", "module not found", "import.*error", "modulenotfounderror"):
		return "import_error", localizedHarnessMessage("ja", "Fix the module/import error. Check dependencies (npm install / pip install).", "モジュール/インポートエラーを修正してください。依存関係を確認してください（npm install / pip install）。")
	case containsAny(lower, "type.*error", "typeerror", "is not assignable", "property.*does not exist"):
		return "type_error", localizedHarnessMessage("ja", "Fix the type error. Check mismatches between type definitions and implementation.", "型エラーを修正してください。型定義と実装の不一致を確認してください。")
	case containsAny(lower, "assertion", "assertionerror", "expect.*received", "tobe", "toequal", "fail", "failed"):
		return "assertion_error", localizedHarnessMessage("ja", "A test assertion failed. Check the difference between expected and actual values.", "テストアサーションが失敗しています。期待値と実際の値の差分を確認してください。")
	case containsAny(lower, "timeout", "etimedout", "timed out"):
		return "timeout", localizedHarnessMessage("ja", "A timeout occurred. Check asynchronous processing and network dependencies.", "タイムアウトが発生しました。非同期処理やネットワーク依存を確認してください。")
	case containsAny(lower, "permission", "eacces", "eperm", "access denied"):
		return "permission_error", localizedHarnessMessage("ja", "A permission error occurred. Check file permissions.", "権限エラーが発生しています。ファイルのパーミッションを確認してください。")
	default:
		return "runtime_error", localizedHarnessMessage("ja", "A runtime error occurred. Inspect the test output in detail.", "ランタイムエラーが発生しています。テスト出力を詳しく確認してください。")
	}
}

// containsAny はテキストが candidates のいずれかを含むか確認する。
func containsAny(text string, candidates ...string) bool {
	for _, c := range candidates {
		if strings.Contains(text, c) {
			return true
		}
	}
	return false
}

// emitEscalationResponse は3-strike エスカレーションレスポンスを出力する。
func (h *taskCompletedHandler) emitEscalationResponse(out io.Writer, taskID, taskSubject string, failCount int) error {
	ts := utcNow()

	// テスト出力を読み込む
	var lastCmd, lastOutput string
	resultFile := h.stateDir + "/test-result.json"
	if data, err := os.ReadFile(resultFile); err == nil {
		var result testResultFile
		if json.Unmarshal(data, &result) == nil {
			lastCmd = result.Command
			lastOutput = limitLines(result.Output, 20)
		}
	}

	category, action := classifyFailure(lastOutput)

	// エスカレーションレポートを stderr に出力
	fmt.Fprintf(os.Stderr, "\n==========================================\n")
	fmt.Fprint(os.Stderr, localizedHarnessMessage("ja", "[ESCALATION] Detected 3 consecutive failures - stopping automatic fix loop\n", "[ESCALATION] 3回連続失敗を検知 - 自動修正ループを停止\n"))
	fmt.Fprintf(os.Stderr, "==========================================\n")
	fmt.Fprintf(os.Stderr, localizedHarnessMessage("ja", "  Task ID       : %s\n", "  タスク ID  : %s\n"), taskID)
	fmt.Fprintf(os.Stderr, localizedHarnessMessage("ja", "  Task name     : %s\n", "  タスク名   : %s\n"), taskSubject)
	fmt.Fprintf(os.Stderr, localizedHarnessMessage("ja", "  Failure count : %d\n", "  連続失敗数 : %d\n"), failCount)
	fmt.Fprintf(os.Stderr, localizedHarnessMessage("ja", "  Detected at   : %s\n", "  検知時刻   : %s\n"), ts)
	fmt.Fprintf(os.Stderr, "------------------------------------------\n")
	fmt.Fprintf(os.Stderr, localizedHarnessMessage("ja", "  [Cause classification]\n  Category      : %s\n\n", "  [原因分類]\n  カテゴリ   : %s\n\n"), category)
	fmt.Fprintf(os.Stderr, localizedHarnessMessage("ja", "  [Recommended action]\n  %s\n\n", "  [推奨アクション]\n  %s\n\n"), action)
	if lastCmd != "" {
		fmt.Fprintf(os.Stderr, localizedHarnessMessage("ja", "  [Last command]\n  %s\n\n", "  [最後に実行したコマンド]\n  %s\n\n"), lastCmd)
	}
	if lastOutput != "" {
		fmt.Fprint(os.Stderr, localizedHarnessMessage("ja", "  [Test output (max 20 lines)]\n", "  [テスト出力（最大20行）]\n"))
		scanner := bufio.NewScanner(strings.NewReader(lastOutput))
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "    %s\n", scanner.Text())
		}
		fmt.Fprintln(os.Stderr)
	}
	fmt.Fprintf(os.Stderr, "==========================================\n\n")

	// エスカレーション記録をタイムラインに追記
	h.appendTimeline(timelineEntry{
		Event:        "escalation_triggered",
		TaskID:       taskID,
		Subject:      taskSubject,
		Timestamp:    ts,
		FailureCount: fmt.Sprintf("%d", failCount),
	})

	// Fix Proposal を生成・保存
	fixTaskID := buildFixTaskID(taskID)
	proposalSubject := sanitizeInlineText("fix: " + taskSubject + " - " + category)
	dod := sanitizeInlineText(localizedHarnessMessage("ja", "Resolve failure category ("+category+") and make the latest test/CI pass", "失敗カテゴリ ("+category+") を解消し、直近のテスト/CI が通ること"))

	proposal := fixProposal{
		SourceTaskID:      taskID,
		FixTaskID:         fixTaskID,
		TaskSubject:       taskSubject,
		ProposalSubject:   proposalSubject,
		FailureCategory:   category,
		RecommendedAction: action,
		DoD:               dod,
		Depends:           taskID,
		CreatedAt:         ts,
		Status:            "pending",
	}

	proposalSaved := h.upsertFixProposal(proposal)

	fixMessage := fmt.Sprintf(localizedHarnessMessage("ja",
		"[FIX PROPOSAL] Task %s failed 3 consecutive times.\nProposal: %s — %s\nDoD: %s\nApprove: approve fix %s\nReject: reject fix %s",
		"[FIX PROPOSAL] タスク %s が3回連続で失敗しました。\n提案: %s — %s\nDoD: %s\n承認: approve fix %s\n却下: reject fix %s"),
		taskID, fixTaskID, proposalSubject, dod, taskID, taskID)
	if !proposalSaved {
		fixMessage += localizedHarnessMessage("ja", "\nWarning: failed to save proposal. Add it to Plans.md manually.", "\n警告: proposal 保存に失敗しました。手動で Plans.md に追加してください。")
	}

	return writeJSON(out, map[string]string{
		"decision":      "approve",
		"reason":        "TaskCompleted: 3-strike escalation triggered - fix proposal queued",
		"systemMessage": fixMessage,
	})
}

// upsertFixProposal は pending-fix-proposals.jsonl に proposal を追加または更新する。
// 同一 source_task_id のエントリがあれば置き換える。
func (h *taskCompletedHandler) upsertFixProposal(proposal fixProposal) bool {
	// シンボルリンクチェック
	if info, err := os.Lstat(h.pendingFixFile); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return false
	}
	if info, err := os.Lstat(h.stateDir); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return false
	}

	// 既存エントリを読み込む（同一 source_task_id を除外）
	var rows []fixProposal
	if f, err := os.Open(h.pendingFixFile); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var row fixProposal
			if err := json.Unmarshal([]byte(line), &row); err != nil {
				continue
			}
			if row.SourceTaskID != proposal.SourceTaskID {
				rows = append(rows, row)
			}
		}
		f.Close()
	}
	rows = append(rows, proposal)

	// ファイルに書き戻す
	if err := os.MkdirAll(h.stateDir, 0o700); err != nil {
		return false
	}

	var buf []byte
	for _, row := range rows {
		data, err := json.Marshal(row)
		if err != nil {
			continue
		}
		buf = append(buf, data...)
		buf = append(buf, '\n')
	}

	if err := os.WriteFile(h.pendingFixFile, buf, 0o644); err != nil {
		return false
	}
	return true
}
