package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const ProjectConfigFileName = ".claude-squad.json"

// EffortLevel represents the effort level for Claude Code.
type EffortLevel string

const (
	EffortLow    EffortLevel = "low"
	EffortMedium EffortLevel = "medium"
	EffortHigh   EffortLevel = "high"
	EffortMax    EffortLevel = "max"
)

// ValidEffortLevels is the ordered list of valid effort levels.
var ValidEffortLevels = []EffortLevel{EffortLow, EffortMedium, EffortHigh, EffortMax}

// DefaultEffortLevel is the default effort level when none is configured.
var DefaultEffortLevel = EffortMedium

// ModelOption represents the model option for Claude Code.
type ModelOption string

const (
	ModelDefault ModelOption = ""
	ModelSonnet  ModelOption = "sonnet"
	ModelOpus    ModelOption = "opus"
	ModelHaiku   ModelOption = "haiku"
)

// ValidModelOptions is the ordered list of valid model options.
var ValidModelOptions = []ModelOption{ModelDefault, ModelSonnet, ModelOpus, ModelHaiku}

// ModelDisplayLabels maps model options to their display labels.
var ModelDisplayLabels = map[ModelOption]string{
	ModelDefault: "default",
	ModelSonnet:  "sonnet",
	ModelOpus:    "opus",
	ModelHaiku:   "haiku",
}

// ModelOptionFromDisplay returns the ModelOption for a display label.
func ModelOptionFromDisplay(label string) ModelOption {
	for opt, lbl := range ModelDisplayLabels {
		if lbl == label {
			return opt
		}
	}
	return ModelDefault
}

// ProjectConfig represents per-project configuration stored at the git repo root.
type ProjectConfig struct {
	DefaultEffort   EffortLevel `json:"default_effort,omitempty"`
	DefaultModel    ModelOption `json:"default_model,omitempty"`
	SkipPermissions *bool       `json:"skip_permissions,omitempty"`
}

// GetSkipPermissions returns the effective skip permissions value (nil defaults to true).
func (c *ProjectConfig) GetSkipPermissions() bool {
	if c.SkipPermissions == nil {
		return true
	}
	return *c.SkipPermissions
}

// SetSkipPermissions sets the skip permissions value.
func (c *ProjectConfig) SetSkipPermissions(v bool) {
	c.SkipPermissions = &v
}

// LoadProjectConfig reads {gitRoot}/.claude-squad.json, returning defaults if missing.
func LoadProjectConfig(gitRoot string) *ProjectConfig {
	cfg := &ProjectConfig{
		DefaultEffort: DefaultEffortLevel,
	}

	configPath := filepath.Join(gitRoot, ProjectConfigFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return cfg
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return &ProjectConfig{DefaultEffort: DefaultEffortLevel}
	}

	// Validate the loaded effort level
	if !isValidEffort(cfg.DefaultEffort) {
		cfg.DefaultEffort = DefaultEffortLevel
	}

	// Validate the loaded model option
	if !isValidModel(cfg.DefaultModel) {
		cfg.DefaultModel = ModelDefault
	}

	return cfg
}

// SaveProjectConfig writes the project config to {gitRoot}/.claude-squad.json.
func SaveProjectConfig(gitRoot string, cfg *ProjectConfig) error {
	configPath := filepath.Join(gitRoot, ProjectConfigFileName)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func isValidEffort(e EffortLevel) bool {
	for _, v := range ValidEffortLevels {
		if v == e {
			return true
		}
	}
	return false
}

func isValidModel(m ModelOption) bool {
	for _, v := range ValidModelOptions {
		if v == m {
			return true
		}
	}
	return false
}
