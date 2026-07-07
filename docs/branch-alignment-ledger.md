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
