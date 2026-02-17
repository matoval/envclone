package cmd

import (
	"fmt"

	"github.com/matoval/envclone/internal/container"
	"github.com/matoval/envclone/internal/exec"
	"github.com/matoval/envclone/internal/platform"
	"github.com/matoval/envclone/internal/state"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec [command...]",
	Short: "Execute a command in the dev container",
	Args:  cobra.MinimumNArgs(1),
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

		runner := &exec.Runner{}
		mgr := &container.Manager{
			Platform:   plat,
			Runner:     runner,
			ProjectDir: dir,
		}

		return mgr.Exec(ctx, env, args)
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}
