package workflows

import (
	"gopkg.in/yaml.v2"
)

// Workflow definition
type Workflow struct {
	Path          string        `yaml:"-"`
	Name          string        `yaml:"Name"`
	SchemaVersion string        `yaml:"SchemaVersion"`
	Actions       yaml.MapSlice `yaml:"Actions"`
	Compute       struct {
		Type           string `yaml:"Type"`
		SharedInstance bool   `yaml:"SharedInstance"`
	} `yaml:"Compute"`
}

// ActionOrGroup is a union of types Action and ActionGroup. Only 1 should be present
type ActionOrGroup struct {
	Action      `yaml:",inline"`
	ActionGroup `yaml:",inline"`
}

// Action defines a single action to run
type Action struct {
	Identifier    string         `yaml:"Identifier"`
	DependsOn     []string       `yaml:"DependsOn"`
	Configuration map[string]any `yaml:"Configuration"`
	Inputs        struct {
		Sources   []string `yaml:"Sources"`
		Artifacts []string `yaml:"Artifacts"`
		Variables []struct {
			Name  string `yaml:"Name"`
			Value string `yaml:"Value"`
		} `yaml:"Variables"`
	} `yaml:"Inputs"`
	Outputs struct {
		Sources   []string          `yaml:"Sources"`
		Artifacts []*OutputArtifact `yaml:"Artifacts"`
		Variables []string          `yaml:"Variables"`
	} `yaml:"Outputs"`
	Caching struct {
		FileCaching FileCaching `yaml:"FileCaching"`
	} `yaml:"Caching"`
	Environment struct {
		Name        string `yaml:"Name"`
		Connections []struct {
			Name string `yaml:"Name"`
			Role string `yaml:"Role"`
		} `yaml:"Connections"`
	} `yaml:"Environment"`
}

// ActionGroup is a grouping of Actions
type ActionGroup struct {
	Actions map[string]*Action `yaml:"Actions"`
}

// OutputArfifact describes an artifact to collect at the end of a plan execution
type OutputArtifact struct {
	Name  string `yaml:"Name"`  // Name of the output artifact
	Files any    `yaml:"Files"` // List of file patterns to include
}

// FileCaching contains a map of [FileCachingEntry]
type FileCaching map[string]FileCachingEntry

// FileCachingEntry describes a cache configuration
type FileCachingEntry struct {
	Path        string   `yaml:"Path"`        // Path to cache
	RestoreKeys []string `yaml:"RestoreKeys"` // Fallback cache keys if this one misses
}
