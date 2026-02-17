package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install prerequisites (nerdctl, rootless containerd)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		switch runtime.GOOS {
		case "linux":
			return setupLinux(ctx)
		case "darwin":
			return setupDarwin(ctx)
		default:
			return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
		}
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func setupLinux(ctx context.Context) error {
	// Check for nerdctl
	if _, err := exec.LookPath("nerdctl"); err != nil {
		fmt.Println("[1/3] Installing nerdctl-full...")
		if err := installNerdctlFull(ctx); err != nil {
			return fmt.Errorf("installing nerdctl-full: %w", err)
		}
		fmt.Println("      nerdctl-full installed to /usr/local")
	} else {
		fmt.Println("[1/3] nerdctl already installed, skipping")
	}

	// Check for containerd-rootless-setuptool.sh
	if _, err := exec.LookPath("containerd-rootless-setuptool.sh"); err != nil {
		return fmt.Errorf("containerd-rootless-setuptool.sh not found after install — check that /usr/local/bin is in your PATH")
	}

	// Check if rootless containerd is running
	out, _ := exec.CommandContext(ctx, "systemctl", "--user", "is-active", "containerd").Output()
	if strings.TrimSpace(string(out)) != "active" {
		fmt.Println("[2/3] Setting up rootless containerd...")
		cmd := exec.CommandContext(ctx, "containerd-rootless-setuptool.sh", "install")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("rootless containerd setup failed: %w", err)
		}
		fmt.Println("      rootless containerd is running")
	} else {
		fmt.Println("[2/3] rootless containerd already running, skipping")
	}

	// Check if buildkit is running
	out, _ = exec.CommandContext(ctx, "systemctl", "--user", "is-active", "buildkit").Output()
	if strings.TrimSpace(string(out)) != "active" {
		fmt.Println("[3/4] Setting up buildkit...")
		cmd := exec.CommandContext(ctx, "containerd-rootless-setuptool.sh", "install-buildkit")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("buildkit setup failed: %w", err)
		}
		fmt.Println("      buildkit is running")
	} else {
		fmt.Println("[3/4] buildkit already running, skipping")
	}

	// Verify
	fmt.Println("[4/4] Verifying...")
	verifyCmd := exec.CommandContext(ctx, "nerdctl", "run", "--rm", "hello-world")
	verifyCmd.Stdout = os.Stdout
	verifyCmd.Stderr = os.Stderr
	if err := verifyCmd.Run(); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	fmt.Println("\nSetup complete! You can now use 'envclone init' and 'envclone up'.")
	return nil
}

func installNerdctlFull(ctx context.Context) error {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "amd64"
	} else if arch == "arm64" {
		arch = "arm64"
	}

	// Get latest release tag from GitHub API
	version, err := getLatestNerdctlVersion(ctx)
	if err != nil {
		return fmt.Errorf("fetching latest nerdctl version: %w", err)
	}

	url := fmt.Sprintf("https://github.com/containerd/nerdctl/releases/download/v%s/nerdctl-full-%s-linux-%s.tar.gz", version, version, arch)
	fmt.Printf("      Downloading %s\n", url)

	// Download and extract — needs sudo for /usr/local
	script := fmt.Sprintf(`curl -fsSL "%s" -o /tmp/nerdctl-full.tar.gz && sudo tar -xzf /tmp/nerdctl-full.tar.gz -C /usr/local && rm /tmp/nerdctl-full.tar.gz`, url)
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getLatestNerdctlVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "curl", "-fsSL", "-H", "Accept: application/json", "https://api.github.com/repos/containerd/nerdctl/releases/latest")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// Simple extraction — find "tag_name":"v1.2.3"
	s := string(out)
	idx := strings.Index(s, `"tag_name"`)
	if idx == -1 {
		return "", fmt.Errorf("could not find tag_name in GitHub API response")
	}
	s = s[idx:]
	start := strings.Index(s, `"v`) + 2
	end := strings.Index(s[start:], `"`) + start
	if start < 2 || end <= start {
		return "", fmt.Errorf("could not parse version from GitHub API response")
	}
	return s[start:end], nil
}

func setupDarwin(ctx context.Context) error {
	// Check for brew
	if _, err := exec.LookPath("brew"); err != nil {
		return fmt.Errorf("Homebrew not found — install it from https://brew.sh")
	}

	// Check for lima
	if _, err := exec.LookPath("limactl"); err != nil {
		fmt.Println("[1/2] Installing Lima...")
		cmd := exec.CommandContext(ctx, "brew", "install", "lima")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("installing Lima: %w", err)
		}
	} else {
		fmt.Println("[1/2] Lima already installed, skipping")
	}

	// Create and start the envclone VM
	fmt.Println("[2/2] Creating Lima VM...")
	out, _ := exec.CommandContext(ctx, "limactl", "list", "--format", "{{.Name}}").Output()
	if !strings.Contains(string(out), "envclone") {
		cmd := exec.CommandContext(ctx, "limactl", "create",
			"--name=envclone",
			"--vm-type=vz",
			"--mount-type=virtiofs",
			"--mount-writable",
			"--containerd=user",
			"template://default",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("creating Lima VM: %w", err)
		}
	}

	startCmd := exec.CommandContext(ctx, "limactl", "start", "envclone")
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr
	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("starting Lima VM: %w", err)
	}

	fmt.Println("\nSetup complete! You can now use 'envclone init' and 'envclone up'.")
	return nil
}
