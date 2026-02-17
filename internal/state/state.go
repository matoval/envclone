package state

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Environment struct {
	ProjectName    string   `json:"projectName"`
	ProjectDir     string   `json:"projectDir"`
	DevContainerID string   `json:"devContainerID"`
	NetNSID        string   `json:"netNSID"`
	ServiceIDs     []string `json:"serviceIDs"`
	SSHPort        int      `json:"sshPort"`
	RemoteUser     string   `json:"remoteUser"`
}

func stateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "envclone")
	return dir, os.MkdirAll(dir, 0o755)
}

func stateFile(projectDir string) (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(projectDir)))[:12]
	return filepath.Join(dir, hash+".json"), nil
}

func Save(projectDir string, env *Environment) error {
	path, err := stateFile(projectDir)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func Load(projectDir string) (*Environment, error) {
	path, err := stateFile(projectDir)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var env Environment
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

func Remove(projectDir string) error {
	path, err := stateFile(projectDir)
	if err != nil {
		return err
	}
	return os.Remove(path)
}
