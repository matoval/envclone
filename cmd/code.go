package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/matoval/envclone/internal/config"
	"github.com/matoval/envclone/internal/container"
	internalExec "github.com/matoval/envclone/internal/exec"
	"github.com/matoval/envclone/internal/platform"
	"github.com/matoval/envclone/internal/ssh"
	"github.com/matoval/envclone/internal/state"
	"github.com/spf13/cobra"
)

var codeCmd = &cobra.Command{
	Use:   "code",
	Short: "Open VS Code connected to the dev container via SSH",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		dir, err := getProjectDir()
		if err != nil {
			return err
		}

		env, err := state.Load(dir)
		if err != nil {
			return fmt.Errorf("no environment found (run 'envclone up' first): %w", err)
		}

		plat, err := platform.Detect()
		if err != nil {
			return err
		}

		runner := &internalExec.Runner{}
		mgr := &container.Manager{
			Platform:   plat,
			Runner:     runner,
			ProjectDir: dir,
		}

		running, err := mgr.IsRunning(ctx, env)
		if err != nil {
			return fmt.Errorf("checking container status: %w", err)
		}
		if !running {
			return fmt.Errorf("dev container is not running (run 'envclone up' first)")
		}

		devContainer := fmt.Sprintf("envclone-%s-dev", env.ProjectName)

		fmt.Println("Setting up SSH in dev container...")
		if err := ssh.SetupSSH(ctx, runner, plat, devContainer, env.RemoteUser, env.SSHPort); err != nil {
			return fmt.Errorf("setting up SSH: %w", err)
		}

		pubKey, err := ssh.FindPublicKey()
		if err != nil {
			return err
		}
		if err := ssh.InjectAuthorizedKey(ctx, runner, plat, devContainer, env.RemoteUser, pubKey); err != nil {
			return fmt.Errorf("injecting SSH key: %w", err)
		}

		if err := ssh.WriteSSHConfig(env.ProjectName, env.SSHPort, env.RemoteUser); err != nil {
			return fmt.Errorf("writing SSH config: %w", err)
		}
		fmt.Printf("Updated ~/.ssh/config with host envclone-%s\n", env.ProjectName)

		workspaceMount := "/workspace"
		cfg, err := config.Load(dir)
		if err == nil && cfg.WorkspaceMount != "" {
			workspaceMount = cfg.WorkspaceMount
		}

		hostAlias := fmt.Sprintf("envclone-%s", env.ProjectName)
		folderURI := fmt.Sprintf("vscode-remote://ssh-remote+%s%s", hostAlias, workspaceMount)
		fmt.Printf("Opening VS Code: %s\n", folderURI)

		codePath := findVSCode()
		if codePath == "" {
			return fmt.Errorf("VS Code 'code' command not found\nInstall it via: VS Code > Command Palette > 'Shell Command: Install code command in PATH'")
		}

		vscodeCmd := exec.CommandContext(ctx, codePath, "--folder-uri", folderURI)
		if err := vscodeCmd.Start(); err != nil {
			return fmt.Errorf("launching VS Code: %w", err)
		}

		return nil
	},
}

// findVSCode returns the path to the VS Code CLI, checking PATH first
// then falling back to known install locations on macOS and Linux.
func findVSCode() string {
	if path, err := exec.LookPath("code"); err == nil {
		return path
	}

	candidates := []string{
		"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
		"/usr/share/code/bin/code",
		"/snap/bin/code",
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func init() {
	rootCmd.AddCommand(codeCmd)
}
