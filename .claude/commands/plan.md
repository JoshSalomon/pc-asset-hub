# Project Planning Skill

Follow this structured planning methodology exactly. Every plan has mandatory phases that must be completed in order. Do not skip or reorder phases unless explicitly instructed by the user.

## Phase 1: Architecture Research and Decisions

Before any implementation planning, research and present the architecture and related design decisions.

- Investigate the current codebase state, relevant files, and existing patterns.
- Present architectural decisions clearly: what is being decided, what the options are, and the recommended approach with rationale.
- If the task involves no architecture changes (e.g., a pure bug fix or cosmetic change), state this explicitly and skip to Phase 2.
- Architecture decisions must NOT appear in later phases — all architectural work happens here.

**STOP: This phase requires user approval before proceeding to Phase 2.**

## Phase 2: High-Level Test Plan

Prepare or modify the high-level test plan for the feature/change.

- Identify what needs to be tested: new functionality, regressions, edge cases, integration points.
- Describe the testing strategy at a high level (unit tests, integration tests, E2E tests, manual verification).
- If the existing test plan already covers this change and no modifications are needed, state this — but user approval is still required.
- If the detailed test plan (Phase 3) is clearly unnecessary for this change, you may suggest to the user that Phase 3 be skipped — but the user must explicitly approve skipping it.

**STOP: This phase requires user approval before proceeding. Even if there are no changes to the test plan, the user must approve.**

## Phase 3: Detailed Test Plan

Create or modify the detailed test plan with specific test cases.

- List specific test cases with inputs, expected outputs, and assertions.
- Cover happy paths, error paths, boundary conditions, and permission checks.
- Map test cases to the acceptance criteria from the relevant user stories.
- If the user approved skipping this phase in Phase 2, skip it. Otherwise, present the detailed test plan.

**STOP: This phase requires user approval before proceeding. Even if there are no changes, the user must approve (unless they already approved skipping this phase).**

## Phase 4: Implementation Plan

After the mandatory phases above are approved, create the implementation plan. This plan should be designed to run unsupervised for extended periods — group any steps requiring user input at the beginning or end, not scattered throughout.

### Step Structure

Break the implementation into **baby steps**. Each baby step:

- Is small enough to be tested independently before moving to the next step.
- Can and should contain multiple tasks.
- Tasks within a step should be executed in parallel where possible.
- Must pass all relevant tests for that step before proceeding to the next.

### Coverage Requirements

- Run coverage tests at every step.
- Target 100% code coverage.
- If any lines cannot reach 100% coverage (e.g., code that only runs inside a web server and is not reachable in unit tests), document the reason per line with an explicit justification. Do not silently accept coverage gaps.

### Progress Logging

After each step is completed, update a log with:
- Description of what was implemented.
- What tests were run and their results.
- Any deviations from the plan and why.

### Multi-Layer Features

When a feature spans multiple layers (database, backend, UI), implement bottom-up:

1. **Database layer** — schema changes, migrations. Fully test before proceeding.
2. **Backend layer** — API endpoints, business logic. Fully test before proceeding.
3. **UI layer** — components, views, interactions. Fully test before proceeding.

Do not start a higher layer until the lower layer is fully tested and passing.

### Final Steps

Every plan must end with these steps, in order:

1. **Security scan**
   - Run a comprehensive security audit using the security-auditor skill or a dedicated security expert agent.
   - Scan for OWASP Top 10 vulnerabilities, insecure patterns, secrets exposure, injection flaws, authentication/authorization issues, and dependency vulnerabilities.
   - Fix all identified vulnerabilities where possible.
   - After fixes, run the full test suite including coverage — all tests must pass and coverage targets must still be met.
   - Produce a security report documenting:
     - Vulnerabilities found and fixed (what was found, what was changed).
     - Remaining vulnerabilities that could not be fixed, with justification and recommended mitigations.
     - Any accepted risks with rationale.

2. **Code cleanup**
   - Run linter and fix all issues.
   - Eliminate code duplication.
   - Add code documentation where the logic is not self-evident (do not over-document).

3. **Full test run**
   - Run the entire test suite (not just tests related to the current change).
   - All tests must pass. If any test fails, fix it before proceeding.
   - Run coverage analysis and verify it meets the 100% target (with documented exceptions).

4. **Documentation updates**
   - Update any affected documentation (PRD, API docs, README, etc.) to reflect the changes.

5. **Long-term memory updates**
   - Store new memories for learnings, patterns, or decisions made during implementation.
   - Review existing memories and remove any that are no longer accurate or relevant.

6. **Stage files for commit**
   - List all files added or modified and why (group by category: code changes, test changes, documentation changes, etc.).
   - List any files explicitly NOT staged and why (e.g., generated files, local config).

7. **Prepare commit message**
   - Write the commit message to a file — do not commit.

**CRITICAL: Do NOT commit changes to git unless the user explicitly requested a commit step in the plan. NEVER push to git without a direct request from the user.**
