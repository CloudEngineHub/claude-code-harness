# Zero-Base Redesign — 0 ベース再設計提案（最終モデル）

Status: **実装完了** — Phase 91.1–91.8 を本 repo に実装（2026-06-06）。配布モデルは当初スケッチの「manifest/mirror を untrack + gitignore + install 時生成」から **generated-and-committed + CI drift-check** に着地した。理由: Claude plugin marketplace は repo を clone して `.claude-plugin/*` を直接読み、Codex/OpenCode の setup script は committed mirror を copy する ＝ install 時に `harness gen` を走らせる経路が無く、untrack すると配布が壊れる。権威ある契約は root `spec.md` の Host Distribution Contract（本書の「gitignore 生成物」記述は当初案の歴史記録）。
Base commit: v4.14.0 (`4c26acd9`)
Method: 18-agent multi-perspective workflow（4 独立案 → 2 lens scoring → synthesis）＋ owner レビュー 4 往復で収束
Confidence: **high**
相対側: [cursor-adapter-candidate.md](./cursor-adapter-candidate.md)（ACP not-adopted の旧判断。本書はそれを「native hook 収束により ACP は enforcement 不要」へ更新）

> テーマ: 当時の Claude Code に合わせて作ったものを、CC のアプデ・Codex/Cursor 対応・状況変化を踏まえて、今 0 ベースで作り直すなら？
> owner 制約: 後方互換性・段階的導入は不要 / プラグイン層を脱して CLI 化してよい / ACP は不要。

---

## 結論（最終モデル）

**プラグインを脱して単一 CLI バイナリ `harness` にする。** 構成は **1 モデル**:

> **生成した skill/agent ＋ 単一 harness CLI ＋ 各 host の native hook（全部が同じ `bin/harness hook pretool` を呼び、deny は exit 2 で統一）＋ 全 host 共通の FLOOR。ACP は不採用。**

- value-bearing な Go 約 20%（R01-R13 policy engine ほか）は verbatim で残す。
- host 結合の約 80%（4 manifest・mirror 木・95 hookhandler・7-way version sync）は殺す。
- host 差分は `hosts.toml` 1 枚。各 host の薄い shim（skill/agent/hooks.json）は `harness gen` で生成（gitignore）。
- **enforcement は 3 host とも native の事前ブロック**（Claude/Codex/Cursor すべて `PreToolUse` 系 hook を持つ）。ACP も companion 丸投げも enforcement には不要。

---

## なぜこの結論か（判断軸）

今の構造の「原罪」は **R01-R13 ガードレールが CC の hook bus にボルト留め**され、Claude だけが守られること。当初の再設計案はこれを ACP（per-action gating の共通プロトコル）で均そうとした。

**しかし owner レビューと 3 つの公式 doc 確認で、ACP は不要と確定した。** Claude / Codex / Cursor の 3 host が **いずれも CC とほぼ同一の native hook（`PreToolUse` 系、`permission deny` / exit 2、`hooks.json`）に収束済み**だから。修正は ACP ではなく:

> **各 host の native hook を「同じ `bin/harness hook pretool` を呼ぶ」よう生成するだけ。1 つの policy engine が 3 host を adjudicate する。**

これは「複製ではなく routing」（1 engine を複数 trigger から呼ぶ）の最もシンプルな実現であり、ACP/プロトコルの新規実装を一切要しない。

---

## 更新履歴（2026-06-05 収束）

本書は当初「C のスパイン + A の ACP opt-in seam」を推奨したが、owner レビュー 4 往復で **ACP 不採用・native hook 収束**へ更新された。経緯:

| 往復 | owner の指摘 | 設計への反映 |
|---|---|---|
| 1 | ACP は不要 | ACP を opt-in seam に降格（当初） |
| 2 | Codex/Cursor「**から**使う」運用がある（host→harness の向き） | direction #2 を一級市民化。#1（harness が駆動）と分離 |
| 3 | 全部「生成 skill/agent + harness CLI」で統一できないか。Codex に Agent も hook もある | 3-tier enforcement は分けすぎと認め、1 モデルへ |
| 4 | 公式 doc を確認しろ | **Codex/Cursor とも native pre-action hook を確認 → ACP を enforcement から完全に外す（不採用）** |

確認した公式 doc（2026-06-05）:
- developers.openai.com/codex/hooks
- cursor.com/ja/docs/hooks
- （Claude Code hooks は既存実装で確認済み）

---

## 3 host の hook が同一モデルに収束（本設計の根拠）

| | Claude Code | Codex | Cursor |
|---|---|---|---|
| 事前ブロック event | `PreToolUse` | `PreToolUse` | `preToolUse`（＋ `beforeShellExecution` / `beforeMCPExecution` / `beforeReadFile`） |
| deny 手段 | `permissionDecision:"deny"` / **exit 2** | `permissionDecision:"deny"` / **exit 2** | `permission:"deny"` / **exit 2** |
| 設定ファイル | `hooks.json` | `hooks.json` | `hooks.json` |
| 配置 | `.claude-plugin/` | `~/.codex/`・`<repo>/.codex/` | `~/.cursor/`・`<repo>/.cursor/` |
| 横取り対象 | Bash/Edit/Write/MCP | Bash/`apply_patch`/Edit/Write/MCP | shell/file/read/MCP |
| 任意 command hook | ✅ | ✅ | ✅ |
| stdin（差分） | `session_id`, `tool_name`, `tool_input` | `session_id`, `tool_name`, `tool_input`, `tool_use_id` | `conversation_id`, `tool_name`, `command`/`file_path` |

**共通分母 = 「`type:command` の hook が exit code 2 で deny できる」。** これが 3 host で効くため、host ごとの出力 JSON 差すら吸収不要（`bin/harness hook pretool` が deny 時に exit 2 する）。残る差（`permissionDecision` vs `permission`、`session_id` vs `conversation_id`、event 名 casing）は **harness hook 側の薄い stdin codec 1 枚**で正規化し、rule engine 本体は不変。

---

## 採用モデルのアーキテクチャ

図: [redesign-visuals/after.svg](./redesign-visuals/after.svg)（PNG・before/after・HTML も同フォルダ）

```
   you ── harness work 1.1.1 / plan / review / release / gen / doctor ──▶ harness（単一CLIバイナリ）

   ┌──────────────── hosts.toml（host差分はこの1枚）──▶ harness gen ──▶ 各host の shim を生成（gitignore）────────────┐
   ▼                                   ▼                                   ▼
 ┌──────────────┐                ┌──────────────┐                  ┌──────────────────────┐
 │ Claude Code   │                │ Codex        │                  │ Cursor                │
 │ 生成 skill/agent│                │ 生成 skill/agent│                  │ 生成 skill/agent       │
 │ native hook:  │                │ native hook: │                  │ native hook:          │
 │ PreToolUse    │                │ PreToolUse   │                  │ preToolUse (+before*) │
 └──────┬───────┘                └──────┬───────┘                  └──────────┬───────────┘
        │  deny=exit2                   │  deny=exit2                          │  deny=exit2
        └───────────────────────────────┴──────────────────────────────────────┘
                                         ▼  （3 host が native 合流）
                       ┌───────────────────────────────────────────────┐
                       │  bin/harness hook pretool ＝ 単一の enforcement 面 │
                       │  → R01–R13 policy engine（核・約80%再利用・stdlib）│
                       └───────────────────────┬───────────────────────┘
                                               ▼  ★ FLOOR（全 host 共通の最終 backstop）
                       ┌───────────────────────────────────────────────┐
                       │  worktree + cherry-pick で harness policy check 再評価 │
                       │  → nested subagent / in-proc shell の穴を埋める      │
                       └───────────────────────────────────────────────┘
  ACP: 不採用（native hook で per-action gating 達成済み）
  companion 丸投げ: 「harness が他ツールを engine として駆動する」時だけ（direction #1 専用）
```

### 2 つの向きを両方サポート

| 向き | 誰が主役 | harness の立場 | enforcement |
|---|---|---|---|
| #1 harness が駆動 | harness | 司令塔（engine を spawn） | companion + FLOOR |
| **#2 host から使う**（Codex/Cursor「から」使う） | host | 生成 skill/agent ＋ `harness` CLI ＋ host の native hook | **native 事前ブロック ＋ FLOOR** |

#2 は当初案で過小評価していたが、3 host とも native hook を持つため **フル enforcement の一級市民**。

---

## 7 つのコア判断（更新版）

| # | 判断 | 選択 | confidence |
|---|------|------|-----------|
| 1 | Go コア再利用 vs 再構築 | **Kernel-EXTRACTION**。`go/internal/guardrail` を stdlib-only `policy/` に verbatim 昇格。membridge(244 LOC)・config・operator CLI・trimmed state を再利用。dead な breezing.Orchestrator を resurrect | high |
| 2 | ACP を採用するか | **不採用**。3 host が native pre-action hook を持つため、ACP の enforcement 価値はゼロ。残る価値は live streaming UX のみ（owner 不要と明言）。将来 streaming が欲しくなったら opt-in 再検討 | high |
| 3 | uniform enforcement の実現 | **複製ではなく routing**。各 host の native hook が同じ `bin/harness hook pretool` を呼ぶ。deny は exit 2 で統一。3 host とも native 事前ブロック | high |
| 4 | mirror/manifest/version-sync 税 | **`harness gen`** が全 host artifact（skill/agent/hooks.json/manifest version/docs）を `hosts.toml`+promptpack から生成。gitignore・git tag 1 個 = version | high |
| 5 | plugin 層を脱するか | **near-total**。一次配布 = 単一 CLI binary。manifest と hooks.json は生成 build-artifact。各 host の native hook 経由で in-proc 事前ブロックは全 host で維持 | high |
| 6 | orchestration / 並列 | **Go-native**。breezing.Orchestrator を resurrect、N 個の flat sub-run。harness が全 fan-out を所有し backend は single-threaded（nested 問題を「N flat seam」に還元） | high |
| 7 | Plans.md モデル | agent-read markdown SSOT のまま。Go は marker-tally drift heuristic のみ | medium |

---

## 何が死に、何が生まれるか

### 死ぬもの

- 4 manifest を手書き source として持つこと（→ `harness gen` の gitignore 生成物）
- skill mirror 基盤（codex/.codex/skills 183・opencode/skills 182・skills-codex・sync-skill-mirrors.sh・build-opencode.js ≈365 file）→ on-install 生成
- 7-way version sync（sync-version.sh ほか）→ git tag 1 個
- 両 companion script を「enforcement の主経路」とすること（companion は direction #1 専用に降格、`companion-result.v1` に正規化）
- breezing.Orchestrator の dead code 状態（resurrect）。breezing skill(445L)→ harness-work `--team` mode
- 27-event hook surface の大半 + proprietary-event→AdditionalContext glue 系 hookhandler（intent は kernel service 化）
- sub-5ms hook-fastpath budget を第一制約とすること（長命/CLI process では無意味）
- migration/self-audit scaffolding 常設（check-residue.sh・deleted-concepts.yaml・self-audit.md）→ deny-list は policy/ に畳む
- version-pinned 上流追従 rule file（hooks-2.1.139/152-plus.md 等）→ **capability-detection 層**（公式 doc 確認で stale が出た反省: 能力は runtime/doc 確認で持つ）
- ~22 skill accretion（auth/crud/ui/deploy・media gen・session cluster・coaching cluster）
- drifted/dangling docs（CLAUDE-skill-catalog.md・scaffolder 参照）
- **ACP adapter（当初案の opt-in seam）→ v1 から削除**。native hook 収束で不要

### 生まれるもの

- 単一配布 binary `harness`。kernel = policy/（R01-R13・deny-list self-audit 同梱）・plans/・membridge/・scanner/・gitport/（17 exec.Command を 1 seam）・state/（trim）・orchestrator/（resurrect）・embedded promptpack
- prompt pack 1 本 = skill+agent の irreducible SSOT
- `hosts.toml` — host-capability descriptor。host ごと: tools-available・hook-event 名（`PreToolUse`/`preToolUse`）・deny 手段・transport・model/effort
- **`harness gen`** — 全 host artifact を生成（**各 host の hooks.json も生成**：Claude=`.claude-plugin/hooks.json`、Codex=`.codex/hooks.json`、Cursor=`.cursor/hooks.json`、すべて `bin/harness hook pretool` を指す）。`--check` で golden fixture と CI diff
- **`harness hook pretool`** — 3 host 共通の単一 enforcement 面。stdin codec で host 差（`permissionDecision`/`permission`、`session_id`/`conversation_id`）を正規化、deny は exit 2
- **`harness policy check`** — cherry-pick FLOOR と hook の両方が呼ぶ R01-R13 surface
- companion-result.v1 — direction #1（harness が駆動）の統一 return envelope
- self-verifying policy ruleset（deny-surface baseline hash、弱体化で起動拒否）
- opt-in cognitive-load extension pack（plan-brief/progress/accept）を別 installable に
- capability-detection 層（host の hook event 名・対応可否を runtime/doc 由来 data で持つ）

---

## 段階スケッチ（更新版・ACP phase なし）

※ owner 制約により big-bang cutover（後方互換なし）。各 phase は green を保つ単位。

0. **Kernel extraction（挙動変更なし）**: `guardrail`→`policy/`（stdlib+regexp）、membridge verbatim、17 exec.Command を `gitport/` 1 seam、state/ trim。rules_test.go を green 維持＝regression anchor。
1. **`harness` CLI スパイン**: policy/+plans/+membridge/+state/+gitport/ を `harness plan|work|review|release|sync|policy check|hook|doctor|mem` の裏に。prompt pack embed。Claude 上で headless にループ可。dogfood。
2. **`hosts.toml` + `harness gen`**: 単一 descriptor を定義、generator が **3 host の hooks.json**（全部 `bin/harness hook pretool`）+ skill/agent + manifest version + docs を emit。`--check`。生成物 gitignore。**これが mirror/version 税を消す**。
3. **3 host hook 配線 + stdin codec**: `harness hook pretool` に host 差正規化 codec（`permissionDecision`/`permission`、`session_id`/`conversation_id`、event 名）。`.codex/hooks.json`・`.cursor/hooks.json`（現状カラ）を生成・配線。3 host とも native 事前ブロックが効くことを smoke で確認。
4. **orchestrator + companion(direction #1) path**: breezing.Orchestrator を resurrect（N flat sub-run）。companion を `companion-result.v1` に正規化（harness が他ツールを engine として駆動する時のみ）。breezing は --team mode に。
5. **FLOOR 硬化**: cherry-pick diff への `harness policy check` + contract-grep（test-support-claim-wording.sh + check-consistency.sh + validate-plugin.sh）を全 backend の必須 pre-merge に。deny-list self-audit を policy/ に畳む。
6. **prune + docs 再生成**: ~22 accreted skill、migration/self-audit scaffolding、version-pinned rule file（→capability-detection）を削除。`harness gen docs` が CLAUDE.md/catalog を再生成。
7. **cutover 検証**: harness repo 自身に `harness work` で 1 full Plan→Ship cycle。通過後にのみ旧 manifest/mirror を committed source から削除。

（旧 Phase 5「opt-in ACP adapter」は削除。native hook 収束で不要。）

---

## owner が決めるべき open question（更新版）

| # | 問い | 状態 / 推奨 |
|---|------|------|
| Q1 | IDE 内で動く必要 vs headless | **解決**: host 常駐（#2）と headless（#1）は **両方一級**。3 host とも native hook で動く。「headless-first」の旧推奨は撤回 |
| Q2 | launch 時の first-class backend | Claude + Codex + Cursor を v1（3 host とも native hook あり）。Gemini optional |
| Q3 | per-action gating は要るか | **解決**: native hook で 3 host とも事前ブロック可能。ACP 不要。FLOOR が共通 backstop |
| Q4 | ~22 accreted skill | 全削除（owner の brand-doc skill は ~/.claude global にある） |
| Q5 | self-bootstrap | 現 harness(v4.14.0) で v1 build → 自己 dogfood してから旧構造削除。rules_test.go を throughout green |

残る判断ポイントは Q2/Q4/Q5（scope と削除範囲）のみ。ACP / enforcement model は確定。

---

## 異論・pushback（更新版）

1. **（旧異論 #1「ACP は over-specified」→ 強化されて確定）**: owner の「ACP 不要」は正しく、3 公式 doc で技術的にも裏付けられた。cross-tool coordination は ACP を要さず「1 orchestrator + 各 host native hook が同じ policy engine に合流」で達成。ACP が足し得たのは live streaming UX のみで、それも owner 不要。→ **ACP は v1 から完全に外す**。
2. **kernel-extraction over rebuild は維持**: value-bearing core（EvaluateRules pure・membridge 244 LOC）は ~20% で tested。捨てて作り直すのは最も間違えやすい所に最大コストを払う。tax の ~80% を削り IP を残す。
3. **自己異論（残る最大リスク）= big-bang cutover**: v4.0.0「Hokage」TS→Go 移行も clean one-shot を謳い 13 件の residue を残した（check-residue.sh が生まれた理由）。本プロジェクトの歴史が「big-bang は漏れる」と言っている。緩和は Phase 7 の自己 dogfood を hard gate にすること。並走（dual-maintenance）は redesign が殺す対象なので採らない。
4. **新たな注意（stale risk）**: 私は当初 repo 内メモ（codex/AGENTS.md「Hooks 未対応」）を信じて Codex hook を弱いと誤答した。**外部ツールの能力は公式 doc で都度確認すべき**。これは「version-pinned rule file を capability-detection に置換」の動機そのもの。

---

## 検討した 4 案とスコア（決定の provenance）

2 lens（pragmatist / visionary）× 6 軸 30 点満点。

| 案 | 一言 | 合計 |
|---|---|---|
| **A** CLI-first ACP Orchestrator（harness=ACP CLIENT、全 backend を ACP で駆動） | 52 |
| **C** Kernel + Prompt Pack（engine 再利用 + 生成 shim、~80% Go 再利用） | 50 |
| B Protocol-native Hub（ACP-on-both-edges、daemon） | 47 |
| D Harness-as-Agent（自前 agent loop、API 直叩き） | 44 |

当初は「C のスパイン + A の ACP seam」を採用したが、**owner レビューで A の ACP 部分が native hook 収束に置き換わり、実質「C のスパイン + 3 host native hook」へ収束**。A の per-action gating の狙いは ACP ではなく各 host の native hook で無料で達成された。

---

## 検証された外部事実（賭けの根拠）

- `EvaluateRules` は first-match-wins、import は stdlib+regexp+hookproto のみ → trigger を hook-stdin から各 host hook に向けるだけで rule table 不変。
- `breezing.Orchestrator`(333 LOC) は test 外から呼ばれない dead code（resurrect 対象）。
- **3 host とも native pre-action hook を持つ（公式 doc 確認）**: Claude `PreToolUse` / Codex `PreToolUse` / Cursor `preToolUse`、いずれも deny + exit 2、hooks.json、任意 command 実行可。
- **exit code 2 = deny が 3 host 共通** → host 差を吸収する単一 enforcement 面が作れる。
- anthropics/claude-code#27203: nested sub-agent は gating を bypass → FLOOR が全 host で必要な理由。

---

## 関連

- [redesign-visuals/](./redesign-visuals/) — before/after 図（SVG/PNG）＋説明 HTML
- [cursor-adapter-candidate.md](./cursor-adapter-candidate.md) — ACP not-adopted の旧判断（本書が native hook 収束で更新）
- `go/DESIGN.md` / `go/SPEC.md` — 現行 v4 Go アーキ（extraction 元）
- `.claude/rules/migration-policy.md` — exclusion-based verification（big-bang 漏れ対策、異論 #3 の根拠）

Sources（2026-06-05 確認）: developers.openai.com/codex/hooks ／ cursor.com/ja/docs/hooks ／ Claude Code hooks（既存実装）
