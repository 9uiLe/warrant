#!/usr/bin/env bash
# Markdown ドキュメントの体裁チェック(CLAUDE.md「過剰な Divider は禁止する」の機械的担保)。
#
# 検出対象: 本文中の水平線(`---` / `***` / `___` の単独行)。
#   - 見出し(`##`)で構造化すれば水平線は不要。可読性を下げるため使わない。
#   - YAML frontmatter の区切り(ファイル先頭の `---` ペア)は除外する。
#   - Markdown テーブルの区切り(`|---|` 等)は単独行ではないため対象外。
#
# 使い方: scripts/lint-docs.sh [path ...]   (省略時は docs/ + README.md + CONCEPT.md)
# 終了コード: 違反があれば 1、なければ 0。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

targets=()
if [ "$#" -gt 0 ]; then
    targets=("$@")
else
    # 既定: docs 配下の .md と主要ルート文書(bash 3.2 互換のため while-read で収集)
    while IFS= read -r f; do
        targets+=("$f")
    done < <(find docs -name '*.md' 2>/dev/null)
    for f in README.md CONCEPT.md; do
        [ -f "$f" ] && targets+=("$f")
    done
fi

python3 - "${targets[@]}" <<'PY'
import re, sys

HR = re.compile(r'^\s*(-{3,}|\*{3,}|_{3,})\s*$')
violations = 0

for path in sys.argv[1:]:
    try:
        lines = open(path, encoding='utf-8').read().split('\n')
    except OSError:
        continue

    # 先頭が `---` なら YAML frontmatter とみなし、対応する閉じ `---` までを除外する。
    fm_end = -1
    if lines and lines[0].strip() == '---':
        for i in range(1, len(lines)):
            if lines[i].strip() == '---':
                fm_end = i
                break

    for idx, line in enumerate(lines):
        if idx <= fm_end:
            continue
        if HR.match(line):
            print(f"{path}:{idx+1}: 過剰な Divider(水平線)。見出しで区切ること: {line.strip()}")
            violations += 1

if violations:
    print(f"\nNG: {violations} 件の過剰な Divider を検出しました(CLAUDE.md ドキュメント規約)。", file=sys.stderr)
    sys.exit(1)
else:
    print("OK: 過剰な Divider はありません。")
PY
