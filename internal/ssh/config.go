package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateConfigBlock returns the SSH config text block for a project,
// wrapped in marker comments for idempotent updates.
func GenerateConfigBlock(projectName string, port int, remoteUser string) string {
	startMarker := fmt.Sprintf("# --- envclone: %s ---", projectName)
	endMarker := fmt.Sprintf("# --- /envclone: %s ---", projectName)
	return fmt.Sprintf(`%s
Host envclone-%s
  HostName localhost
  Port %d
  User %s
  StrictHostKeyChecking no
  UserKnownHostsFile /dev/null
%s`, startMarker, projectName, port, remoteUser, endMarker)
}

// WriteSSHConfig writes or updates the envclone block in ~/.ssh/config.
// If a block for this project already exists, it is replaced.
func WriteSSHConfig(projectName string, port int, remoteUser string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding home directory: %w", err)
	}

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return fmt.Errorf("creating ~/.ssh directory: %w", err)
	}

	configPath := filepath.Join(sshDir, "config")
	block := GenerateConfigBlock(projectName, port, remoteUser)

	existing, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading SSH config: %w", err)
	}

	content := string(existing)
	startMarker := fmt.Sprintf("# --- envclone: %s ---", projectName)
	endMarker := fmt.Sprintf("# --- /envclone: %s ---", projectName)

	startIdx := strings.Index(content, startMarker)
	endIdx := strings.Index(content, endMarker)

	if startIdx >= 0 && endIdx >= 0 {
		content = content[:startIdx] + block + content[endIdx+len(endMarker):]
	} else {
		if len(content) > 0 && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if len(content) > 0 {
			content += "\n"
		}
		content += block + "\n"
	}

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing SSH config: %w", err)
	}

	return nil
}
