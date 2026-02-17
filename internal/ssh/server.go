package ssh

import (
	"context"
	"fmt"

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

	// Start sshd
	args = plat.NerdctlArgs("exec", "-d", containerName, "/usr/sbin/sshd", "-D")
	if _, err := runner.Run(ctx, args[0], args[1:]...); err != nil {
		return fmt.Errorf("starting sshd: %w", err)
	}

	return nil
}
