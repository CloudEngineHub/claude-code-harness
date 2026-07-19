#!/bin/bash
# Shared Plans.md Status-column marker helpers for shell consumers.
# Parsing rules align with go/internal/plans/plans.go (task rows only; Status = last cell).

# Count task rows whose trimmed Status cell starts with marker (case-sensitive prefix).
# Usage: count_status_cells <marker> [plans-file]
count_status_cells() {
  local marker=$1
  local file=${2:-${PLANS_FILE:-Plans.md}}
  if [ ! -f "$file" ]; then
    echo 0
    return 0
  fi
  awk -v m="$marker" '
    function trim(value) {
      gsub(/^[ \t`]+|[ \t`]+$/, "", value)
      return value
    }
    function split_row(line, cells,   n, i) {
      n = split(line, cells, "|")
      for (i = 1; i <= n; i++) {
        cells[i] = trim(cells[i])
      }
      return n
    }
    function first_cell(cells) {
      return cells[2]
    }
    function status_cell(cells, n) {
      return cells[n - 1]
    }
    function is_task_row(cells,   first) {
      first = first_cell(cells)
      gsub(/`/, "", first)
      if (first == "" || first == "Task" || first ~ /^-+$/) {
        return 0
      }
      if (first ~ /^(pm|cc|cursor):/) {
        return 0
      }
      return 1
    }
    function status_starts_with(cell, marker,   trimmed) {
      trimmed = trim(cell)
      return index(trimmed, marker) == 1
    }
    {
      if ($0 !~ /^\|/) {
        next
      }
      n = split_row($0, cells)
      if (n < 6) {
        next
      }
      if (!is_task_row(cells)) {
        next
      }
      if (status_starts_with(status_cell(cells, n), m)) {
        count++
      }
    }
    END { print count + 0 }
  ' "$file"
}

# Count task rows whose Status cell matches an extended regex (e.g. cc:(done|完了)).
count_status_cells_matching() {
  local pattern=$1
  local file=${2:-${PLANS_FILE:-Plans.md}}
  if [ ! -f "$file" ]; then
    echo 0
    return 0
  fi
  awk -v pattern="$pattern" '
    function trim(value) {
      gsub(/^[ \t`]+|[ \t`]+$/, "", value)
      return value
    }
    function split_row(line, cells,   n, i) {
      n = split(line, cells, "|")
      for (i = 1; i <= n; i++) {
        cells[i] = trim(cells[i])
      }
      return n
    }
    function first_cell(cells) {
      return cells[2]
    }
    function status_cell(cells, n) {
      return cells[n - 1]
    }
    function is_task_row(cells,   first) {
      first = first_cell(cells)
      gsub(/`/, "", first)
      if (first == "" || first == "Task" || first ~ /^-+$/) {
        return 0
      }
      if (first ~ /^(pm|cc|cursor):/) {
        return 0
      }
      return 1
    }
    {
      if ($0 !~ /^\|/) {
        next
      }
      n = split_row($0, cells)
      if (n < 6) {
        next
      }
      if (!is_task_row(cells)) {
        next
      }
      if (status_cell(cells, n) ~ pattern) {
        count++
      }
    }
    END { print count + 0 }
  ' "$file"
}

# Print "line:row" for task rows whose Status cell matches pattern (limit default 20).
list_status_cell_tasks() {
  local pattern=$1
  local file=${2:-${PLANS_FILE:-Plans.md}}
  local limit=${3:-20}
  if [ ! -f "$file" ]; then
    return 0
  fi
  awk -v pattern="$pattern" -v limit="$limit" '
    function trim(value) {
      gsub(/^[ \t`]+|[ \t`]+$/, "", value)
      return value
    }
    function split_row(line, cells,   n, i) {
      n = split(line, cells, "|")
      for (i = 1; i <= n; i++) {
        cells[i] = trim(cells[i])
      }
      return n
    }
    function first_cell(cells) {
      return cells[2]
    }
    function status_cell(cells, n) {
      return cells[n - 1]
    }
    function is_task_row(cells,   first) {
      first = first_cell(cells)
      gsub(/`/, "", first)
      if (first == "" || first == "Task" || first ~ /^-+$/) {
        return 0
      }
      if (first ~ /^(pm|cc|cursor):/) {
        return 0
      }
      return 1
    }
    {
      if ($0 !~ /^\|/) {
        next
      }
      n = split_row($0, cells)
      if (n < 6) {
        next
      }
      if (!is_task_row(cells)) {
        next
      }
      if (status_cell(cells, n) ~ pattern) {
        print NR ":" $0
        printed++
        if (printed >= limit) {
          exit
        }
      }
    }
  ' "$file"
}

# List heading/checklist task lines whose text matches pattern (non-table Plans formats).
list_heading_plan_tasks() {
  local pattern=$1
  local file=${2:-${PLANS_FILE:-Plans.md}}
  local limit=${3:-20}
  if [ ! -f "$file" ]; then
    return 0
  fi
  awk -v pattern="$pattern" -v limit="$limit" '
    /^[[:space:]]*[-*+][[:space:]]+\[[ xX]\]/ || /^[[:space:]]*#+[[:space:]]+/ {
      if ($0 ~ pattern) {
        print NR ":" $0
        printed++
        if (printed >= limit) {
          exit
        }
      }
    }
  ' "$file"
}
