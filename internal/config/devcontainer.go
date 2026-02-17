package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type BuildConfig struct {
	Dockerfile string `json:"dockerfile"`
	Context    string `json:"context,omitempty"`
}

type DevContainer struct {
	Name              string            `json:"name"`
	Image             string            `json:"image,omitempty"`
	Build             *BuildConfig      `json:"build,omitempty"`
	WorkspaceFolder   string            `json:"workspaceFolder,omitempty"`
	WorkspaceMount    string            `json:"workspaceMount,omitempty"`
	ForwardPorts      []int             `json:"forwardPorts,omitempty"`
	PostCreateCommand string            `json:"postCreateCommand,omitempty"`
	PostStartCommand  string            `json:"postStartCommand,omitempty"`
	RemoteUser        string            `json:"remoteUser,omitempty"`
	Mounts            []string          `json:"mounts,omitempty"`
	Features          map[string]any    `json:"features,omitempty"`
	RunArgs           []string          `json:"runArgs,omitempty"`
	Services          []ServiceConfig   `json:"services,omitempty"`
}

type ServiceConfig struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Ports   []string `json:"ports,omitempty"`
	Env     []string `json:"env,omitempty"`
	Volumes []string `json:"volumes,omitempty"`
}

func Load(projectDir string) (*DevContainer, error) {
	configPath := filepath.Join(projectDir, ".devcontainer", "devcontainer.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading devcontainer.json: %w", err)
	}

	var cfg DevContainer
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing devcontainer.json: %w", err)
	}

	if cfg.Image == "" && cfg.Build == nil {
		return nil, fmt.Errorf("devcontainer.json: either \"image\" or \"build.dockerfile\" is required")
	}
	if cfg.Build != nil && cfg.Build.Dockerfile == "" {
		return nil, fmt.Errorf("devcontainer.json: \"build.dockerfile\" cannot be empty")
	}
	if cfg.Name == "" {
		cfg.Name = filepath.Base(projectDir)
	}

	return &cfg, nil
}
