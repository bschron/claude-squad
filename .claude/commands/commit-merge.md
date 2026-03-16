# Commit and Merge

Commit changes on the current branch, then merge into main — both operations in sequence.

## Instruction

Delegate to a **agent-git agent with model=haiku**:

```
Task tool:
  subagent_type = "agent-git"
  model = "haiku"
  prompt = (see below)
```

### Prompt to send:

> mode=commit-merge
> User hint: $ARGUMENTS

**Rules:**
- **subagent_type MUST be "agent-git"** and **model MUST be "haiku"**
- NEVER execute git commands directly — always delegate to the agent
- After the agent returns, report the commit hash, message, and merge result to the user
