Build and install the claude-squad binary from main.

Steps:
1. Run `cd /Users/bschron/CodeApps/claude-squad && git branch --show-current` to confirm we're on main or that main exists.
2. Run `go build ./...` to verify compilation succeeds.
3. Run `go test ./...` to verify all tests pass. If tests fail, report the failures and stop.
4. Run `go install .` to install the binary to `~/go/bin/claude-squad` (which is symlinked from `/opt/homebrew/bin/cs`).
5. Verify the binary was updated by checking the modification time: `ls -la /Users/bschron/go/bin/claude-squad`
6. Report success.
