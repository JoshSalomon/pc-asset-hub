
## Code Coverage — Non-Negotiable

**100% coverage is the goal. Not 95%. Not 98%. 100%.**

This project treats test coverage as a first-class quality metric. Every session that touches code MUST leave coverage equal to or better than it found it. Regressions are failures.

### Rules

1. **Every new line must be covered.** No exceptions without explicit human approval.
2. **Every modified file must have its coverage measured and reported.** If you touched a file at 80% coverage, you are responsible for improving it — not just covering your new lines.
3. **Pre-existing uncovered lines in modified files are YOUR problem.** When you modify a file, you own ALL uncovered lines in that file. Write tests for as many as possible. The goal is to leave the file at higher coverage than you found it.
4. **Coverage numbers must be measured, not assumed.** Run the actual coverage tools. Do not carry forward old numbers or estimate. Use `scripts/uncovered-new-lines.sh` and `scripts/uncovered-new-lines-ui.sh` for new-line checks. Use `scripts/coverage-summary.sh` for overall metrics.
5. **Per-file coverage must never decrease.** If `CatalogDetailPage.tsx` was at 81% before your changes and it's at 79% after, you have a regression. Fix it before committing.
6. **Report ALL coverage numbers honestly.** Include the raw counts (e.g., "1843/2179 = 84.6%"). Do not round in ways that hide regressions.
7. **Do not wait to be asked.** Run coverage proactively after implementation, not only when the human invokes /coverage-test.

### Rationalizations that are NOT acceptable

- "The overall percentage didn't change" — per-file matters, not just overall.
- "Those lines were already uncovered before my changes" — you touched the file, you own it.
- "It's just a bind error / framework code" — prove it can't be tested, with ultrathink.
- "Coverage is at 95%, that's good enough" — 95% means 5% of the code is untested. That's not good enough.
- "I'll improve coverage in a later session" — improve it NOW.

## LTM Integration

This project uses the [Claude LTM plugin](https://github.com/JoshSalomon/claude-ltm) for persistent memory across sessions.

### Proactive Memory Usage

When working on tasks, proactively search for relevant memories:

- **Before debugging**: Use `mcp__ltm__recall` to search for prior solutions to similar errors
- **Before implementing features**: Search for related patterns or past decisions
- **When encountering familiar problems**: Check if there's a stored solution

Example scenarios to trigger recall:
- Error messages or exceptions → search for the error type or message
- Working on a specific component → search for that component name
- Configuration issues → search for "config" or the specific setting

After solving a difficult problem, use `mcp__ltm__store_memory` to save the solution for future reference. Always notify the user when a memory is stored (e.g., "Stored this solution to LTM for future reference.").

### Extended Thinking Memory Consultation

**IMPORTANT**: When operating in extended thinking modes ("think harder" or "ultrathink"), you MUST consult long-term memory as part of your reasoning process:

1. **At the start of extended thinking**: Search for memories related to the current task using `mcp__ltm__recall`
2. **During analysis**: Reference any relevant memories found to inform your approach
3. **Before finalizing**: Check if similar problems were solved before and what worked

This ensures that valuable past learnings are incorporated into complex reasoning tasks.
