
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
