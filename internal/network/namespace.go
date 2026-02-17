package network

import (
	"context"
	"fmt"

	"github.com/matoval/envclone/internal/exec"
	"github.com/matoval/envclone/internal/platform"
)

// CreateNetNS creates a pause container that provides a shared network namespace.
// All dev and service containers join this namespace with --network=container:<id>.
func CreateNetNS(ctx context.Context, runner *exec.Runner, plat platform.Platform, projectName string) (string, error) {
	name := fmt.Sprintf("envclone-%s-netns", projectName)
	args := plat.NerdctlArgs(
		"run", "-d",
		"--name", name,
		"--hostname", projectName,
		"--label", fmt.Sprintf("envclone.project=%s", projectName),
		"--label", "envclone.role=netns",
		"registry.k8s.io/pause:3.10",
	)

	id, err := runner.Run(ctx, args[0], args[1:]...)
	if err != nil {
		return "", fmt.Errorf("creating network namespace container: %w", err)
	}

	return id, nil
}

// RemoveNetNS stops and removes the network namespace container.
func RemoveNetNS(ctx context.Context, runner *exec.Runner, plat platform.Platform, projectName string) error {
	name := fmt.Sprintf("envclone-%s-netns", projectName)
	args := plat.NerdctlArgs("rm", "-f", name)
	_, err := runner.Run(ctx, args[0], args[1:]...)
	return err
}
