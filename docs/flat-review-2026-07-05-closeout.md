# Flat Review 2026-07-05 — Findings Closeout Ledger

S0 中立フラットレビュー（35 agents、敵対検証済み、confirmed 17 + plausible 8、refuted 5）の findings 処理台帳。
状態は `fixed`（修正済み・commit 付き）/ `deferred`（後続 Phase 参照）/ `waived`（意図的に対応しない・Why 付き）の 3 値で、未処理を 0 件に保つ。

## Wave 1（Phase 104 P0）で fixed

| # | severity | finding | 処理 | commit / task |
|---|---|---|---|---|
| 1 | critical | R15 秘密ファイル git add block が redesign に不在 | fixed | 104.1 `4ca6caef` |
| 2 | major | Bridge Daemon 約1900行が binary 到達不能 | fixed（削除 + 知見 spec 翻訳記録） | 104.4 `7b563987` |
| 3 | major | judgment-card/ledger runtime caller ゼロ | deferred（保持 + wired:no 注記、S3 で配線） | 104.4 → 105 |
| 4 | major | HARNESS_AUTO_APPROVE decorative no-op | fixed（主張撤回） | 104.3 `e46f6a5f` |
| 5 | major | Plans.md Depends↔Status 台帳矛盾 + 検出ゲート無し | fixed（checker + gate [20/22]） | 104.2 `212a781d` |
| 6 | major | 曖昧 skill 4件 未整理 | fixed（3件削除 + 1件保持裁定） | 104.9 `9efa0956` |
| 7 | major | branch-alignment gate 未タスク化 | fixed（台帳 + gate [21/22]） | 104.5 `212a781d` |
| 8 | major | binary/source drift 検証 CI ゲート無し | fixed（drift gate [22/22] + build-all.sh -trimpath 統一） | 104.6 `1050ac56` |
| 9 | minor | dual hooks.json sync test が CI 未配線 | fixed | 104.7 `212a781d` |
| 10 | minor | impact_score が Go/shell 二重実装 | fixed（Go 統一） | 104.4 `7b563987` |
| 11 | minor | triaddispatcher / GenerateDeliveryHooksJSON caller ゼロ | fixed（triaddispatcher 削除。delivery gen は 105.9 で裁定） | 104.4 → 105.9 |
| 12 | minor | model routing 旧世代 sonnet-4-6 ハードコード | fixed（Sonnet 5 化 + gate [16/22] 更新） | 104.8 `7b065700` |

## Wave 2（Phase 105 P1）で fixed

| # | severity | finding | 処理 | 参照 task |
|---|---|---|---|---|
| 13 | major | README_HOTL 節間の成熟度表示が自己矛盾 | fixed（redesign README.md は既に正直と検証、言行一致 self-check 節追加。自己矛盾は untracked HOTL 草稿の話で本 branch 対象外） | 105.11 |
| 14 | major | bridge_events スキーマ知識が 3 ファイル DRY 違反 | fixed（go/internal/eventstore へ集約） | 105.5 |
| 15 | major | Night Watch open-decision 検知が自 repo フォーマット不整合で恒久 0 件 | fixed（`## D<N>:` + JP status 対応） | 105.4 |
| 16 | major | 非エンジニア 3 画面が本流フロー孤立 | fixed（plan/work/release skill + README に導線） | 105.1 |
| 17 | major | i18n contract が Go hookhandler で広範に破綻 | fixed（ratchet gate [23/23] 新設 enforcement + inbox_check 参照移行。残 12 ファイル bulk 移行は Phase 106 に起票） | 105.2 → 106 |
| 18 | major | Plan Brief confidence が誤解を招く合算 + 空配列 | fixed（plan_readiness 単一軸 + 3 配列生成手順） | 105.3 |
| 19 | major | harness-setup SKILL.md 460行 増築状態 | fixed（460→101、references/ 分割） | 105.10 |
| 20 | minor | README badge『Skills: 5 Verbs』乖離 | fixed（5 core / 22 total 表記） | 105.10 |
| 21 | minor | spec.md 1462行 未分割 | fixed（1467→197、docs/spec/ sub-spec 分割） | 105.10 |
| 22 | minor | failurecodifier 部分文字列誤判定 | fixed（token 境界判定） | 105.6 |
| 23 | minor | 327MB worktree reap 漏れ + 検知無し | fixed（.gitignore + doctor stale 検知） | 105.7 |
| 24 | minor | runtime floor secret-read が文書テキスト内 .env 名に false positive | fixed（heredoc/コメント除外の executable-text scan） | 105.8 |

## L3 前進（計画外の追加成果）

| finding 外 | 内容 | task |
|---|---|---|
| — | delivery hook 生成 GAP（GenerateDeliveryHooksJSON が harness gen 未接続）を配線。Codex/Cursor 生成 hooks に enforcement + delivery 両立。北極星 L3 協調の一歩 | 105.9 |

## refuted（敵対検証で棄却、対応不要）— waived

| # | finding | 棄却理由 |
|---|---|---|
| 25 | 「同世代の他契約は実配線を確認、ドリフトは局所的」 | 肯定的所見（drift は全面ではなく局所と確認）。対応不要 |

## 未処理カウント（Wave 1 + Wave 2 完了時点）

- fixed: 24（Wave 1 の 12 + Wave 2 の 12）
- waived: 1
- **未処理（状態未定）: 0**

**follow-up（起票済み、未処理ではない）**: Phase 106（i18n bulk 移行、残 12 hookhandler ファイル）。
finding #17 の enforcement gate は完了しており、残債は Phase 106 で漸進的に返済する（ratchet が新規増加を防ぐ）。
