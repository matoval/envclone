package container

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/matoval/envclone/internal/config"
	"github.com/matoval/envclone/internal/exec"
	"github.com/matoval/envclone/internal/network"
	"github.com/matoval/envclone/internal/platform"
	"github.com/matoval/envclone/internal/state"
)

type ContainerInfo struct {
	Name   string
	Role   string
	Status string
}

type Manager struct {
	Platform   platform.Platform
	Runner     *exec.Runner
	Config     *config.DevContainer
	ProjectDir string
}

func (m *Manager) projectName() string {
	return filepath.Base(m.ProjectDir)
}

func (m *Manager) Up(ctx context.Context) (*state.Environment, error) {
	name := m.projectName()

	// Clean up any existing containers for this project
	m.removeExisting(ctx, name)

	// Build image from Dockerfile if configured
	if m.Config.Build != nil {
		if err := m.buildImage(ctx, name); err != nil {
			return nil, fmt.Errorf("building image: %w", err)
		}
	}

	// Create shared network namespace with SSH port published
	netNSID, err := network.CreateNetNS(ctx, m.Runner, m.Platform, name, m.Platform.SSHPort())
	if err != nil {
		return nil, err
	}
	netNSContainer := fmt.Sprintf("envclone-%s-netns", name)

	// Create service containers
	var serviceIDs []string
	for _, svc := range m.Config.Services {
		id, err := m.createService(ctx, name, netNSContainer, svc)
		if err != nil {
			return nil, fmt.Errorf("creating service %s: %w", svc.Name, err)
		}
		serviceIDs = append(serviceIDs, id)
	}

	// Create dev container
	devID, err := m.createDevContainer(ctx, name, netNSContainer)
	if err != nil {
		return nil, fmt.Errorf("creating dev container: %w", err)
	}

	// Run postCreateCommand if set
	if m.Config.PostCreateCommand != "" {
		devContainer := fmt.Sprintf("envclone-%s-dev", name)
		args := m.Platform.NerdctlArgs("exec", devContainer, "sh", "-c", m.Config.PostCreateCommand)
		if _, err := m.Runner.Run(ctx, args[0], args[1:]...); err != nil {
			fmt.Printf("Warning: postCreateCommand failed: %v\n", err)
		}
	}

	// Run postStartCommand if set
	if m.Config.PostStartCommand != "" {
		devContainer := fmt.Sprintf("envclone-%s-dev", name)
		args := m.Platform.NerdctlArgs("exec", devContainer, "sh", "-c", m.Config.PostStartCommand)
		if _, err := m.Runner.Run(ctx, args[0], args[1:]...); err != nil {
			fmt.Printf("Warning: postStartCommand failed: %v\n", err)
		}
	}

	remoteUser := m.Config.RemoteUser
	if remoteUser == "" {
		remoteUser = "root"
	}

	return &state.Environment{
		ProjectName:    name,
		ProjectDir:     m.ProjectDir,
		DevContainerID: devID,
		NetNSID:        netNSID,
		ServiceIDs:     serviceIDs,
		SSHPort:        m.Platform.SSHPort(),
		RemoteUser:     remoteUser,
	}, nil
}

func (m *Manager) buildImage(ctx context.Context, projectName string) error {
	tag := fmt.Sprintf("envclone-%s:latest", projectName)
	dockerfilePath := m.Config.Build.Dockerfile

	// Resolve relative Dockerfile path against the .devcontainer directory
	if !filepath.IsAbs(dockerfilePath) {
		dockerfilePath = filepath.Join(m.ProjectDir, ".devcontainer", dockerfilePath)
	}

	buildContext := filepath.Dir(dockerfilePath)
	if m.Config.Build.Context != "" {
		buildContext = m.Config.Build.Context
		if !filepath.IsAbs(buildContext) {
			buildContext = filepath.Join(m.ProjectDir, ".devcontainer", buildContext)
		}
	}

	fmt.Printf("Building image %s from %s...\n", tag, dockerfilePath)
	args := m.Platform.NerdctlArgs("build", "-t", tag, "-f", dockerfilePath, buildContext)
	_, err := m.Runner.Run(ctx, args[0], args[1:]...)
	return err
}

func (m *Manager) createDevContainer(ctx context.Context, projectName, netNSContainer string) (string, error) {
	containerName := fmt.Sprintf("envclone-%s-dev", projectName)

	// Determine host source path and container mount target
	hostPath := m.ProjectDir
	if m.Config.WorkspaceFolder != "" {
		hostPath = m.Config.WorkspaceFolder
	}
	containerPath := "/workspace"
	if m.Config.WorkspaceMount != "" {
		containerPath = m.Config.WorkspaceMount
	}

	mountArgs := m.Platform.MountArgs(hostPath, containerPath)

	args := m.Platform.NerdctlArgs("run", "-d",
		"--name", containerName,
		"--label", fmt.Sprintf("envclone.project=%s", projectName),
		"--label", "envclone.role=dev",
		"--network", fmt.Sprintf("container:%s", netNSContainer),
	)
	args = append(args, mountArgs...)
	args = append(args, "-w", containerPath, "--init")
	args = append(args, m.Config.RunArgs...)

	image := m.Config.Image
	if m.Config.Build != nil {
		image = fmt.Sprintf("envclone-%s:latest", projectName)
	}
	args = append(args, image, "sleep", "infinity")

	id, err := m.Runner.Run(ctx, args[0], args[1:]...)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (m *Manager) createService(ctx context.Context, projectName, netNSContainer string, svc config.ServiceConfig) (string, error) {
	containerName := fmt.Sprintf("envclone-%s-%s", projectName, svc.Name)

	args := m.Platform.NerdctlArgs("run", "-d",
		"--name", containerName,
		"--label", fmt.Sprintf("envclone.project=%s", projectName),
		"--label", "envclone.role=service",
		"--network", fmt.Sprintf("container:%s", netNSContainer),
	)

	for _, env := range svc.Env {
		args = append(args, "-e", env)
	}

	for _, vol := range svc.Volumes {
		args = append(args, "-v", vol)
	}

	args = append(args, svc.Image)

	id, err := m.Runner.Run(ctx, args[0], args[1:]...)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (m *Manager) removeExisting(ctx context.Context, projectName string) {
	args := m.Platform.NerdctlArgs("ps", "-a", "--filter", fmt.Sprintf("label=envclone.project=%s", projectName), "--format", "{{.ID}}")
	out, err := m.Runner.Run(ctx, args[0], args[1:]...)
	if err != nil || out == "" {
		return
	}
	ids := strings.Fields(out)
	if len(ids) > 0 {
		rmArgs := m.Platform.NerdctlArgs(append([]string{"rm", "-f"}, ids...)...)
		m.Runner.Run(ctx, rmArgs[0], rmArgs[1:]...)
	}
}

// IsRunning checks if the dev container is currently running.
func (m *Manager) IsRunning(ctx context.Context, env *state.Environment) (bool, error) {
	devContainer := fmt.Sprintf("envclone-%s-dev", env.ProjectName)
	args := m.Platform.NerdctlArgs("inspect", "--format", "{{.State.Running}}", devContainer)
	out, err := m.Runner.Run(ctx, args[0], args[1:]...)
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(out) == "true", nil
}

func (m *Manager) Down(ctx context.Context, env *state.Environment) error {
	// Stop and remove all containers by label
	name := env.ProjectName
	args := m.Platform.NerdctlArgs("ps", "-a", "--filter", fmt.Sprintf("label=envclone.project=%s", name), "--format", "{{.ID}}")
	out, err := m.Runner.Run(ctx, args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("listing containers: %w", err)
	}

	ids := strings.Fields(out)
	if len(ids) > 0 {
		rmArgs := m.Platform.NerdctlArgs(append([]string{"rm", "-f"}, ids...)...)
		if _, err := m.Runner.Run(ctx, rmArgs[0], rmArgs[1:]...); err != nil {
			return fmt.Errorf("removing containers: %w", err)
		}
	}

	return nil
}

func (m *Manager) Shell(ctx context.Context, env *state.Environment) error {
	devContainer := fmt.Sprintf("envclone-%s-dev", env.ProjectName)
	args := m.Platform.NerdctlArgs("exec", "-it", devContainer, "/bin/bash")
	return m.Runner.RunInteractive(ctx, args[0], args[1:]...)
}

func (m *Manager) Exec(ctx context.Context, env *state.Environment, command []string) error {
	devContainer := fmt.Sprintf("envclone-%s-dev", env.ProjectName)
	nerdctlArgs := []string{"exec", devContainer}
	nerdctlArgs = append(nerdctlArgs, command...)
	args := m.Platform.NerdctlArgs(nerdctlArgs...)
	return m.Runner.RunInteractive(ctx, args[0], args[1:]...)
}

func (m *Manager) Status(ctx context.Context, env *state.Environment) ([]ContainerInfo, error) {
	args := m.Platform.NerdctlArgs("ps", "-a",
		"--filter", fmt.Sprintf("label=envclone.project=%s", env.ProjectName),
		"--format", "{{.Names}}\t{{.Labels}}\t{{.Status}}",
	)
	out, err := m.Runner.Run(ctx, args[0], args[1:]...)
	if err != nil {
		return nil, err
	}

	var infos []ContainerInfo
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		role := "unknown"
		if strings.Contains(parts[1], "envclone.role=dev") {
			role = "dev"
		} else if strings.Contains(parts[1], "envclone.role=service") {
			role = "service"
		} else if strings.Contains(parts[1], "envclone.role=netns") {
			role = "netns"
		}
		infos = append(infos, ContainerInfo{
			Name:   parts[0],
			Role:   role,
			Status: parts[2],
		})
	}
	return infos, nil
}
