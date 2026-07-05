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

## Wave 2（Phase 105 P1）で対応予定 — deferred

| # | severity | finding | 参照 task |
|---|---|---|---|
| 13 | major | README_HOTL 節間の成熟度表示が自己矛盾 | 105.11 |
| 14 | major | bridge_events スキーマ知識が 3 ファイル DRY 違反 | 105.5 |
| 15 | major | Night Watch open-decision 検知が自 repo フォーマット不整合で恒久 0 件 | 105.4 |
| 16 | major | 非エンジニア 3 画面が本流フロー孤立 | 105.1 |
| 17 | major | i18n contract が Go hookhandler で広範に破綻 | 105.2 |
| 18 | major | Plan Brief confidence が誤解を招く合算 + 空配列 | 105.3 |
| 19 | major | harness-setup SKILL.md 460行 増築状態 | 105.10 |
| 20 | minor | README badge『Skills: 5 Verbs』乖離 | 105.10 |
| 21 | minor | spec.md 1462行 未分割 | 105.10 |
| 22 | minor | failurecodifier 部分文字列誤判定 | 105.6 |
| 23 | minor | 327MB worktree reap 漏れ + 検知無し | 105.7 |
| 24 | minor | runtime floor secret-read が文書テキスト内 .env 名に false positive | 105.8 |

## refuted（敵対検証で棄却、対応不要）— waived

| # | finding | 棄却理由 |
|---|---|---|
| 25 | 「同世代の他契約は実配線を確認、ドリフトは局所的」 | 肯定的所見（drift は全面ではなく局所と確認）。対応不要 |

## 未処理カウント

- fixed: 12（Wave 1 完了）
- deferred: 12（Wave 2 = Phase 105）
- waived: 1
- **未処理（状態未定）: 0**

Wave 2 完了時に deferred 12 件を fixed へ移し、本台帳の未処理を 0 に保ったまま S3 を閉じる。
