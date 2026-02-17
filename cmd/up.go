package cmd

import (
	"fmt"

	"github.com/matoval/envclone/internal/config"
	"github.com/matoval/envclone/internal/container"
	"github.com/matoval/envclone/internal/exec"
	"github.com/matoval/envclone/internal/platform"
	"github.com/matoval/envclone/internal/state"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the dev environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		dir, err := getProjectDir()
		if err != nil {
			return err
		}

		cfg, err := config.Load(dir)
		if err != nil {
			return err
		}

		plat, err := platform.Detect()
		if err != nil {
			return err
		}

		if err := plat.EnsureRuntime(ctx); err != nil {
			return fmt.Errorf("runtime not ready: %w", err)
		}

		runner := &exec.Runner{}
		mgr := &container.Manager{
			Platform:   plat,
			Runner:     runner,
			Config:     cfg,
			ProjectDir: dir,
		}

		env, err := mgr.Up(ctx)
		if err != nil {
			return err
		}

		if err := state.Save(dir, env); err != nil {
			return fmt.Errorf("saving state: %w", err)
		}

		fmt.Println("Environment is up!")
		fmt.Printf("  Dev container: %s\n", env.DevContainerID)
		fmt.Printf("  Services:      %d\n", len(env.ServiceIDs))
		fmt.Println("\nRun 'envclone shell' to open a shell.")
		fmt.Println("Run 'envclone ssh-config' to get VS Code SSH config.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}
