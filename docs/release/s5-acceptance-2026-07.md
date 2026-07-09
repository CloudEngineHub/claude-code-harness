# S5 Acceptance Ledger — redesign 線 区切りリリース (2026-07-08)

**判定: 11 項目全 green (M2 CI 側注記あり)。区切りリリース gate 通過。**

GOD_plans.md §7「安定稼働の受け入れ基準」11 項目の機械検証 evidence。
正本 GOD_plans.md は local-only で branch から到達不能のため、基準本文をここに inline 転記する（台帳を self-contained にする）。

| # | 基準 (inline) | 検証コマンド | 結果 | evidence |
|---|---|---|---|---|
| 1 | `bash tests/validate-plugin.sh` exit 0・失敗 0 | tests/validate-plugin.sh | ✅ | 失敗: 0 / 全テスト合格 |
| 2 | `go test ./...` 全 package exit 0・FAIL 0 | cd go && go test ./... | ✅ | 44 package ok / FAIL 0 (main-loop 環境で実行) |
| 3 | `bash scripts/ci/check-consistency.sh` 全ゲート exit 0 | scripts/ci/check-consistency.sh | ✅ | exit 0「すべてのチェックに合格」(binary drift gate 含む) |
| 4 | R15 相当ルールが go/internal/policy に存在し、秘密ファイルの git add 合成入力で pre-tool exit 2 | grep R15 rules.go + 既存 bypass test | ✅ | rules.go:186 `R15:no-stage-secret-file`、Phase 104.1 bypass test PASS |
| 5 | Plans.md Depends↔Status 整合チェッカーが存在し違反 0 | check-consistency.sh Plans dependency closure gate | ✅ | check-consistency.sh:1026 closure gate、全ゲート合格に含まれる |
| 6 | HARNESS_AUTO_APPROVE=on で承認スキップ test ≥1、または主張撤回済み | grep README/spec | ✅ (撤回) | README「records enablement gate... not skipped」+ sub-spec auto-approve scope 節で deferred 明記 (984456bf) |
| 7 | bridge/mailbox/bridgedelivery/triaddispatcher/impactscore 各 package が importer≥1 or 撤回注記 | 存在確認 + importer grep | ✅ | bridge系4=削除+retired-aliases 登録、impactscore=go/cmd/harness/impact_score.go が importer |
| 8 | platform binary と go/ source の checksum 一致 CI ゲート green | check-binary-source-drift.sh | ✅ | 109.0 で 4 platform 決定論再生成、drift OK (49024b95) |
| 9 | tests/test-hooks-sync.sh が 7/7 PASS | tests/test-hooks-sync.sh | ✅ | 7/7 passed |
| 10 | main merge-base 以降の安全 commit が port済/waive記録で未処理 0 件 | docs/branch-alignment-ledger.md | ✅ | 86 commit 全分類 (port 27 / already-included 7 / waive 52、未分類 0) |
| 11 | bin/harness self-audit の deny-baseline 非退行チェックが green | harness self-audit baseline | ✅ | current_sha256 == baseline_sha256 (ok) |

## 注記

- **M2 (CI 側 binary build flags)**: `.github/workflows/validate-plugin.yml` は plain `go build` (no -trimpath) で committed binary を上書きしてから drift check する構造のため、M2 patch (operator 手動適用) 未適用の間は **CI 側のみ** drift gate が赤になり得る。**ローカルは基準3で green 確認済み**。M2 は保護 path のため human-only。
- 基準 2 は sandbox 制約 (network bind) を避けるため main-loop 環境で実行 → 44 package ok / FAIL 0。
