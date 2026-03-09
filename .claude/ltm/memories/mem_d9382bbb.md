---
id: "mem_d9382bbb"
topic: "Fix LTM plugin SELinux permission errors on Fedora/podman"
tags:
  - ltm
  - selinux
  - podman
  - permissions
  - debugging
phase: 0
difficulty: 0.4
created_at: "2026-03-09T15:27:39.274775+00:00"
created_session: 16
---
## Problem
LTM MCP server returns `Permission denied: '/data/memories/mem_XXXX.md'` when trying to read/write memories. Happens on every new Claude Code session because the LTM container gets a new SELinux MCS category pair on each `podman run`.

### Root Cause
The LTM plugin's `run-mcp.sh` mounts `.claude/ltm/` to `/data` inside the container using `-v ${DATA_DIR}:/data:Z`. The `:Z` flag assigns a unique MCS label (e.g., `s0:c277,c631`) to the files. Each new container run gets a different MCS pair, so files created by a previous session are inaccessible.

### Fix (run at start of session if LTM errors)
```bash
chcon -R -l s0 /home/jsalomon/src/pc-asset-hub/.claude/ltm/
```
This strips the MCS categories from all LTM files, making them accessible to any container regardless of its MCS pair.

### Permanent Fix
Change `:Z` to `:z` (lowercase) in the LTM plugin's `run-mcp.sh` at `~/.claude/plugins/marketplaces/claude-ltm/scripts/run-mcp.sh`. The `:z` flag uses a shared SELinux label instead of a private one. This is a plugin-level change — file an issue at the claude-ltm repo.

### Environment
- Fedora with SELinux enforcing
- Podman (rootless containers with `--userns=keep-id`)
- LTM plugin running as MCP server in container
