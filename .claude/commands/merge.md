Merge the current worktree branch into main.

Steps:
1. Run `git status` to confirm we're in a worktree branch (not main). If on main, abort with an error.
2. Get the current branch name with `git branch --show-current`.
3. Check for uncommitted changes. If there are any, ask the user if they want to commit first.
4. Switch to main: `git -C <repo-root> checkout main`
5. Merge the worktree branch: `git -C <repo-root> merge <branch-name>`
6. If there are merge conflicts, report them and stop.
7. Report success with the merged branch name.
