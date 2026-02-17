package platform

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Darwin struct{}

func (d *Darwin) Name() string { return "darwin" }

func (d *Darwin) NerdctlArgs(args ...string) []string {
	limaArgs := []string{"limactl", "shell", "envclone", "--", "nerdctl"}
	return append(limaArgs, args...)
}

func (d *Darwin) EnsureRuntime(ctx context.Context) error {
	// Check if Lima VM "envclone" exists and is running
	out, err := exec.CommandContext(ctx, "limactl", "list", "--format", "{{.Name}}:{{.Status}}").Output()
	if err != nil {
		return fmt.Errorf("failed to list Lima VMs: %w", err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && parts[0] == "envclone" {
			if parts[1] == "Running" {
				return nil
			}
			// VM exists but not running
			if _, err := exec.CommandContext(ctx, "limactl", "start", "envclone").Output(); err != nil {
				return fmt.Errorf("failed to start Lima VM: %w", err)
			}
			return nil
		}
	}

	// VM doesn't exist — create it
	createArgs := []string{
		"limactl", "create",
		"--name=envclone",
		"--vm-type=vz",
		"--mount-type=virtiofs",
		"--mount-writable",
		"--containerd=user",
		"template://default",
	}
	cmd := exec.CommandContext(ctx, createArgs[0], createArgs[1:]...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create Lima VM: %w\n%s", err, string(out))
	}

	if _, err := exec.CommandContext(ctx, "limactl", "start", "envclone").Output(); err != nil {
		return fmt.Errorf("failed to start Lima VM after creation: %w", err)
	}

	return nil
}

func (d *Darwin) MountArgs(hostPath, containerPath string) []string {
	return []string{"-v", fmt.Sprintf("%s:%s", hostPath, containerPath)}
}

func (d *Darwin) SSHPort() int { return 2222 }

func (d *Darwin) Cleanup(ctx context.Context) error { return nil }
