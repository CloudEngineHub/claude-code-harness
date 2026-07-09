# Phase 110: Post-v5.0.0 Operator Closeout

**Status (2026-07-09):** executed (110.0 skipped by operator — no confirmed leak)

| Task | Status |
|------|--------|
| 110.0 token rotation | skipped (no confirmed leak; `/.codex/` already gitignored) |
| 110.1 tree align | done — `archive/local-main-pre-v5` @ ed6b18c9, main @ c100dbb4+ |
| 110.2 109.5 marker | done in Plans.md |
| 110.3 HOTL + EN | done — A research draft (`docs/research/README_HOTL_ja.md`), EN=No |
| 110.4 Dependabot | done triage — #232/#234/#230 auto-merge queued; #233 deferred; vitest alerts accept-risk (bench-only) |
| 110.5 salvage | done — all drop, archive retained |
| 110.6 closeout | done with this commit |

---


作成日: 2026-07-09  
前提: v5.0.0 "Zero-Base Redesign Mainline" 公開済み  
Release: https://github.com/Chachamaru127/claude-code-harness/releases/tag/v5.0.0  
origin/main: `c100dbb4`  
Plans 正本: **origin/main の Plans.md**（local main は pre-redesign 樹で 15 ahead / 249 behind — 正本ではない）

## Spec skip reason

Product behavior / API / 配布 contract を変えない operator hygiene + ledger sync。
- 110.1–110.2: git/workspace alignment + marker（task ledger only）
- 110.3: secret rotation（ops、code 変更なし）
- 110.4: docs owner judgment（README_HOTL claim surface — 公開 claim を変える場合のみ後続で Spec delta）
- 110.5: Dependabot triage（deps、redesign 後の再評価）
- 110.6: optional local-value salvage（port が product に触れるなら別 Phase に昇格）

`team_validation_mode: manual-pass`（Product / Architecture / Security / QA / Skeptic を単独で分離評価済み。根拠は本ファイル「検証メモ」）。

---

## 進捗サマリー（harness-sync 2026-07-09）

| 面 | 状態 |
|----|------|
| GitHub Release v5.0.0 | 公開済み・draft=false・asset 4 |
| origin Plans 109.0–109.4 | cc:done |
| origin Plans **109.5** | **cc:todo のまま（drift）** — 実作業は完了 |
| local `VERSION` | 4.16.4（stale） |
| origin `VERSION` | 5.0.0 |
| local Plans.md | Phase 96 まで（redesign Phase 91–109 なし） |
| local branch | main, **15 ahead / 249 behind** origin/main |
| working tree | dirty: model-routing Sonnet5 / agents / README HOTL 追記 / binaries / Plans 94.4 marker / `.gitignore` `/.codex/` / untracked `README_HOTL_ja.md` |
| Dependabot alerts | open **21**（vitest CRITICAL 系が大半 + @ai-sdk/provider-utils LOW） |
| Dependabot PRs open | #230 #232 #233 #234（+ #228 Phase97 は別件） |
| Secrets | `.codex/environments/environment.toml` に GH_TOKEN / FIRECRAWL_API_KEY 平文（**git 非追跡**。working-tree `.gitignore` で `/.codex/` 追加済み未 commit） |

### Drift 検出表

| Task | 現在 (origin Plans) | 変更後 | 理由 |
|------|---------------------|--------|------|
| 109.5 | cc:todo | cc:done [c100dbb4 / v5.0.0] | PR #235+#236 merge, tag, 4 assets, EN notes 済 |
| 94.4 (local only) | cc:wip → dirty cc:done | **discard with local tree** | origin redesign 樹に 94.4 は別 lineage。local closeout は 109.1 port 対象 |

### ローカル 15 commit の判定（推奨）

`git cherry` は 15 全て `+`（patch-id 未取込）。ただし **109.1 が guard 4 本を機能 port 済み**と台帳記載。skill prune / progressive disclosure は redesign 線の skill 木（例: origin `harness-work` ~1010 行 vs local ~509 行）と衝突しやすい。

| 群 | commits | 推奨 |
|----|---------|------|
| guard Phase94 closeout | 462e94ff…ed6b18c9 | **drop**（109.1a port 済み。差分が残るなら 110.6 で spot-check） |
| skill prune / progressive disclosure | fa88d4cf…7405c30d + d199ef91 | **archive then drop by default**。必要なら origin 上で再実装 |
| wip HOTL wiring | f6cdb042 | **salvage 候補**（plan-brief/accept/progress）。origin に skill は既にあるため diff 単位で要否判断 |
| merge skill-prune | 8ca82ced | drop with parents |
| dirty uncommitted model-routing | working tree | **drop**（origin 既に sonnet-5） |
| dirty README HOTL 3 画面追記 | working tree | f6cdb042 とまとめて 110.4/110.6 |
| untracked README_HOTL_ja.md | working tree | **keep as review artifact**（110.4） |
| `.gitignore` `/.codex/` | working tree | **keep and commit on origin tree**（110.3 とセット） |

### 振り返り（v5.0.0）

| 指標 | 値 |
|------|-----|
| 完了マイルストーン | Phase 109 release done |
| ブロック | release preflight clean-checkout 偽 green（生成物残存） |
| 学び | 生成物 assert は materialize-then-verify（FACT-4）。$HOME plugin fallback も同型 |
| 次に活かす | release-preflight / 生成物テストは clean checkout 前提を固定 |

---

## Phase 110 タスク（origin Plans.md へ適用）

適用タイミング: **110.1 完了後**に origin/main 上の `Plans.md` へ append + 109.5 marker 更新。

| Task | 内容 | DoD | Depends | Status |
|------|------|-----|---------|--------|
| 110.0 | `[lane:gate]` `[tdd:skip:ops]` **token rotation（P0）**。`.codex/environments/` に平文の `GH_TOKEN` / `FIRECRAWL_API_KEY` がある。GitHub PAT と Firecrawl key を **revoke → 再発行 → 新値を env 注入**（ファイルに平文を残さない）。旧値が agent log / session transcript に載った可能性を前提に扱う | (a) 旧 GH PAT が GitHub 上で revoke 済み、(b) 旧 Firecrawl key が invalidate 済み、(c) 新 secret は shell env / secret manager のみ（repo 内平文 0）、(d) `/.codex/` が `.gitignore` で無視されることを origin 樹で commit | - | cc:TODO |
| 110.1 | `[lane:gate]` `[tdd:skip:git-ops]` **local main を origin/main (`c100dbb4`) に揃える**。推奨: (1) dirty を `git stash -u` または `archive/local-main-pre-v5` ブランチに退避、(2) `git branch archive/local-main-pre-v5 HEAD`、(3) `git switch main && git reset --hard origin/main`、(4) stash から **README_HOTL_ja.md** と **`.gitignore /.codex/`** のみ再適用。15 commit の丸ごと rebase は **非推奨**（249 commit redesign との衝突コスト過大） | (a) `git rev-parse HEAD` == `origin/main`、(b) `VERSION` == `5.0.0`、(c) Plans.md に Phase 109 が存在する、(d) archive branch が残る、(e) 作業ツリーに意図しない binary/model-routing dirty が無い | - | cc:TODO |
| 110.2 | `[lane:gate]` `[tdd:skip:ledger]` **109.5 marker を cc:done に**。evidence: PR #235 `970dff0d`, PR #236 `c100dbb4`, tag `v5.0.0`, assets=4, notes EN+Before/After。Phase 109 Purpose に release URL を 1 行追記可 | (a) 109.5 Status が `cc:done [c100dbb4]`、(b) DoD (a)(b)(c) を満たす evidence が Status or Purpose に残る、(c) `rg '109\.5.*cc:todo' Plans.md` が 0 | 110.1 | cc:TODO |
| 110.3 | `[lane:release]` `[tdd:skip:docs-only]` **README_HOTL_ja.md owner レビュー + EN 判断**。S0 で「言行不一致」指摘済み（Bridge Daemon / Decision Card 未配線 vs ✅ 表記）。判定: (A) 内部メモのまま gitignore/local-only、(B) 公開 README に昇格するなら claim を evidence 付きに落とす、(C) EN 版を作るか skip | (a) owner が A/B/C を 1 つ選んだ記録（decisions.md or Plans Status）、(b) B なら ✅/🔜 が origin 実装 evidence と一致、(c) EN 要否が Yes/No で確定、(d) 公開するなら `git diff --check` PASS | 110.1 | cc:TODO |
| 110.4 | `[lane:release]` `[tdd:skip:dependency-triage]` **Dependabot 再評価（21 alerts / open PRs）**。redesign 本流化後の default branch で: (i) vitest CRITICAL が `benchmarks/` 配下か production path か分類、(ii) #230/#232/#234 Actions bump は CI green なら merge 候補、(iii) #233 agent-eval major は bench 隔離のまま個別判断、(iv) alert を close or accept risk 記録 | (a) 21 alerts を path×severity で表に分類、(b) merge / defer / accept-risk が各 PR/alert に付く、(c) production runtime path の CRITICAL が 0 または issue 化、(d) Supply Chain Alert Contract と矛盾しない | 110.1 | cc:TODO |
| 110.5 | `[lane:fast]` `[tdd:skip:optional-salvage]` **local-only 価値の取捨**。archive 15 commit + stash を走査し、origin に無い差分だけ cherry-pick or 再実装。既定: guard 群 skip / model-routing skip / skill prune は origin redesign 優先で skip / HOTL wiring は 110.3 結果が B のときだけ port | (a) archive に対する keep/drop 表が Plans or mem に残る、(b) keep が 0 なら task を cc:done で close、(c) keep>0 なら focused test + review を DoD に追加した子 task を起票 | 110.1, 110.2 | cc:TODO |
| 110.6 | `[Closeout]` `[lane:release]` `[tdd:skip:test-aggregation]` Phase 110 closeout。release/tag は作らない | (a) 110.0–110.4 done（110.5 は skip 可）、(b) `bash tests/validate-plugin.sh` PASS on origin tree、(c) `git status` clean or 意図した untracked のみ、(d) GOD_plans §8 M6/M token 状態更新、(e) no tag | 110.0, 110.2, 110.3, 110.4 | cc:TODO |

---

## 実行順（推奨）

```
110.0 token rotation     ← 今すぐ（他と並列可、最優先）
110.1 tree align         ← Risk Gate: reset --hard（archive 必須）
110.2 109.5 marker       ← harness-sync 一発
110.3 HOTL README 判断   ← owner のみ
110.4 Dependabot triage  ← 機械分類は agent、merge は GO
110.5 salvage (optional)
110.6 closeout
```

### 非推奨

- local 15 commit を `git rebase origin/main` で全部載せる（衝突地獄、guard 二重化）
- dirty binary 4 本を origin に持ち込む
- token を再発行せず `.codex/environments` を commit する

---

## 検証メモ（manual-pass perspectives）

| 視点 | 所見 |
|------|------|
| Product | v5.0.0 は出荷済み。残は hygiene。HOTL README の ✅ は claim risk（S0 既知） |
| Architecture | Plans/SSOT 正本は origin redesign。local 4.16 樹は archive 対象 |
| Security | **token rotation P0**。Dependabot vitest は path 分類が先（bench 隔離なら accept-risk 可） |
| QA | 109.5 DoD は release URL + assets で満たす。validate-plugin は 110.1 後に再走 |
| Skeptic | 「14 commit を rebase」は見かけの作業量で、中身の大半は port 済み or 木が違う。reset+archive の方が安全 |

---

## 109.5 marker 更新用 1 行（110.2）

```markdown
| 109.5 | ... | ... | 109.4 | cc:done [c100dbb4] (PR #235+#236, tag v5.0.0, assets=4, EN notes 2026-07-09) |
```
