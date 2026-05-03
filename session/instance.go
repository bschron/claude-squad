package session

import (
	"claude-squad/config"
	"claude-squad/log"
	"claude-squad/session/git"
	"claude-squad/session/tmux"
	"path/filepath"

	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// buildStartProgram consolidates the program string with flags for Claude.
func (i *Instance) buildStartProgram(extraFlags ...string) string {
	prog := i.Program
	if isClaudeProgram(prog) {
		for _, f := range extraFlags {
			prog += " " + f
		}
		if i.SkipPermissions {
			prog += " --dangerously-skip-permissions"
		}
		if i.Effort != "" {
			prog += " --effort " + string(i.Effort)
		}
		if i.Model != "" {
			prog += " --model " + string(i.Model)
		}
	}
	return prog
}

type Status int

const (
	// Running is the status when the instance is running and claude is working.
	Running Status = iota
	// Ready is if the claude instance is ready to be interacted with (waiting for user input).
	Ready
	// Loading is if the instance is loading (if we are starting it up or something).
	Loading
	// Paused is if the instance is paused (worktree removed but branch preserved).
	Paused
)

// Instance is a running instance of claude code.
type Instance struct {
	// Title is the title of the instance.
	Title string
	// Path is the path to the workspace.
	Path string
	// Branch is the branch of the instance.
	Branch string
	// Status is the status of the instance.
	Status Status
	// Program is the program to run in the instance.
	Program string
	// Height is the height of the instance.
	Height int
	// Width is the width of the instance.
	Width int
	// CreatedAt is the time the instance was created.
	CreatedAt time.Time
	// UpdatedAt is the time the instance was last updated.
	UpdatedAt time.Time
	// AutoYes is true if the instance should automatically press enter when prompted.
	AutoYes bool
	// Prompt is the initial prompt to pass to the instance on startup
	Prompt string
	// Effort is the effort level for Claude Code (low, medium, high, max)
	Effort config.EffortLevel
	// Model is the model option for Claude Code (sonnet, opus, haiku, or empty for default)
	Model config.ModelOption
	// SkipPermissions controls whether --dangerously-skip-permissions is passed
	SkipPermissions bool

	// DiffStats stores the current git diff statistics
	diffStats *git.DiffStats

	// lastDiffContentUpdate records when Content was last populated by
	// UpdateDiffContent. Used to debounce full-diff refreshes while the
	// Diff tab is visible.
	lastDiffContentUpdate time.Time

	// selectedBranch is the existing branch to start on (empty = new branch from HEAD)
	selectedBranch string

	// The below fields are initialized upon calling Start().

	started bool
	// tmuxSession is the tmux session for the instance.
	tmuxSession *tmux.TmuxSession
	// gitWorktree is the git worktree for the instance.
	gitWorktree *git.GitWorktree

	// isExternal is true for sessions discovered from external tools (e.g. Claude Code --worktree).
	// External sessions are not persisted to state.json and destructive operations are blocked.
	isExternal bool

	// IdleSince tracks when the instance first became idle (no content changes).
	// Used to debounce the Running→Ready transition by 1 second.
	IdleSince *time.Time
}

// IsExternal returns true if this instance was discovered from an external tool.
func (i *Instance) IsExternal() bool {
	return i.isExternal
}

// SetManaged converts an external session to a managed session.
func (i *Instance) SetManaged() {
	i.isExternal = false
}

// ToInstanceData converts an Instance to its serializable form
func (i *Instance) ToInstanceData() InstanceData {
	skipPerms := i.SkipPermissions
	data := InstanceData{
		Title:           i.Title,
		Path:            i.Path,
		Branch:          i.Branch,
		Status:          i.Status,
		Height:          i.Height,
		Width:           i.Width,
		CreatedAt:       i.CreatedAt,
		UpdatedAt:       time.Now(),
		Program:         i.Program,
		AutoYes:         i.AutoYes,
		Effort:          i.Effort,
		Model:           i.Model,
		SkipPermissions: &skipPerms,
	}

	// Only include worktree data if gitWorktree is initialized
	if i.gitWorktree != nil {
		data.Worktree = GitWorktreeData{
			RepoPath:         i.gitWorktree.GetRepoPath(),
			WorktreePath:     i.gitWorktree.GetWorktreePath(),
			SessionName:      i.Title,
			BranchName:       i.gitWorktree.GetBranchName(),
			BaseCommitSHA:    i.gitWorktree.GetBaseCommitSHA(),
			IsExistingBranch: i.gitWorktree.IsExistingBranch(),
		}
	}

	// Only include diff stats if they exist
	if i.diffStats != nil {
		data.DiffStats = DiffStatsData{
			Added:   i.diffStats.Added,
			Removed: i.diffStats.Removed,
			Content: i.diffStats.Content,
		}
	}

	return data
}

// FromInstanceData creates a new Instance from serialized data
func FromInstanceData(data InstanceData) (*Instance, error) {
	// Backward compat: legacy sessions without skip_permissions default to true
	skipPerms := true
	if data.SkipPermissions != nil {
		skipPerms = *data.SkipPermissions
	}

	instance := &Instance{
		Title:           data.Title,
		Path:            data.Path,
		Branch:          data.Branch,
		Status:          data.Status,
		Height:          data.Height,
		Width:           data.Width,
		CreatedAt:       data.CreatedAt,
		UpdatedAt:       data.UpdatedAt,
		Program:         data.Program,
		Effort:          data.Effort,
		Model:           data.Model,
		SkipPermissions: skipPerms,
		gitWorktree: git.NewGitWorktreeFromStorage(
			data.Worktree.RepoPath,
			data.Worktree.WorktreePath,
			data.Worktree.SessionName,
			data.Worktree.BranchName,
			data.Worktree.BaseCommitSHA,
			data.Worktree.IsExistingBranch,
		),
		diffStats: &git.DiffStats{
			Added:   data.DiffStats.Added,
			Removed: data.DiffStats.Removed,
			Content: data.DiffStats.Content,
		},
	}

	// Detect legacy sessions (worktree in ~/.claude-squad/worktrees/)
	configDir, _ := config.GetConfigDir()
	globalWorktreeDir := filepath.Join(configDir, "worktrees")
	isLegacy := strings.HasPrefix(data.Worktree.WorktreePath, globalWorktreeDir)

	if instance.Paused() {
		instance.started = true
		if isLegacy {
			instance.tmuxSession = tmux.NewLegacyTmuxSession(instance.Title, instance.Program)
		} else {
			projectDirname := filepath.Base(data.Worktree.RepoPath)
			instance.tmuxSession = tmux.NewTmuxSession(instance.Title, instance.Program, projectDirname)
		}
	} else {
		if err := instance.Start(false); err != nil {
			return nil, err
		}
	}

	return instance, nil
}

// Options for creating a new instance
type InstanceOptions struct {
	// Title is the title of the instance.
	Title string
	// Path is the path to the workspace.
	Path string
	// Program is the program to run in the instance (e.g. "claude", "aider --model ollama_chat/gemma3:1b")
	Program string
	// If AutoYes is true, then
	AutoYes bool
	// Branch is an existing branch name to start the session on (empty = new branch from HEAD)
	Branch string
}

func NewInstance(opts InstanceOptions) (*Instance, error) {
	t := time.Now()

	// Convert path to absolute
	absPath, err := filepath.Abs(opts.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return &Instance{
		Title:          opts.Title,
		Status:         Ready,
		Path:           absPath,
		Program:        opts.Program,
		Height:         0,
		Width:          0,
		CreatedAt:      t,
		UpdatedAt:      t,
		AutoYes:        false,
		selectedBranch: opts.Branch,
	}, nil
}

// NewExternalInstance creates an Instance from a DiscoveredSession.
// It connects to the existing tmux session and git worktree without owning them.
func NewExternalInstance(ds DiscoveredSession) (*Instance, error) {
	t := time.Now()

	tmuxSession := tmux.NewExternalTmuxSession(ds.TmuxSessionName)
	if !tmuxSession.DoesSessionExist() {
		return nil, fmt.Errorf("tmux session %s does not exist", ds.TmuxSessionName)
	}

	gitWorktree := git.NewGitWorktreeFromStorage(
		ds.RepoPath,
		ds.WorktreePath,
		ds.WorktreeName,
		ds.BranchName,
		"", // baseCommitSHA computed later via diff
		true,
	)

	instance := &Instance{
		Title:       ds.WorktreeName,
		Path:        ds.WorktreePath,
		Branch:      ds.BranchName,
		Status:      Running,
		Program:     tmux.ProgramClaude,
		CreatedAt:   t,
		UpdatedAt:   t,
		started:     true,
		tmuxSession: tmuxSession,
		gitWorktree: gitWorktree,
		isExternal:  true,
	}

	// Connect PTY to the existing tmux session
	if err := tmuxSession.Restore(); err != nil {
		return nil, fmt.Errorf("failed to restore external tmux session: %w", err)
	}

	return instance, nil
}

func (i *Instance) RepoName() (string, error) {
	if !i.started {
		return "", fmt.Errorf("cannot get repo name for instance that has not been started")
	}
	return i.gitWorktree.GetRepoName(), nil
}

func (i *Instance) SetStatus(status Status) {
	i.Status = status
}

// SetSelectedBranch sets the branch to use when starting the instance.
func (i *Instance) SetSelectedBranch(branch string) {
	i.selectedBranch = branch
}

// firstTimeSetup is true if this is a new instance. Otherwise, it's one loaded from storage.
func (i *Instance) Start(firstTimeSetup bool) error {
	if i.Title == "" {
		return fmt.Errorf("instance title cannot be empty")
	}

	// Resolve repo root for tmux naming
	repoRoot, err := git.FindGitRepoRoot(i.Path)
	if err != nil {
		return fmt.Errorf("failed to find git repo root: %w", err)
	}
	projectDirname := filepath.Base(repoRoot)

	var tmuxSession *tmux.TmuxSession
	if i.tmuxSession != nil {
		// Use existing tmux session (useful for testing)
		tmuxSession = i.tmuxSession
	} else {
		// Create new tmux session
		tmuxSession = tmux.NewTmuxSession(i.Title, i.Program, projectDirname)
	}
	i.tmuxSession = tmuxSession

	if firstTimeSetup {
		if i.selectedBranch != "" {
			gitWorktree, err := git.NewGitWorktreeFromBranch(i.Path, i.selectedBranch, i.Title)
			if err != nil {
				return fmt.Errorf("failed to create git worktree from branch: %w", err)
			}
			i.gitWorktree = gitWorktree
			i.Branch = i.selectedBranch
		} else {
			gitWorktree, branchName, err := git.NewGitWorktree(i.Path, i.Title)
			if err != nil {
				return fmt.Errorf("failed to create git worktree: %w", err)
			}
			i.gitWorktree = gitWorktree
			i.Branch = branchName
		}
	}

	// Setup error handler to cleanup resources on any error
	var setupErr error
	defer func() {
		if setupErr != nil {
			if cleanupErr := i.Kill(); cleanupErr != nil {
				setupErr = fmt.Errorf("%v (cleanup error: %v)", setupErr, cleanupErr)
			}
		} else {
			i.started = true
		}
	}()

	if !firstTimeSetup {
		// Reuse existing session
		if err := tmuxSession.Restore(); err != nil {
			setupErr = fmt.Errorf("failed to restore existing session: %w", err)
			return setupErr
		}
	} else {
		// Setup git worktree first
		if err := i.gitWorktree.Setup(); err != nil {
			setupErr = fmt.Errorf("failed to setup git worktree: %w", err)
			return setupErr
		}

		// Build program with flags for Claude
		startProgram := i.buildStartProgram()

		// Create new session
		if err := i.tmuxSession.Start(i.gitWorktree.GetWorktreePath(), startProgram); err != nil {
			// Cleanup git worktree if tmux session creation fails
			if cleanupErr := i.gitWorktree.Cleanup(); cleanupErr != nil {
				err = fmt.Errorf("%v (cleanup error: %v)", err, cleanupErr)
			}
			setupErr = fmt.Errorf("failed to start new session: %w", err)
			return setupErr
		}
	}

	i.SetStatus(Running)

	return nil
}

// isClaudeProgram returns true if the program is Claude Code.
func isClaudeProgram(program string) bool {
	return program == tmux.ProgramClaude || strings.HasSuffix(program, tmux.ProgramClaude)
}

// Kill terminates the instance and cleans up all resources
func (i *Instance) Kill() error {
	if !i.started {
		// If instance was never started, just return success
		return nil
	}

	var errs []error

	// Always try to cleanup both resources, even if one fails
	// Clean up tmux session first since it's using the git worktree
	if i.tmuxSession != nil {
		if err := i.tmuxSession.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close tmux session: %w", err))
		}
	}

	// Then clean up git worktree
	if i.gitWorktree != nil {
		if err := i.gitWorktree.Cleanup(); err != nil {
			errs = append(errs, fmt.Errorf("failed to cleanup git worktree: %w", err))
		}
	}

	return i.combineErrors(errs)
}

// combineErrors combines multiple errors into a single error
func (i *Instance) combineErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}

	errMsg := "multiple cleanup errors occurred:"
	for _, err := range errs {
		errMsg += "\n  - " + err.Error()
	}
	return fmt.Errorf("%s", errMsg)
}

func (i *Instance) Preview() (string, error) {
	if !i.started || i.Status == Paused {
		return "", nil
	}
	return i.tmuxSession.CapturePaneContent()
}

func (i *Instance) HasUpdated() (updated bool, hasPrompt bool, hasBackgroundTasks bool) {
	if !i.started {
		return false, false, false
	}
	return i.tmuxSession.HasUpdated()
}

// PollStatus performs a single tmux capture-pane and derives all status signals
// (trust-prompt dismissal, content-change detection, prompt/background flags)
// from that one capture. Use this on the metadata tick instead of calling
// CheckAndHandleTrustPrompt + HasUpdated back-to-back (which captured twice).
func (i *Instance) PollStatus() (updated bool, hasPrompt bool, hasBackgroundTasks bool) {
	if !i.started || i.tmuxSession == nil {
		return false, false, false
	}
	return i.tmuxSession.PollStatus()
}

// descendantCPUThreshold is the minimum aggregate %CPU across the
// worktree-associated process set (panePID included) for the session to
// count as "actively working". Empirically idle Claude with all MCP
// servers loaded + idle vite + idle chrome subprocesses read 0.0% even
// after hours of uptime; active Claude in extended thinking + subagent
// dispatch reads 2–8% on the pane process alone; running `playwright test`
// between page loads dips to ~0.9%; full-throttle test execution crosses
// 5%. 0.5 separates "doing some work right now" from "alive but idle"
// without missing the quiet-phase test runs or extended-thinking pauses.
const descendantCPUThreshold = 0.5

// watchedFileFreshness is how recent a log/output file referenced by a
// descendant's argv must be modified to count as activity. 30s tolerates
// burst writes from test runners and build tools that flush in batches.
const watchedFileFreshness = 30 * time.Second

// HasActiveDescendants reports whether any process associated with this
// instance is currently doing work. "Associated" means either a descendant
// of the panePID OR a process whose argv references the worktree path
// (catches detached test runners, e.g. `nohup playwright test` with PPID=1
// running out of the worktree's node_modules whose work happens entirely
// outside the panePID descendant tree). Two complementary signals:
//
//  1. Aggregate %CPU across the associated set ≥ descendantCPUThreshold —
//     catches smoke tests, builds, chrome-headless renderers detached from
//     the Claude tree.
//  2. Any process in the set has argv with a log/output path (absolute or
//     relative to the worktree) modified within watchedFileFreshness —
//     catches Claude's polling shells (`until grep -q ... foo.log;
//     do sleep N; done`) where descendant CPU stays at zero.
//
// snap may be nil if the process listing failed; callers treat that as
// "no signal".
func (i *Instance) HasActiveDescendants(snap *ProcessSnapshot) bool {
	if snap == nil || !i.started || i.Status == Paused || i.tmuxSession == nil {
		return false
	}
	pid, err := i.tmuxSession.PanePID()
	if err != nil || pid == 0 {
		return false
	}
	worktreePath := ""
	if i.gitWorktree != nil {
		worktreePath = i.gitWorktree.GetWorktreePath()
	}
	roots := snap.WorktreeRoots(worktreePath, pid)
	if snap.DescendantCPU(roots) >= descendantCPUThreshold {
		return true
	}
	if snap.HasFreshWatchedFile(roots, worktreePath, watchedFileFreshness) {
		return true
	}
	return false
}

// workArtifactDirs are directories under the worktree where a running test
// runner / build tool writes artifacts. When their mtime is fresh, work is
// happening — even if Claude is idle and no descendant CPU is detectable.
// The list is kept narrow on purpose: dirs like dist/, build/, .next/, .vite/
// would tick constantly during normal dev-server operation and produce false
// positives. These specific dirs only get touched during real test/build runs.
var workArtifactDirs = []string{
	"test-results",
	"playwright-report",
	"coverage",
	".claude/test-run-output",
}

// WorktreeArtifactActive reports whether any well-known test/build artifact
// directory under the worktree was modified within watchedFileFreshness.
// Catches detached test runners (e.g. `npm run test:e2e:timeline` launched
// with PPID=1) whose output goes to fds Claude doesn't track in argv: the
// CPU may be near zero between page transitions and there's no polling
// shell, but the test continues writing failure artifacts to test-results/.
func (i *Instance) WorktreeArtifactActive() bool {
	if !i.started || i.Status == Paused || i.gitWorktree == nil {
		return false
	}
	worktreePath := i.gitWorktree.GetWorktreePath()
	if worktreePath == "" {
		return false
	}
	cutoff := time.Now().Add(-watchedFileFreshness)
	for _, sub := range workArtifactDirs {
		info, err := os.Stat(filepath.Join(worktreePath, sub))
		if err != nil {
			continue
		}
		if info.ModTime().After(cutoff) {
			return true
		}
	}
	return false
}

// TranscriptActive returns true when Claude is producing output for this
// instance's worktree. Two signals are checked, either one suffices:
//
//   - JSONL transcript mtime under ~/.claude/projects/<encoded>/ — fires
//     while the model is streaming a turn or a foreground tool just returned.
//   - Background task .output mtime under /tmp/claude-<uid>/<encoded>/.../tasks/
//     — fires while shells launched via Claude's "run in background" feature
//     are still appending output, even when the JSONL is silent and the pane
//     hash is stable.
//
// Used as an anti-idle signal so the session doesn't get debounced into Ready
// while there's actual work happening.
func (i *Instance) TranscriptActive() bool {
	if !i.started || i.Status == Paused || i.gitWorktree == nil {
		return false
	}
	worktreePath := i.gitWorktree.GetWorktreePath()
	if transcriptRecentlyModified(worktreePath, transcriptActiveThreshold) {
		return true
	}
	if backgroundTaskActive(worktreePath, backgroundTaskActiveThreshold) {
		return true
	}
	return false
}

// TapEnter sends an enter key press to the tmux session if AutoYes is enabled.
// CheckAndHandleTrustPrompt checks for and dismisses the trust prompt for supported programs.
func (i *Instance) CheckAndHandleTrustPrompt() bool {
	if !i.started || i.tmuxSession == nil {
		return false
	}
	program := i.Program
	if !strings.HasSuffix(program, tmux.ProgramClaude) &&
		!strings.HasSuffix(program, tmux.ProgramAider) &&
		!strings.HasSuffix(program, tmux.ProgramGemini) {
		return false
	}
	return i.tmuxSession.CheckAndHandleTrustPrompt()
}

func (i *Instance) TapEnter() {
	if !i.started || !i.AutoYes {
		return
	}
	if err := i.tmuxSession.TapEnter(); err != nil {
		log.ErrorLog.Printf("error tapping enter: %v", err)
	}
}

func (i *Instance) Attach() (chan struct{}, error) {
	if !i.started {
		return nil, fmt.Errorf("cannot attach instance that has not been started")
	}
	return i.tmuxSession.Attach()
}

// ExecAttach returns a tea.ExecCommand for attaching to this instance's tmux session.
func (i *Instance) ExecAttach() (tea.ExecCommand, error) {
	if !i.started {
		return nil, fmt.Errorf("cannot attach instance that has not been started")
	}
	return i.tmuxSession.ExecAttach(), nil
}

func (i *Instance) SetPreviewSize(width, height int) error {
	if !i.started || i.Status == Paused {
		return fmt.Errorf("cannot set preview size for instance that has not been started or " +
			"is paused")
	}
	return i.tmuxSession.SetDetachedSize(width, height)
}

// GetGitWorktree returns the git worktree for the instance
func (i *Instance) GetGitWorktree() (*git.GitWorktree, error) {
	if !i.started {
		return nil, fmt.Errorf("cannot get git worktree for instance that has not been started")
	}
	return i.gitWorktree, nil
}

// GetWorktreePath returns the worktree path for the instance, or empty string if unavailable
func (i *Instance) GetWorktreePath() string {
	if i.gitWorktree == nil {
		return ""
	}
	return i.gitWorktree.GetWorktreePath()
}

func (i *Instance) Started() bool {
	return i.started
}

// SetTitle sets the title of the instance. Returns an error if the instance has started.
// We cant change the title once it's been used for a tmux session etc.
func (i *Instance) SetTitle(title string) error {
	if i.started {
		return fmt.Errorf("cannot change title of a started instance")
	}
	i.Title = title
	return nil
}

func (i *Instance) Paused() bool {
	return i.Status == Paused
}

// WorktreeExists returns true if the worktree directory exists on disk.
func (i *Instance) WorktreeExists() bool {
	path := i.GetWorktreePath()
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// TmuxAlive returns true if the tmux session is alive. This is a sanity check before attaching.
func (i *Instance) TmuxAlive() bool {
	if i.tmuxSession == nil {
		return false
	}
	return i.tmuxSession.DoesSessionExist()
}

// Pause stops the tmux session and removes the worktree, preserving the branch
func (i *Instance) Pause() error {
	if !i.started {
		return fmt.Errorf("cannot pause instance that has not been started")
	}
	if i.Status == Paused {
		return fmt.Errorf("instance is already paused")
	}

	var errs []error

	// Check if there are any changes to commit
	if dirty, err := i.gitWorktree.IsDirty(); err != nil {
		errs = append(errs, fmt.Errorf("failed to check if worktree is dirty: %w", err))
		log.ErrorLog.Print(err)
	} else if dirty {
		// Commit changes locally (without pushing to GitHub)
		commitMsg := fmt.Sprintf("[claudesquad] update from '%s' on %s (paused)", i.Title, time.Now().Format(time.RFC822))
		if err := i.gitWorktree.CommitChanges(commitMsg); err != nil {
			errs = append(errs, fmt.Errorf("failed to commit changes: %w", err))
			log.ErrorLog.Print(err)
			// Return early if we can't commit changes to avoid corrupted state
			return i.combineErrors(errs)
		}
	}

	// Detach from tmux session instead of closing to preserve session output
	if err := i.tmuxSession.DetachSafely(); err != nil {
		errs = append(errs, fmt.Errorf("failed to detach tmux session: %w", err))
		log.ErrorLog.Print(err)
		// Continue with pause process even if detach fails
	}

	// Check if worktree exists before trying to remove it
	if _, err := os.Stat(i.gitWorktree.GetWorktreePath()); err == nil {
		// Remove worktree but keep branch
		if err := i.gitWorktree.Remove(); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove git worktree: %w", err))
			log.ErrorLog.Print(err)
			return i.combineErrors(errs)
		}

		// Only prune if remove was successful
		if err := i.gitWorktree.Prune(); err != nil {
			errs = append(errs, fmt.Errorf("failed to prune git worktrees: %w", err))
			log.ErrorLog.Print(err)
			return i.combineErrors(errs)
		}
	}

	if err := i.combineErrors(errs); err != nil {
		log.ErrorLog.Print(err)
		return err
	}

	i.SetStatus(Paused)
	_ = clipboard.WriteAll(i.gitWorktree.GetBranchName())
	return nil
}

// Resume recreates the worktree and restarts the tmux session
func (i *Instance) Resume() error {
	if !i.started {
		return fmt.Errorf("cannot resume instance that has not been started")
	}
	if i.Status != Paused {
		return fmt.Errorf("can only resume paused instances")
	}

	// Check if branch is checked out
	if checked, err := i.gitWorktree.IsBranchCheckedOut(); err != nil {
		log.ErrorLog.Print(err)
		return fmt.Errorf("failed to check if branch is checked out: %w", err)
	} else if checked {
		return fmt.Errorf("cannot resume: branch is checked out, please switch to a different branch")
	}

	// Setup git worktree
	if err := i.gitWorktree.Setup(); err != nil {
		log.ErrorLog.Print(err)
		return fmt.Errorf("failed to setup git worktree: %w", err)
	}

	// Build program with flags for Claude
	startProgram := i.buildStartProgram("--continue")

	// Check if tmux session still exists from pause, otherwise create new one
	if i.tmuxSession.DoesSessionExist() {
		// Session exists, just restore PTY connection to it
		if err := i.tmuxSession.Restore(); err != nil {
			log.ErrorLog.Print(err)
			// If restore fails, fall back to creating new session
			if err := i.tmuxSession.Start(i.gitWorktree.GetWorktreePath(), startProgram); err != nil {
				log.ErrorLog.Print(err)
				// Cleanup git worktree if tmux session creation fails
				if cleanupErr := i.gitWorktree.Cleanup(); cleanupErr != nil {
					err = fmt.Errorf("%v (cleanup error: %v)", err, cleanupErr)
					log.ErrorLog.Print(err)
				}
				return fmt.Errorf("failed to start new session: %w", err)
			}
		}
	} else {
		// Create new tmux session
		if err := i.tmuxSession.Start(i.gitWorktree.GetWorktreePath(), startProgram); err != nil {
			log.ErrorLog.Print(err)
			// Cleanup git worktree if tmux session creation fails
			if cleanupErr := i.gitWorktree.Cleanup(); cleanupErr != nil {
				err = fmt.Errorf("%v (cleanup error: %v)", err, cleanupErr)
				log.ErrorLog.Print(err)
			}
			return fmt.Errorf("failed to start new session: %w", err)
		}
	}

	i.SetStatus(Running)
	return nil
}

// Revive recreates a dead tmux session for an instance whose worktree still exists.
// This handles the case where tmux sessions are killed (e.g. after a reboot) but the
// worktree and branch are still intact.
func (i *Instance) Revive() error {
	if !i.started {
		return fmt.Errorf("cannot revive instance that has not been started")
	}
	if i.Status == Paused {
		return fmt.Errorf("use Resume() for paused instances")
	}
	if i.tmuxSession.DoesSessionExist() {
		return fmt.Errorf("tmux session is still alive, no need to revive")
	}

	// Build program with --continue flag for Claude so it picks up the previous conversation
	startProgram := i.buildStartProgram("--continue")

	// Create new tmux session in the existing worktree directory
	if err := i.tmuxSession.Start(i.gitWorktree.GetWorktreePath(), startProgram); err != nil {
		return fmt.Errorf("failed to revive session: %w", err)
	}

	i.SetStatus(Running)
	return nil
}

// UpdateDiffStats refreshes only the added/removed counts via `git diff --shortstat`.
// This runs on every metadata tick, so it must stay cheap — the full diff content
// is fetched separately by UpdateDiffContent when the Diff tab needs it.
func (i *Instance) UpdateDiffStats() error {
	stats, err := i.FetchQuickStats()
	if err != nil {
		return err
	}
	i.ApplyDiffStats(stats)
	return nil
}

// FetchQuickStats runs `git diff --shortstat` without mutating the Instance.
// Safe to call from a background goroutine; apply the result via ApplyDiffStats
// from the main goroutine.
func (i *Instance) FetchQuickStats() (*git.DiffStats, error) {
	if !i.started {
		return nil, nil
	}
	if i.Status == Paused {
		return nil, nil
	}
	stats := i.gitWorktree.QuickStats()
	if stats.Error != nil {
		if strings.Contains(stats.Error.Error(), "base commit SHA not set") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get diff stats: %w", stats.Error)
	}
	return stats, nil
}

// ApplyDiffStats assigns the given stats to the instance, preserving any
// previously-fetched Content so the Diff tab keeps showing it until the next
// UpdateDiffContent.
func (i *Instance) ApplyDiffStats(stats *git.DiffStats) {
	if !i.started {
		i.diffStats = nil
		return
	}
	if i.Status == Paused {
		return
	}
	if stats == nil {
		i.diffStats = nil
		return
	}
	if i.diffStats != nil {
		stats.Content = i.diffStats.Content
	}
	i.diffStats = stats
}

// UpdateDiffContent refreshes the full diff content (expensive) for this instance.
// Call this on demand when the Diff tab is visible, not on the metadata tick.
func (i *Instance) UpdateDiffContent() error {
	if !i.started || i.Status == Paused {
		return nil
	}
	stats := i.gitWorktree.Diff()
	if stats.Error != nil {
		if strings.Contains(stats.Error.Error(), "base commit SHA not set") {
			return nil
		}
		return fmt.Errorf("failed to get diff content: %w", stats.Error)
	}
	i.diffStats = stats
	i.lastDiffContentUpdate = time.Now()
	return nil
}

// LastDiffContentUpdate returns when the full diff content was last refreshed.
func (i *Instance) LastDiffContentUpdate() time.Time {
	return i.lastDiffContentUpdate
}

// GetDiffStats returns the current git diff statistics
func (i *Instance) GetDiffStats() *git.DiffStats {
	return i.diffStats
}

// SendPrompt sends a prompt to the tmux session
func (i *Instance) SendPrompt(prompt string) error {
	if !i.started {
		return fmt.Errorf("instance not started")
	}
	if i.tmuxSession == nil {
		return fmt.Errorf("tmux session not initialized")
	}
	if err := i.tmuxSession.SendKeys(prompt); err != nil {
		return fmt.Errorf("error sending keys to tmux session: %w", err)
	}

	// Brief pause to prevent carriage return from being interpreted as newline
	time.Sleep(100 * time.Millisecond)
	if err := i.tmuxSession.TapEnter(); err != nil {
		return fmt.Errorf("error tapping enter: %w", err)
	}

	return nil
}

// PreviewFullHistory captures the entire tmux pane output including full scrollback history
func (i *Instance) PreviewFullHistory() (string, error) {
	if !i.started || i.Status == Paused {
		return "", nil
	}
	return i.tmuxSession.CapturePaneContentWithOptions("-", "-")
}

// SetTmuxSession sets the tmux session for testing purposes
func (i *Instance) SetTmuxSession(session *tmux.TmuxSession) {
	i.tmuxSession = session
}

// GetTmuxSessionName returns the tmux session name for this instance.
func (i *Instance) GetTmuxSessionName() string {
	if i.tmuxSession == nil {
		return ""
	}
	return i.tmuxSession.GetSessionName()
}

// SendKeys sends keys to the tmux session
func (i *Instance) SendKeys(keys string) error {
	if !i.started || i.Status == Paused {
		return fmt.Errorf("cannot send keys to instance that has not been started or is paused")
	}
	return i.tmuxSession.SendKeys(keys)
}
