#!/usr/bin/env bash
#
# Generate a per-package backend coverage table with statement counts.
# Outputs markdown table with covered/total for each package.
#
# Usage:
#   scripts/go-coverage-table.sh [coverage.out]
#
# Prerequisites:
#   go test ./internal/... -count=1 -coverprofile=coverage.out
#

set -euo pipefail

COVFILE="${1:-coverage.out}"

if [[ ! -f "$COVFILE" ]]; then
    echo "Coverage file not found: $COVFILE"
    echo "Run: go test ./internal/... -count=1 -coverprofile=$COVFILE"
    exit 1
fi

python3 -c "
import re, collections, sys

pkg_stats = collections.defaultdict(lambda: {'covered': 0, 'total': 0})

with open('$COVFILE') as f:
    for line in f:
        if line.startswith('mode:'):
            continue
        m = re.match(r'(.+?):(\d+)\.\d+,(\d+)\.\d+\s+(\d+)\s+(\d+)', line)
        if not m:
            continue
        filepath = m.group(1)
        num_stmts = int(m.group(4))
        count = int(m.group(5))

        pkg = '/'.join(filepath.replace('github.com/project-catalyst/pc-asset-hub/', '').split('/')[:-1])

        pkg_stats[pkg]['total'] += num_stmts
        if count > 0:
            pkg_stats[pkg]['covered'] += num_stmts

print('| Package | Coverage |')
print('|---------|----------|')
for pkg in sorted(pkg_stats.keys()):
    s = pkg_stats[pkg]
    pct = (s['covered'] * 100 / s['total']) if s['total'] > 0 else 0
    print(f'| \`{pkg}\` | {pct:.1f}% ({s[\"covered\"]}/{s[\"total\"]}) |')

print()
total_covered = sum(s['covered'] for s in pkg_stats.values())
total = sum(s['total'] for s in pkg_stats.values())
print(f'**Total: {total_covered}/{total} = {total_covered*100/total:.1f}%**')
"
