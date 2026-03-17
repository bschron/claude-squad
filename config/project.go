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

const (
	DefaultInstanceLimit = 10
	MinInstanceLimit     = 1
	MaxInstanceLimit     = 50
)

// ValidInstanceLimits is the ordered list of valid instance limit options.
var ValidInstanceLimits = []int{5, 10, 15, 20, 25, 50}

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

// SoundOption represents a system sound for alerts.
type SoundOption string

const (
	SoundBasso     SoundOption = "Basso"
	SoundBlow      SoundOption = "Blow"
	SoundBottle    SoundOption = "Bottle"
	SoundFrog      SoundOption = "Frog"
	SoundFunk      SoundOption = "Funk"
	SoundGlass     SoundOption = "Glass"
	SoundHero      SoundOption = "Hero"
	SoundMorse     SoundOption = "Morse"
	SoundPing      SoundOption = "Ping"
	SoundPop       SoundOption = "Pop"
	SoundPurr      SoundOption = "Purr"
	SoundSosumi    SoundOption = "Sosumi"
	SoundSubmarine SoundOption = "Submarine"
	SoundTink      SoundOption = "Tink"
)

// ValidSoundOptions is the ordered list of valid sound options.
var ValidSoundOptions = []SoundOption{
	SoundBasso, SoundBlow, SoundBottle, SoundFrog, SoundFunk,
	SoundGlass, SoundHero, SoundMorse, SoundPing, SoundPop,
	SoundPurr, SoundSosumi, SoundSubmarine, SoundTink,
}

// DefaultSound is the default alert sound.
var DefaultSound = SoundPop

// SoundDisplayLabels maps sound options to their display labels.
var SoundDisplayLabels = map[SoundOption]string{
	SoundBasso:     "basso",
	SoundBlow:      "blow",
	SoundBottle:    "bottle",
	SoundFrog:      "frog",
	SoundFunk:      "funk",
	SoundGlass:     "glass",
	SoundHero:      "hero",
	SoundMorse:     "morse",
	SoundPing:      "ping",
	SoundPop:       "pop",
	SoundPurr:      "purr",
	SoundSosumi:    "sosumi",
	SoundSubmarine: "submarine",
	SoundTink:      "tink",
}

// ProjectConfig represents per-project configuration stored at the git repo root.
type ProjectConfig struct {
	DefaultEffort   EffortLevel `json:"default_effort,omitempty"`
	DefaultModel    ModelOption `json:"default_model,omitempty"`
	SkipPermissions *bool       `json:"skip_permissions,omitempty"`
	SoundAlert      *bool       `json:"sound_alert,omitempty"`
	AlertSound      SoundOption `json:"alert_sound,omitempty"`
	InstanceLimit   *int        `json:"instance_limit,omitempty"`
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

// GetInstanceLimit returns the effective instance limit (nil defaults to DefaultInstanceLimit).
func (c *ProjectConfig) GetInstanceLimit() int {
	if c.InstanceLimit == nil {
		return DefaultInstanceLimit
	}
	return *c.InstanceLimit
}

// SetInstanceLimit sets the instance limit value.
func (c *ProjectConfig) SetInstanceLimit(v int) {
	if v < MinInstanceLimit {
		v = MinInstanceLimit
	}
	if v > MaxInstanceLimit {
		v = MaxInstanceLimit
	}
	c.InstanceLimit = &v
}

// GetSoundAlert returns the effective sound alert value (nil defaults to false).
func (c *ProjectConfig) GetSoundAlert() bool {
	if c.SoundAlert == nil {
		return false
	}
	return *c.SoundAlert
}

// SetSoundAlert sets the sound alert value.
func (c *ProjectConfig) SetSoundAlert(v bool) {
	c.SoundAlert = &v
}

// GetAlertSound returns the effective alert sound (empty defaults to DefaultSound).
func (c *ProjectConfig) GetAlertSound() SoundOption {
	if c.AlertSound == "" {
		return DefaultSound
	}
	return c.AlertSound
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

	// Validate the loaded alert sound
	if cfg.AlertSound != "" && !isValidSound(cfg.AlertSound) {
		cfg.AlertSound = ""
	}

	// Validate the loaded instance limit
	if cfg.InstanceLimit != nil {
		if *cfg.InstanceLimit < MinInstanceLimit || *cfg.InstanceLimit > MaxInstanceLimit {
			cfg.InstanceLimit = nil
		}
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

func isValidSound(s SoundOption) bool {
	for _, v := range ValidSoundOptions {
		if v == s {
			return true
		}
	}
	return false
}
