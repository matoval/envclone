package platform

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Linux struct{}

func (l *Linux) Name() string { return "linux" }

func (l *Linux) NerdctlArgs(args ...string) []string {
	return append([]string{"nerdctl"}, args...)
}

func (l *Linux) EnsureRuntime(ctx context.Context) error {
	// Check if rootless containerd is running
	out, err := exec.CommandContext(ctx, "systemctl", "--user", "is-active", "containerd").Output()
	if err != nil || strings.TrimSpace(string(out)) != "active" {
		return fmt.Errorf("rootless containerd is not running\nStart it with: systemctl --user start containerd\nOr set it up with: containerd-rootless-setuptool.sh install")
	}
	return nil
}

func (l *Linux) MountArgs(hostPath, containerPath string) []string {
	return []string{"-v", fmt.Sprintf("%s:%s", hostPath, containerPath)}
}

func (l *Linux) SSHPort() int { return 2222 }

func (l *Linux) Cleanup(ctx context.Context) error { return nil }
