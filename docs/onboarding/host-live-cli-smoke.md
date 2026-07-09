# Live CLI smoke（正式対応 H4）— オペレーター用コピペ帳

最終更新: 2026-07-09  
対象: Phase 111 残り（`supported` / 正式対応に上げる前の **live H4**）

## ひとこと

**「その CLI で、本当に plan が 1 回回ったか」**を、あなたが手で確認する用。  
structural テスト（`test-host-workflow-smoke.sh`）とは別物。

---

## あなたが言ったら、AI がすぐ解釈するキーワード

結果を返すときは **下の 1 行フレーズだけ**でよい。

| あなたの発話（そのままコピペ可） | 意味 | AI が次にやること |
|----------------------------------|------|-------------------|
| `LIVE: claude PASS` | Claude live H4 合格 | 111.7 記録・必要なら昇格 PR 準備 |
| `LIVE: codex PASS` | Codex live H4 合格 | 同上 |
| `LIVE: cursor PASS` | Cursor live H4 合格 | 同上 |
| `LIVE: grok PASS` | Grok live H4 合格 | 同上 |
| `LIVE: claude FAIL: <理由>` | 不合格 | 原因分類（auth / skill 未発見 / 成果物なし） |
| `LIVE: all PASS` | 4 host 全部 live 合格 | 正式対応昇格可否を H1–H8 で再判定 |
| `LIVE: status` | 進捗確認 | どの host が PASS/未実施か一覧 |

**PASS の最低条件（全 host 共通）**

1. 対象 CLI が **スキルを認識**した（`/harness-plan` や skill 起動）  
2. **成果物**が残った（下の「成功の見た目」）  
3. 失敗理由が「モデルが雑」ではなく **Harness 経路**の問題なら FAIL  

成果物の置き場（推奨）:

```text
out/workflow-smoke/live/<host>/plan-artifact.md
```

---

## 0. 共通 prep（1 回）

CCH リポジトリで:

```bash
cd /Users/tachibanashuuta/LocalWork/Code/CC-harness/claude-code-harness

# structural は先に緑である前提
bash tests/test-host-workflow-smoke.sh --host claude
bash tests/test-host-workflow-smoke.sh --host codex
bash tests/test-host-workflow-smoke.sh --host cursor
bash tests/test-host-workflow-smoke.sh --host grok

mkdir -p out/workflow-smoke/live/{claude,codex,cursor,grok}
```

---

## 1. Claude Code（live）

### コピペ

```bash
cd /Users/tachibanashuuta/LocalWork/Code/CC-harness/claude-code-harness

# 成果物パス
export LIVE_OUT="$PWD/out/workflow-smoke/live/claude"
mkdir -p "$LIVE_OUT"

# Claude Code をこの repo で起動して、次をそのまま貼る:
```

Claude セッションに貼るプロンプト:

```text
/harness-plan
Live H4 smoke only.
Write ONE short plan for: "Add a single-line comment to README.md that says live-h4-claude".
Do NOT implement.
Save the plan markdown to: out/workflow-smoke/live/claude/plan-artifact.md
Include a section titled "Acceptance criteria" with at least 2 bullets.
When done, reply with exactly: LIVE: claude PASS
```

### 成功の見た目

```bash
test -f out/workflow-smoke/live/claude/plan-artifact.md \
  && rg -n "Acceptance criteria|受け入れ" out/workflow-smoke/live/claude/plan-artifact.md \
  && echo "LIVE: claude PASS (file ok)"
```

あなた → AI: `LIVE: claude PASS`

---

## 2. Codex CLI（live）

### コピペ

```bash
cd /Users/tachibanashuuta/LocalWork/Code/CC-harness/claude-code-harness
export LIVE_OUT="$PWD/out/workflow-smoke/live/codex"
mkdir -p "$LIVE_OUT"

# モデルは環境の既定でよい。必要なら:
# export CODEX_MODEL="$(bash scripts/model-routing.sh --host codex --role worker --field model)"

codex exec --skip-git-repo-check "$(cat <<'EOF'
Use the harness-plan skill (or $harness-plan) for this Live H4 smoke.
Do NOT implement code.
Write ONE short plan for: "Add a single-line comment to README.md that says live-h4-codex".
Save markdown to out/workflow-smoke/live/codex/plan-artifact.md
Must include a section "Acceptance criteria" with >=2 bullets.
When finished, print exactly: LIVE: codex PASS
EOF
)"
```

### 成功の見た目

```bash
test -f out/workflow-smoke/live/codex/plan-artifact.md \
  && rg -n "Acceptance criteria|受け入れ" out/workflow-smoke/live/codex/plan-artifact.md \
  && echo "LIVE: codex PASS (file ok)"
```

あなた → AI: `LIVE: codex PASS`

---

## 3. Cursor（cursor-agent live）

### コピペ

```bash
cd /Users/tachibanashuuta/LocalWork/Code/CC-harness/claude-code-harness
export LIVE_OUT="$PWD/out/workflow-smoke/live/cursor"
mkdir -p "$LIVE_OUT"

# setup 済みであること
bash scripts/setup-cursor.sh --check

MODEL="$(bash scripts/model-routing.sh --host cursor --role worker --field model)"

cursor-agent -p --mode plan --model "$MODEL" --output-format text \
  "Live H4 smoke. Use harness-plan skill if available.
Do NOT implement.
Write ONE short plan for: Add a single-line comment to README.md that says live-h4-cursor.
Save markdown to out/workflow-smoke/live/cursor/plan-artifact.md
Include section 'Acceptance criteria' with at least 2 bullets.
When done print exactly: LIVE: cursor PASS"
```

Desktop でやる場合:

1. Cursor でこの repo を開く  
2. Reload Window  
3. `/harness-plan` または同等で同じ内容を依頼  
4. 成果物パスは同じ  

### 成功の見た目

```bash
test -f out/workflow-smoke/live/cursor/plan-artifact.md \
  && rg -n "Acceptance criteria|受け入れ" out/workflow-smoke/live/cursor/plan-artifact.md \
  && echo "LIVE: cursor PASS (file ok)"
```

あなた → AI: `LIVE: cursor PASS`

---

## 4. Grok（live）

### コピペ

```bash
cd /Users/tachibanashuuta/LocalWork/Code/CC-harness/claude-code-harness
export LIVE_OUT="$PWD/out/workflow-smoke/live/grok"
mkdir -p "$LIVE_OUT"

# Grok 公式 install 済みであること
bash scripts/setup-grok.sh --check
grok plugin list | rg -n 'claude-code-harness' || true

MODEL="$(bash scripts/model-routing.sh --host grok --role worker --field model)"

# headless
grok -p --model "$MODEL" --cwd "$PWD" "$(cat <<'EOF'
Live H4 smoke. Use the harness-plan skill (/harness-plan).
Do NOT implement code.
Write ONE short plan for: "Add a single-line comment to README.md that says live-h4-grok".
Save markdown to out/workflow-smoke/live/grok/plan-artifact.md
Must include a section titled "Acceptance criteria" with at least 2 bullets.
When done, reply with exactly: LIVE: grok PASS
EOF
)"
```

TUI でやる場合: 同じ repo で `grok` 起動 → 上の EOF 内プロンプトを貼る。

### 成功の見た目

```bash
test -f out/workflow-smoke/live/grok/plan-artifact.md \
  && rg -n "Acceptance criteria|受け入れ" out/workflow-smoke/live/grok/plan-artifact.md \
  && echo "LIVE: grok PASS (file ok)"
```

あなた → AI: `LIVE: grok PASS`

---

## 5. 一括チェック（ファイルだけ）

```bash
cd /Users/tachibanashuuta/LocalWork/Code/CC-harness/claude-code-harness
for h in claude codex cursor grok; do
  f="out/workflow-smoke/live/$h/plan-artifact.md"
  if test -f "$f" && rg -q "Acceptance criteria|受け入れ" "$f"; then
    echo "OK  $h"
  else
    echo "NG  $h  (missing or no Acceptance criteria)"
  fi
done
```

全部 OK なら AI へ:

```text
LIVE: all PASS
```

---

## 6. FAIL の書き方（テンプレ）

```text
LIVE: codex FAIL: skill harness-plan not found
LIVE: cursor FAIL: no plan-artifact.md written
LIVE: grok FAIL: auth / model unavailable
LIVE: claude FAIL: permission blocked write to out/
```

---

## 7. 合格後に何が起きるか

| 状況 | 次のアクション |
|------|----------------|
| 1 host だけ PASS | その host の live 証拠を `docs/research/*` or Plans に日付付き追記 |
| 4 host PASS | H1–H8 を再判定 → 111.3.3 / 111.4.4 / 111.5.4 の **昇格 PR** を切れる |
| structural だけ PASS | **正式対応にはしない**（今の状態） |

関連:

- 方針: `docs/research/host-workflow-smoke-policy.md`
- H1–H8: `docs/spec/planning-and-host-adapter.md`
- 印刷用: `bash scripts/print-live-cli-smoke.sh`
