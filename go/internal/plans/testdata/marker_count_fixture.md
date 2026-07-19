# Marker count regression fixture (Task 119.1)

| `cc:wip` / `cc:WIP` | 着手中 | Impl |

正規出力は pm:requested → cc:todo → cc:wip → cc:done → pm:approved

| Task | 内容 | DoD | Depends | Status |
|---|---|---|---|---|
| 1.1 | cc:WIP 状態が 10 分超なら re-spawn する仕組み | test PASS | - | cc:done [abc123] |
| 1.2 | 実装A | test PASS | - | cc:wip |
| 1.3 | 実装B | test PASS | 1.2 | cc:todo |
| 1.4 | 実装C | test PASS | - | cc:WIP |
| 1.5 | 実装D | test PASS | - | cc:完了 |
