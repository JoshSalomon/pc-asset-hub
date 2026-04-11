#!/usr/bin/env bash
# Install Playwright browsers for system tests.
# Only needed on the host machine (not in containers).
set -euo pipefail
cd "$(dirname "$0")/../ui"
npx playwright install chromium
echo "Playwright chromium installed. Run: make test-e2e"
