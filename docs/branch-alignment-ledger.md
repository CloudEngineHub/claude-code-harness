# Branch Alignment Ledger

This ledger tracks safety commits from main that must be ported or explicitly waived before redesign/mainline alignment proceeds. Status values are intentionally closed: `ported` or `waived` only.

| Commit | Area | Status | Why |
|--------|------|--------|-----|
| `1873c8a3` | R15 secret-file staging block | ported | Translated into `go/internal/policy` in Phase 104.1. |
| `6ac3eec0` | R15 git `-C` / subshell bypass | ported | Covered by Phase 104.1 bypass tests. |
| `4d7b6245` | R15 quote-aware lexing | ported | Covered by Phase 104.1 parser/policy port. |
| `3b8e64db` | R15 backslash escape | ported | Covered by Phase 104.1 bypass tests. |
| `5249ad76` | PR #218 safety contract | waived | Product contract already represented on this branch; no additional source delta required for S1. |
| `5a2d0df9` | PR #218 follow-up | waived | Same boundary as `5249ad76`; tracked to avoid silent omission. |
| `d4b8573c` | PR #219 safety contract | waived | Not required for the Phase 104 P0 gate after current source audit. |
| `d9b3fd34` | Egress owner-scope | waived | Owner-scope policy is outside Phase 104 gate implementation; keep as explicit branch-alignment item. |
| `2141c7ef` | setup-hook guard | waived | Setup hook guard does not block S1 gate completion on this branch. |
| `8097802e` | setup-hook guard follow-up | waived | Same boundary as `2141c7ef`; tracked as a paired main commit. |
| `fa88d4cf` | autonomous-confirmation scope | ported | Ported `.claude/rules/autonomous-confirmation-scope.md` (52 lines) to redesign in Phase 104.5. |

## 2026-07-07 Full Inventory (Phase 109.1 — release alignment)

Scope: `git log --no-merges 794bfc36..main` = **86 commits**（Plans.md 記載の 116 は merge commit 30 件込み）。上の S1 期の waive は「Phase 104 P0 gate に不要」という判定であり、release alignment では再分類する（下表が優先）。

| Group | Commits | Class | Basis | Rep SHA |
|---|---|---|---|---|
| 2026-07-07 guard hardening | 4 | **port (reimplement)** | redesign guard に bookkeeping 免除が無く hooks.json wrapper に CLAUDE_PROJECT_DIR/$PWD 穴が現存。cherry-pick 不可、機能ごと再実装 | ed6b18c9 |
| bookkeeping 免除 base + 2 rounds | 3 | **port (decision 2026-07-07)** | 免除が無いと harness-release の multi-commit flow (109.5) が #219 と同型でブロックされる。hardening 込みで移植 | d4b8573c |
| runtimefloor egress owner-scope | 1 | **port** | HARNESS_RUNTIME_FLOOR_EGRESS=off 免除が redesign に無い。additive (secret-read 側と衝突なし) | d9b3fd34 |
| SubagentStop reviewer-persist backstop | 1 | **port (Go 再実装)** | redesign の runSubagentStop は lifecycle のみで review-result.json を書かない | 5249ad76 |
| dependency/security bumps | 5 | **port (verify each)** | 低リスク。benchmarks/.github 配下の version 乖離だけ確認 | 4f962eb3 |
| known-limitations doc | 1 | **port** | redesign に相当 doc なし、standalone | 08fa2331 |
| autonomous-confirmation-scope / runtimefloor 5-cat base / R15 hardening / review-result part-1 | 7 | already-included | byte 一致 or Phase 104.1/108.x で superseding 実装済み | 1873c8a3 |
| skill refactor 鎖 / mirror sync / release bookkeeping / Phase 94 bookkeeping / #200 trims / hooks doc-drift / P35 footer / cross-session relay 15 件 | 43 | waive | redesign 側で構造置換済み (channelswake/livemsg/sublead 等) or main 固有 bookkeeping | b1141223 |
| Phase 95 release delegation | 9 | conflict-review → 109.1b で検証 | redesign harness-release の references/ 未確認 | f499f577 |
| Windows harness-mem companion | 2 | conflict-review → 109.1b | redesign companion.go に Windows 分岐なし。Go-native 化済みか要確認 | c8706db8 |
| setup-hook harness.toml bootstrap | 3 | conflict-review → 109.1b | fresh install に必要か要確認 | 8097802e |
| release-chore 内の実 fix / bc96f759 / 小型 docs / HOTL wip | 7 | conflict-review → 109.1b | 個別に軽検証して port/waive 確定 | 7dd175c5 |

**Totals**: port 15 / already-included 7 / waive 43 / conflict-review (109.1b で確定) 21 = 86, 未分類 0。

## 2026-07-07 DP-2 / DP-4 裁定反映 (Phase 109.2, operator 承認済み)

- **DP-2 (cut list)**: bridge/mailbox/bridgedelivery/triaddispatcher = Phase 104.4 で削除済 (importer 0)。impactscore = Go 統一済 (importer: go/cmd/harness/impact_score.go)。曖昧 skill gogcli-ops/cc-cursor-cc = 削除済 + retired-aliases.v1.yaml 登録済。agent-browser/cc-update-review = 保持裁定どおり存置。→ 追加作業なし、裁定は既に実装反映済み。
- **DP-4 (auto-approve 主張撤回)**: README は既に「ledger only, prompts not skipped」表現。sub-spec docs/spec/operations-memory-and-collaboration.md の空だった auto-approve scope 節に撤回本文を追記 (984456bf)。→ GOD_plans §7-6 の撤回パス green。

## 2026-07-07 conflict-review 21 件の確定 (Phase 109.1b)

精査結果: port 7 グループ / waive 3 グループ。21 raw commit は代表 SHA でグループ化されていたため、下記で個別解決。

| commit群 | 確定 | 根拠 | port 対象ファイル |
|---|---|---|---|
| Phase 95 release delegation (9, f499f577) | waive | redesign は既に gh release create 呼び出しなし + release.yml + test-release-skill-no-gh-release.sh 存在。冗長 | — |
| Windows harness-mem companion (0e3d5ab6, c8706db8) | **port** | redesign companion.go に Windows 分岐ゼロ。Win で "%1 is not a valid Win32 application" (#207) | go/internal/harnessmem/companion.go (手動再移植) |
| setup-hook harness.toml bootstrap (8097802e) | **port** | runSetupInit に harness.toml 生成なし。fresh install で harness sync 失敗 (#201) | setup_hook.go + scaffold 抽出 |
| 7dd175c5 plugin runtime cache | **port (partial)** | direct-script hook wrapper に file-existence guard 欠落 + codex-companion.sh の MODEL_ARGS unbound (4 site)。version bump hunk は waive | hooks.json ×2 / sync-plugin-cache.sh / build-host-plugin-dist.sh / codex-companion.sh |
| 631ed798 CI gate | **port (test fix)** | runtimefloor_test.go の t.TempDir() が /tmp allowlist と衝突し Linux CI で誤 fail。CHANGELOG hunk は waive | runtimefloor_test.go の home path 固定化のみ |
| 5d537b7c codex AGENTS.md | **port** | codex/AGENTS.md に stale "Hooks は未対応" 3 箇所 (実態と矛盾、MEMORY North Star note と不整合) | codex/AGENTS.md 3 行 |
| 08fa2331 reviewer + known-limitations | **port** | known-limitations.md 不在 + reviewer.md に中立列挙 instruction なし | agents/reviewer.md + docs/known-limitations.md |
| a6f58e20 i18n.md SSOT | **port** | docs/i18n.md 不在 + pointer なし | docs/i18n.md + CLAUDE.md/README pointer |
| 451b4a68 cursor tier | waive (moot) | redesign の cursor tier は candidate のまま。PR #174 promotion は redesign 系譜に未 merge。修復対象が存在しない | — |
| f6cdb042 HOTL wip | waive (moot) | plan-brief/accept は redesign で独立配線済 (superseded)。progress-tracker hook は依存 script 不在で dead code。self-labeled wip | — |

**確定**: conflict-review 21 → port 12 commit相当 (8 rep SHA) / waive 9 commit相当。→ 全 86 commit の分類完了 (port 27 / already-included 7 / waive 52、未分類 0)。

## 2026-07-08 本流化 merge 戦略 (Phase 109.4, operator 承認 戦略1)

`git merge -s ours origin/main` で main を ancestor に取り込み、tree は redesign を採用。理由: port 27 commit は 109.1a/109.1b で手移植済み、waive 52 は redesign が意図的に置換・削除した subsystem との衝突のため取り込まない。merge 後 `git rev-list HEAD..origin/main` = 0 (PR conflict 0)。

**-s ours で取りこぼす main 側差分と扱い**:
- `go/go.mod` / `go.sum`: main の indirect→direct 昇格 + jsonschema/yaml 追加。redesign は独自依存ツリーを持ち go test 44 package green のため機能的欠落なし。security bump ではない。
- `benchmarks/breezing-bench/*/package-lock.json`: benchmark 用 Node 依存。production 配布物外。
- **Dependabot 21 脆弱性 (M6)**: main default branch 固有。redesign 本流化後に別途再評価する human-only 事項。本 merge では扱わない。
