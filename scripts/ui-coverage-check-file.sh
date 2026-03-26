#!/usr/bin/env bash
#
# Check if a file appears in UI coverage data and show its coverage stats.
#
# Usage:
#   scripts/ui-coverage-check-file.sh <pattern>     # search by pattern
#   scripts/ui-coverage-check-file.sh --list-all    # list all files in coverage
#
# Examples:
#   scripts/ui-coverage-check-file.sh LandingPage
#   scripts/ui-coverage-check-file.sh useCatalogDiagram
#   scripts/ui-coverage-check-file.sh --list-all
#

set -euo pipefail

COVERAGE_FILE="${COVERAGE_FILE:-ui/coverage/coverage-final.json}"

if [[ ! -f "$COVERAGE_FILE" ]]; then
    echo "Coverage file not found: $COVERAGE_FILE"
    echo "Run: cd ui && npx vitest run --config vitest.browser.config.ts --coverage"
    exit 1
fi

if [[ "${1:-}" == "--list-all" ]]; then
    python3 -c "
import json, sys
with open('$COVERAGE_FILE') as f:
    d = json.load(f)
print(f'Total files in coverage: {len(d)}')
print()
for k in sorted(d.keys()):
    short = k.split('/ui/src/')[-1] if '/ui/src/' in k else k
    s = d[k]['s']
    total = len(s)
    covered = sum(1 for v in s.values() if v > 0)
    pct = (covered * 100 / total) if total > 0 else 0
    print(f'  {short}: {covered}/{total} ({pct:.1f}%)')
"
    exit 0
fi

PATTERN="${1:?Usage: ui-coverage-check-file.sh <pattern>}"

python3 -c "
import json, sys
with open('$COVERAGE_FILE') as f:
    d = json.load(f)

pattern = '$PATTERN'
matches = [k for k in d.keys() if pattern in k]

if not matches:
    # List all files to help debug
    print(f'NOT FOUND: no file matching \"{pattern}\" in coverage data.')
    print(f'Total files tracked: {len(d)}')
    print()
    print('All tracked files:')
    for k in sorted(d.keys()):
        short = k.split('/ui/src/')[-1] if '/ui/src/' in k else k
        print(f'  {short}')
    sys.exit(1)

for k in matches:
    short = k.split('/ui/src/')[-1] if '/ui/src/' in k else k
    entry = d[k]
    s = entry['s']
    total = len(s)
    covered = sum(1 for v in s.values() if v > 0)
    pct = (covered * 100 / total) if total > 0 else 0
    print(f'=== {short} ({covered}/{total} = {pct:.1f}%) ===')

    uncovered = []
    # Read source lines for uncovered statements
    for sid, count in s.items():
        if count == 0:
            loc = entry['statementMap'].get(sid, {})
            start = loc.get('start', {})
            line = start.get('line', '?')
            uncovered.append(int(line) if isinstance(line, int) else 0)

    if uncovered:
        print(f'  {len(uncovered)} uncovered statements on lines: {sorted(set(uncovered))}')
    else:
        print('  All statements covered!')
"
