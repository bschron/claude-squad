package session

import (
	"path/filepath"
	"sort"
)

// ProjectInfo describes a project discovered from state.json.
type ProjectInfo struct {
	RepoPath     string
	DisplayName  string
	SessionCount int
}

// DiscoverAllProjects returns all unique projects found in state.json.
func DiscoverAllProjects(storage *Storage) ([]ProjectInfo, error) {
	all, err := storage.LoadInstances()
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, inst := range all {
		rp := inst.ToInstanceData().Worktree.RepoPath
		if rp == "" {
			continue
		}
		counts[rp]++
	}

	var projects []ProjectInfo
	for rp, count := range counts {
		projects = append(projects, ProjectInfo{
			RepoPath:     rp,
			DisplayName:  filepath.Base(rp),
			SessionCount: count,
		})
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].DisplayName < projects[j].DisplayName
	})

	return projects, nil
}
