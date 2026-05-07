# Push

Push current branch to remote.

## GitHub account switch (REQUIRED)

This machine has two `gh` accounts logged in: `uniumautomacao` (default) and `bschron`. Pushes to `bschron`-owned remotes fail with 403 unless the active gh account is switched, because the credential helper hands the wrong token.

**Before pushing**, the agent MUST:
1. Capture the currently active gh account: `gh auth status` (look for the user marked "active")
2. Switch to bschron: `gh auth switch -u bschron`

**After pushing** (success OR failure), the agent MUST restore the original account:
3. `gh auth switch -u <original-user>`

The restore step runs unconditionally — even if the push fails — so the machine is left in its original state.

## Dispatch

Delegate to `agent-git` (haiku): *"mode=push. User hint: $ARGUMENTS. IMPORTANT: Before `git push`, run `gh auth status` to capture the active account, then `gh auth switch -u bschron`. After the push (success or failure), run `gh auth switch -u <original-account>` to restore. Report both the push result and the account-switch round-trip."*

After the agent returns, report the result to the user.
