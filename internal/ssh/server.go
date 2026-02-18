package ssh

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/matoval/envclone/internal/exec"
	"github.com/matoval/envclone/internal/platform"
)

// SetupSSH installs and configures openssh-server inside the dev container.
func SetupSSH(ctx context.Context, runner *exec.Runner, plat platform.Platform, containerName, remoteUser string, port int) error {
	// Install openssh-server
	installCmd := "apt-get update && apt-get install -y openssh-server || dnf install -y openssh-server || apk add openssh"
	args := plat.NerdctlArgs("exec", containerName, "sh", "-c", installCmd)
	if _, err := runner.Run(ctx, args[0], args[1:]...); err != nil {
		return fmt.Errorf("installing openssh-server: %w", err)
	}

	// Create sshd run directory
	args = plat.NerdctlArgs("exec", containerName, "mkdir", "-p", "/run/sshd")
	if _, err := runner.Run(ctx, args[0], args[1:]...); err != nil {
		return fmt.Errorf("creating /run/sshd: %w", err)
	}

	// Configure sshd
	sshdConfig := fmt.Sprintf(`Port %d
PermitRootLogin yes
PasswordAuthentication no
PubkeyAuthentication yes
`, port)
	args = plat.NerdctlArgs("exec", containerName, "sh", "-c",
		fmt.Sprintf("echo '%s' > /etc/ssh/sshd_config.d/envclone.conf", sshdConfig))
	if _, err := runner.Run(ctx, args[0], args[1:]...); err != nil {
		return fmt.Errorf("configuring sshd: %w", err)
	}

	// Generate host keys
	args = plat.NerdctlArgs("exec", containerName, "ssh-keygen", "-A")
	if _, err := runner.Run(ctx, args[0], args[1:]...); err != nil {
		return fmt.Errorf("generating host keys: %w", err)
	}

	// Start sshd (only if not already running)
	checkCmd := "pgrep -x sshd > /dev/null 2>&1"
	args = plat.NerdctlArgs("exec", containerName, "sh", "-c", checkCmd)
	if _, err := runner.Run(ctx, args[0], args[1:]...); err != nil {
		args = plat.NerdctlArgs("exec", "-d", containerName, "/usr/sbin/sshd", "-D")
		if _, err := runner.Run(ctx, args[0], args[1:]...); err != nil {
			return fmt.Errorf("starting sshd: %w", err)
		}
	}

	return nil
}

// FindPublicKey searches ~/.ssh/ for the user's public key.
func FindPublicKey() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}

	candidates := []string{
		filepath.Join(home, ".ssh", "id_ed25519.pub"),
		filepath.Join(home, ".ssh", "id_rsa.pub"),
		filepath.Join(home, ".ssh", "id_ecdsa.pub"),
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			return strings.TrimSpace(string(data)), nil
		}
	}

	return "", fmt.Errorf("no SSH public key found in ~/.ssh/ (tried id_ed25519.pub, id_rsa.pub, id_ecdsa.pub)\nGenerate one with: ssh-keygen -t ed25519")
}

// InjectAuthorizedKey adds the given public key to the container's authorized_keys file.
func InjectAuthorizedKey(ctx context.Context, runner *exec.Runner, plat platform.Platform, containerName, remoteUser, pubKey string) error {
	sshDir := "/root/.ssh"
	if remoteUser != "" && remoteUser != "root" {
		sshDir = fmt.Sprintf("/home/%s/.ssh", remoteUser)
	}
	authKeysPath := sshDir + "/authorized_keys"

	// Create .ssh directory with correct permissions
	mkdirCmd := fmt.Sprintf("mkdir -p %s && chmod 700 %s", sshDir, sshDir)
	args := plat.NerdctlArgs("exec", containerName, "sh", "-c", mkdirCmd)
	if _, err := runner.Run(ctx, args[0], args[1:]...); err != nil {
		return fmt.Errorf("creating .ssh directory: %w", err)
	}

	// Check if key already exists (idempotent)
	checkCmd := fmt.Sprintf("grep -qF '%s' %s 2>/dev/null", pubKey, authKeysPath)
	args = plat.NerdctlArgs("exec", containerName, "sh", "-c", checkCmd)
	if _, err := runner.Run(ctx, args[0], args[1:]...); err == nil {
		return nil
	}

	// Append the key
	appendCmd := fmt.Sprintf("echo '%s' >> %s && chmod 600 %s", pubKey, authKeysPath, authKeysPath)
	args = plat.NerdctlArgs("exec", containerName, "sh", "-c", appendCmd)
	if _, err := runner.Run(ctx, args[0], args[1:]...); err != nil {
		return fmt.Errorf("injecting authorized key: %w", err)
	}

	return nil
}
