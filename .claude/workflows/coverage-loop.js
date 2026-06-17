export const meta = {
  name: 'coverage-loop',
  description: 'Deterministic generate/review coverage loop — no human-in-the-loop override of REJECTED verdicts',
  phases: [
    { title: 'Generate', detail: 'Run coverage, write tests, produce report' },
    { title: 'Review', detail: 'Adversarial audit of coverage report (read-only, ultrathink)' },
  ],
}

// Schema for structured review verdict — no ambiguity, no interpretation
const VERDICT_SCHEMA = {
  type: 'object',
  properties: {
    verdict: {
      type: 'string',
      enum: ['APPROVED', 'REJECTED'],
      description: 'APPROVED if all coverage checks pass. REJECTED if any issue found.',
    },
    findings: {
      type: 'string',
      description: 'If REJECTED: specific findings with file:line references and suggested test approaches. If APPROVED: brief confirmation.',
    },
    uncoveredNewLines: {
      type: 'integer',
      description: 'Number of uncovered new lines reported by the generator',
    },
    backendCoverage: {
      type: 'string',
      description: 'Backend coverage as "covered/total (X uncov)" e.g. "6312/6367 (55 uncov)"',
    },
    uiCoverage: {
      type: 'string',
      description: 'UI coverage as "covered/total (X uncov)" e.g. "3557/3713 (156 uncov)"',
    },
  },
  required: ['verdict', 'findings'],
}

// Context comes from args (passed by the caller)
const context = args.context || 'No context provided'
const generateSkillPath = args.generateSkillPath || '/tmp/home-jsalomon/.claude/skills/coverage-test-subskill/SKILL.md'
const reviewSkillPath = args.reviewSkillPath || '/tmp/home-jsalomon/.claude/skills/coverage-review-subskill/SKILL.md'

const uiCoverageInstructions = `
### Important instructions for UI coverage:
- Use NODE_OPTIONS="--max-old-space-size=4096" to prevent OOM
- Use --reporter=dot to reduce output
- Redirect output to file, don't pipe through tail
- UI coverage takes 15-30 minutes — this is normal
- coverage-final.json only appears after vitest fully exits
- Run from the ui/ directory
`

const generateBasePrompt = `You are running the coverage-generate skill.
Read the skill at ${generateSkillPath} and follow it exactly.

${context}

${uiCoverageInstructions}

### Important instructions for all coverage:
- Delete old coverage files FIRST (Step 0)
- Do NOT use -coverpkg for Go coverage
- Report covered/total counts, not just percentages
- For every uncovered new line: write a test FIRST, justify only if the test fails
- For every "uncoverable" justification: show the test you attempted and why it failed
- The review agent will use ultrathink to independently try to find test approaches
  — if it finds one you didn't try, your justification will be REJECTED

Execute all steps. Produce the full report.`

const reviewBasePrompt = `You are running the coverage-review skill. You are READ-ONLY.
You do NOT run tests, write code, or edit any files.
You only read files and verify claims.

Read the skill at ${reviewSkillPath} and follow it exactly.

Audit the coverage report that was just produced.
Read docs/coverage-report.md, read source files to verify
uncovered line justifications, check arithmetic independently.

CRITICAL: For every "uncoverable" line, use ultrathink to
independently try to find a test approach. Your default position
is that every line CAN be covered. Only accept "uncoverable" if
you genuinely cannot think of a way after deep analysis AND the
generator showed the test they attempted.

Return your verdict via the structured output tool.
For each REJECTED "uncoverable" line, include your suggested
test approach in the findings so the generator knows what to try.`

// === THE LOOP ===
// This is deterministic. REJECTED = loop. APPROVED = exit.
// No agent gets to interpret, override, or dismiss the verdict.

let lastFindings = ''
let iterations = 0

for (let i = 0; i < 5; i++) {
  iterations = i + 1
  log(`=== Iteration ${iterations} ===`)

  // --- GENERATE ---
  phase('Generate')
  const genPrompt = i === 0
    ? generateBasePrompt
    : `You are running the coverage-generate skill to fix review findings.
Read the skill at ${generateSkillPath} and follow it exactly.

${context}

${uiCoverageInstructions}

The coverage review REJECTED your report. Findings:

${lastFindings}

Fix each issue:
- Wrong numbers → re-measure coverage
- Coverable lines left uncovered → write tests
- Lines REJECTED with reviewer's suggested test approach → TRY the reviewer's approach first
- Missing test attempts in justifications → write the test, show results
- Report structure issues → fix docs/coverage-report.md
- Arithmetic discrepancies → reconcile block-by-block

Report back with the corrected results.`

  await agent(genPrompt, {
    label: `generate-${iterations}`,
    phase: 'Generate',
  })

  // --- REVIEW ---
  phase('Review')
  const review = await agent(reviewBasePrompt, {
    label: `review-${iterations}`,
    phase: 'Review',
    schema: VERDICT_SCHEMA,
  })

  if (!review) {
    log(`Review agent returned null (iteration ${iterations}). Treating as REJECTED.`)
    lastFindings = 'Review agent failed to return a verdict. Re-run.'
    continue
  }

  log(`Iteration ${iterations}: ${review.verdict}`)

  if (review.verdict === 'APPROVED') {
    log(`Coverage APPROVED after ${iterations} iteration(s).`)
    return {
      approved: true,
      iterations,
      backendCoverage: review.backendCoverage || 'not reported',
      uiCoverage: review.uiCoverage || 'not reported',
      findings: review.findings,
    }
  }

  // REJECTED — loop unconditionally.
  // No interpretation. No "the reviewer is confused." No override.
  lastFindings = review.findings
  log(`REJECTED. Findings: ${(review.findings || '').substring(0, 300)}...`)
}

// 5 iterations exhausted — escalate to human
log(`5 iterations exhausted. Escalating to human.`)
return {
  approved: false,
  iterations,
  lastFindings,
  message: 'Coverage review loop exhausted 5 iterations without approval. Human must review.',
}
