package cmd

import (
	"fmt"

	"github.com/matoval/envclone/internal/container"
	"github.com/matoval/envclone/internal/exec"
	"github.com/matoval/envclone/internal/platform"
	"github.com/matoval/envclone/internal/state"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop the dev environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		dir, err := getProjectDir()
		if err != nil {
			return err
		}

		env, err := state.Load(dir)
		if err != nil {
			return fmt.Errorf("no environment found: %w", err)
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

		if err := mgr.Down(ctx, env); err != nil {
			return err
		}

		if err := state.Remove(dir); err != nil {
			return fmt.Errorf("removing state: %w", err)
		}

		fmt.Println("Environment stopped.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
