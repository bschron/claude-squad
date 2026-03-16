package session

import (
	"claude-squad/config"
	"encoding/json"
	"fmt"
	"time"
)

// InstanceData represents the serializable data of an Instance
type InstanceData struct {
	Title     string    `json:"title"`
	Path      string    `json:"path"`
	Branch    string    `json:"branch"`
	Status    Status    `json:"status"`
	Height    int       `json:"height"`
	Width     int       `json:"width"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	AutoYes         bool                `json:"auto_yes"`
	Effort          config.EffortLevel  `json:"effort,omitempty"`
	Model           config.ModelOption  `json:"model,omitempty"`
	SkipPermissions *bool               `json:"skip_permissions,omitempty"`

	Program   string          `json:"program"`
	Worktree  GitWorktreeData `json:"worktree"`
	DiffStats DiffStatsData   `json:"diff_stats"`
}

// GitWorktreeData represents the serializable data of a GitWorktree
type GitWorktreeData struct {
	RepoPath         string `json:"repo_path"`
	WorktreePath     string `json:"worktree_path"`
	SessionName      string `json:"session_name"`
	BranchName       string `json:"branch_name"`
	BaseCommitSHA    string `json:"base_commit_sha"`
	IsExistingBranch bool   `json:"is_existing_branch"`
}

// DiffStatsData represents the serializable data of a DiffStats
type DiffStatsData struct {
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	Content string `json:"content"`
}

// Storage handles saving and loading instances using the state interface
type Storage struct {
	state config.InstanceStorage
}

// NewStorage creates a new storage instance
func NewStorage(state config.InstanceStorage) (*Storage, error) {
	return &Storage{
		state: state,
	}, nil
}

// SaveInstances saves the list of instances to disk
func (s *Storage) SaveInstances(instances []*Instance) error {
	// Convert instances to InstanceData, skipping external instances
	data := make([]InstanceData, 0)
	for _, instance := range instances {
		if instance.Started() && !instance.IsExternal() {
			data = append(data, instance.ToInstanceData())
		}
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal instances: %w", err)
	}

	return s.state.SaveInstances(jsonData)
}

// LoadInstances loads the list of instances from disk
func (s *Storage) LoadInstances() ([]*Instance, error) {
	jsonData := s.state.GetInstances()

	var instancesData []InstanceData
	if err := json.Unmarshal(jsonData, &instancesData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instances: %w", err)
	}

	instances := make([]*Instance, len(instancesData))
	for i, data := range instancesData {
		instance, err := FromInstanceData(data)
		if err != nil {
			return nil, fmt.Errorf("failed to create instance %s: %w", data.Title, err)
		}
		instances[i] = instance
	}

	return instances, nil
}

// LoadInstancesForProject loads instances filtered by repository path.
// Sessions with an empty RepoPath (created before project filtering) are
// included so they remain visible after upgrade.
func (s *Storage) LoadInstancesForProject(repoPath string) ([]*Instance, error) {
	all, err := s.LoadInstances()
	if err != nil {
		return nil, err
	}
	var filtered []*Instance
	for _, inst := range all {
		data := inst.ToInstanceData()
		if data.Worktree.RepoPath == repoPath || data.Worktree.RepoPath == "" {
			filtered = append(filtered, inst)
		}
	}
	return filtered, nil
}

// SaveInstancesForProject saves the project's instances while preserving
// instances that belong to other projects.
func (s *Storage) SaveInstancesForProject(repoPath string, projectInstances []*Instance) error {
	all, err := s.LoadInstances()
	if err != nil {
		// If we can't load existing data, fall back to saving what we have.
		return s.SaveInstances(projectInstances)
	}

	// Keep instances from other projects (non-empty, different repo path).
	var merged []*Instance
	for _, inst := range all {
		data := inst.ToInstanceData()
		if data.Worktree.RepoPath != "" && data.Worktree.RepoPath != repoPath {
			merged = append(merged, inst)
		}
	}

	// Append the current project's instances.
	merged = append(merged, projectInstances...)

	return s.SaveInstances(merged)
}

// DeleteInstance removes an instance from storage
func (s *Storage) DeleteInstance(title string) error {
	instances, err := s.LoadInstances()
	if err != nil {
		return fmt.Errorf("failed to load instances: %w", err)
	}

	found := false
	newInstances := make([]*Instance, 0)
	for _, instance := range instances {
		data := instance.ToInstanceData()
		if data.Title != title {
			newInstances = append(newInstances, instance)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("instance not found: %s", title)
	}

	return s.SaveInstances(newInstances)
}

// UpdateInstance updates an existing instance in storage
func (s *Storage) UpdateInstance(instance *Instance) error {
	instances, err := s.LoadInstances()
	if err != nil {
		return fmt.Errorf("failed to load instances: %w", err)
	}

	data := instance.ToInstanceData()
	found := false
	for i, existing := range instances {
		existingData := existing.ToInstanceData()
		if existingData.Title == data.Title {
			instances[i] = instance
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("instance not found: %s", data.Title)
	}

	return s.SaveInstances(instances)
}

// DeleteAllInstances removes all stored instances
func (s *Storage) DeleteAllInstances() error {
	return s.state.DeleteAllInstances()
}
