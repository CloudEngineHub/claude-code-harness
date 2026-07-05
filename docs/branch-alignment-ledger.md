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
