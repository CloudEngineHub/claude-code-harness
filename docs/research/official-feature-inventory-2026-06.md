# Official 3-CLI Feature Inventory (2026-06)

Phase 93.2.1 research-only attestation. This inventory fixes **official host capabilities** against **Claude Code Harness (CCH)** overlap and adoption posture.

## Evidence sources (repo-first)

| Source | Used for |
|--------|----------|
| `docs/CLAUDE-feature-table.md` | Claude Code versions, hooks, Agent Teams, Monitor, worktree, `/loop` |
| `docs/agent-view-policy.md` | `claude agents` (agent view) Research Preview + flags |
| `docs/research/cursor-adapter-candidate.md` | `cursor-agent` CLI flags, stream-json, `--mode ask`, permissions model |
| `.claude/rules/cursor-cli-only.md` | Cursor companion policy, permissions.json, read/write topology |
| `.claude/rules/codex-cli-only.md` | Codex companion, app-server, subagent native APIs |
| `docs/upstream-update-snapshot-2026-04-25.md` | Codex `0.124.0` stable hooks + app-server |
| `docs/upstream-update-snapshot-2026-05-03.md` | Codex `0.128.0` plugin-bundled hooks, MultiAgentV2 |
| `docs/upstream-update-snapshot-2026-05-10.md` | Codex `0.130.0` app-server Thread pagination, ThreadStore resume/fork |
| `spec.md` | `hosts.toml` / `hostgen`, tri-host `PreToolUse` convergence, support tiers |
| `hosts.toml` | Per-host hook event, matcher, deny mechanism |
| `docs/long-running-harness.md` | `/loop` + `ScheduleWakeup` dynamic workflow contract |

**Attestation rule:** `not_observed != absent`. Rows without repo version or GA evidence are marked **unverified**; we do not infer official absence.

## Inventory (5 columns)

| 機能名 | 最低バージョン | GA/preview | CCH 重複 | 採用・置換・捨てる |
|--------|----------------|------------|----------|-------------------|
| **Claude Code — Agent Teams** | CC `v2.1.71+`（`CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS` 要） | preview | `breezing` / `TeammateIdle`・`TaskCompleted` hooks / `docs/team-composition.md` | **採用**（breezing 前提）。GA 時置換抽象化: `hosts.toml` + hostgen の teammate spawn 記述と task-dependency モデルに寄せ、実験フラグ依存を削る |
| **Claude Code — Channels** | unverified（本 repo に公式バージョン根拠なし） | unverified | Phase 92.6 `livemsg` / Session Coordination broadcast 設計（`spec.md`） | **捨てる**（本 Phase）。公式 Channels 仕様が repo に固定されるまで CCH 独自 channel を増やさない |
| **Claude Code — agent view (`claude agents`)** | CC `v2.1.139+`（`--json` は `v2.1.145+`） | preview | `docs/agent-view-policy.md`（operator 専用。breezing teammate spawn と分離） | **採用**（operator 監視のみ）。GA 時置換抽象化: `orchestration-ledger` + scorecard を一次 UI とし、`claude agents` は diagnostic fallback に降格 |
| **Claude Code — dynamic workflows (`/loop`, Cron, ScheduleWakeup)** | CC `v2.1.71+` | GA | `skills/harness-loop` / `docs/long-running-harness.md` | **採用**。GA 時置換抽象化: wake 契約（pacing・max-cycles・resume-pack）を host 非依存の harness-loop スキル契約として維持し、スケジューラ API 名だけ差し替え |
| **Claude Code — hooks（PreToolUse / PostToolUse / Stop / SessionStart）** | CC `v2.1.50+` 系（継続拡張。R01-R13 は `v2.1.77+` deny 尊重） | GA | `hooks/hooks.json` + `bin/harness hook *` + `go/internal/policy` | **採用**（正本）。他 host の PreToolUse は **置換**先（hostgen が同一 kernel へルーティング） |
| **Claude Code — hooks（PreCompact / PostCompact）** | PreCompact `v2.1.105+`、PostCompact `v2.1.76+` | GA | `pre-compact-save` / WIP 警告 agent hook / compaction event ledger | **採用**。長時間 Worker の compaction 安全弁として維持 |
| **Claude Code — Monitor tool** | CC `v2.1.98+`（`monitors` manifest `v2.1.105+`） | GA | CI/デプロイ進捗追跡（Feature Table）、Phase 92.6.2 delivery `monitor` 経路 | **採用**（CC のみ）。Codex/Cursor は Monitor 無し → `turn` fallback（Plans 92.6.2） |
| **Claude Code — worktree isolation** | CC `v2.1.50+`（`isolation: worktree` / `EnterWorktree`） | GA | `.harness-worktrees/`（並列タスク）+ `.claude/worktrees/`（CC ライブ隔離。`spec.md` で分離） | **採用**。CCH 並列 root は `.harness-worktrees/` を正本とし、CC ネイティブ worktree はエージェント隔離専用 |
| **Cursor — lifecycle hooks（1.7+ 公式ドキュメント想定）** | unverified（repo は `cursor.com/docs/agent/hooks` のみ。`1.7+` 表記は Plans 93.2.1 要件） | unverified | `hostgen` → `.cursor/hooks.json` `preToolUse` → `bin/harness hook pre-tool --host cursor` | **採用**（candidate tier 内）。GA 時置換抽象化: `hosts.toml` の event/matcher/deny 列だけ更新し、R01-R13 本体は不変 |
| **Cursor — Stop hook `followup_message`** | unverified（repo に公式フィールド根拠なし） | unverified | CC `Stop` / `TeammateIdle` の `continue:false` パターン（別メカニズム） | **捨てる**（未確認のまま依存しない）。確認後は hostgen の stop 応答マッピング層で吸収可能 |
| **Cursor — `cursor-agent --output-format stream-json`** | CLI `2026.05.28-a70ca7c`（local `--help` 確認） | GA | `scripts/cursor-companion.sh` Route A、exit-code-first 契約 | **採用**（headless 委譲）。最終 `.result` のみ使い、ストリームは進捗 UX 用に限定 |
| **Cursor — `--mode ask`（read-only hard stop）** | CLI `2026.05.28+`（spike 確認: `--force` でも write 不可） | GA | `skills/cursor-ask` / `.claude/rules/cursor-cli-only.md` lean path | **採用**（調査・second-opinion）。write は `cursor-do` / `breezing --cursor` へ分離 |
| **Cursor — `~/.cursor/permissions.json` allowlist** | unverified（schema は Cursor docs 引用。CLI 版は未固定） | GA | なし（CCH は worktree + Lead review + FLOOR を境界とする） | **採用**（利便性のみ）。**置換不可**: security boundary として使わない（Cursor 公式も best-effort） |
| **Cursor — `cursor-agent -p --trust --workspace`** | CLI `2026.05.28+` | GA | `cursor-companion.sh` 必須フラグ、`*.cursor.sh` egress allowlist | **採用**。`--workspace` は CWD ヒントのみ。**置換**: 実効封じ込めは専用 `.git` worktree + fingerprint + cherry-pick |
| **Codex — app-server protocol** | Codex `0.124.0` stable+（multi-environment sessions） | GA | `scripts/codex-companion.sh` → 公式 `codex-plugin-cc` app-server | **採用**（companion 経由）。hand-rolled JSON-RPC は **捨てる** |
| **Codex — subagents（`spawn_agent` / `wait_agent` 等）** | unverified（`codex-cli-only.md` が skills-codex 例外として言及） | unverified | Lead 経路は `codex-companion.sh task`（raw spawn 禁止） | **採用**（`skills-codex/` 内のみ）。Lead/breezing は companion **置換**を維持 |
| **Codex — PreToolUse hook（Harness 観測: Bash 中心、公式は MCP/`apply_patch`/Bash）** | Codex `0.124.0` stable hooks | GA | `hosts.toml` `[codex]` matcher `*` → `bin/harness hook pre-tool --host codex`（R01-R13 重複） | **採用**（hostgen）。Plans 93.1.2 の再昇格条件どおり非 Bash イベントは **未採用**（将来 hostgen matcher 拡張で置換） |
| **Codex — resume thread（`--resume-last` / ThreadStore）** | Codex `0.130.0` stable（ThreadStore resume/fork 改善） | GA | `codex-companion.sh --resume-last`；cross-session は `harness-mem resume-pack`（別系統） | **採用**（同一 Codex スレッド継続）。セッション横断は mem に **置換**しない |
| **Codex — plugin-bundled hooks** | Codex `0.128.0` stable+ | GA | Harness no-inline-hooks 方針（upstream snapshot） | **採用**（表示・配布判断のみ）。inline hook 推測生成は **捨てる** |

## Row counts

- Claude Code: 8 rows
- Cursor: 6 rows
- Codex: 5 rows
- **Total data rows: 19**（ヘッダ除く `^|` 行数は 21）

## Notes

- **CCH 重複** means Harness already implements or routes the capability; adoption chooses official surface vs kernel abstraction.
- **preview 採用** rows include one-line GA replacement abstraction in the 採用 column.
- Cursor remains support tier **candidate** / **internal-compatible** per `spec.md`; this inventory does not promote public `supported` claims.

---

> 本 inventory は手動 attestation であり CI 検証対象外。harness-mem への保存は Lead が統合時に実施する。
