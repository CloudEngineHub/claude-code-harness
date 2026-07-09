# AppleScript 決定記録 — activate / open -a のみ許可

Status: **決定済み** (Phase 93.1.2, 2026-06)
Scope: Cursor バックエンド companion / macOS 自動化
Rule SSOT: `.claude/rules/cursor-cli-only.md` の「AppleScript ポリシー」節

---

## 決定

AppleScript および macOS shell 自動化は、**`activate`（アプリ前面化）と `open -a`（アプリ起動）のみ許可**する。
**`System Events` 経由の keystroke / click 注入は禁止**する（2026-06 決定）。

### 背景

- UI イベント注入（キー入力・クリック）はユーザー操作と区別できず、Harness hook 層の deny を迂回する。
- アプリ起動・前面化は副作用が限定的で、PreToolUse / R01-R13 と衝突しない。
- より広い AppleScript 権限は、出所識別・sandbox・hook 対応が揃うまで見送る。

---

## 再昇格 4 条件

以下の **いずれか** が満たされたら、System Events 注入を含む AppleScript 利用の再検討を行う。
4 条件すべてが揃う必要はないが、各条件が解消するリスクを明記する。

### 1. hook input への source-identifier 追加

**何が変わると安全側に倒せるか**: PreToolUse / hook 入力に注入イベントの出所識別子
（agent / user / automation 等）が付与され、UI 注入をユーザー入力と機械的に区別できるようになる。
hook deny が迂回不能な監査経路が確立すれば、限定付きで System Events を再許可できる。

### 2. Read tool の sandbox 拡張

**何が変わると安全側に倒せるか**: Read tool（および関連 file access）が macOS sandbox 下で
パス・権限をより厳密に confine でき、AppleScript 経由の意図しないファイル／UI 操作の
漏洩面積を縮小できる。封じ込めが実効化すれば、自動化の許可範囲を段階的に広げられる。

### 3. Codex hooks の非 Bash イベント対応

**何が変わると安全側に倒せるか**: Codex 側 hook が Bash 以外のイベント（AppleScript 実行、
UI 操作等）も PreToolUse 相当で捕捉・deny できる。3 host 横断で同一 policy engine が
非 shell 自動化を adjudicate できれば、注入経路もガードレール内に収められる。

### 4. Cursor AppleScript dictionary 提供

**何が変わると安全側に倒せるか**: Cursor が公式 AppleScript dictionary（許可 verb・
副作用の契約）を提供し、agent が「何をしてよいか」を型付きで参照できる。
未文書化の System Events 乱用ではなく、契約済み API のみ使う運用に移行できる。

---

## 関連

- `.claude/rules/cursor-cli-only.md` — AppleScript ポリシー（運用ルール）
- `docs/research/cursor-adapter-candidate.md` — Cursor バックエンド採用判断
- `docs/research/zero-base-redesign-2026-06.md` — 3 host native hook 収束
