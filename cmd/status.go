package cmd

import (
	"fmt"
	"text/tabwriter"
	"os"

	"github.com/matoval/envclone/internal/container"
	"github.com/matoval/envclone/internal/exec"
	"github.com/matoval/envclone/internal/platform"
	"github.com/matoval/envclone/internal/state"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of the dev environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		dir, err := getProjectDir()
		if err != nil {
			return err
		}

		env, err := state.Load(dir)
		if err != nil {
			fmt.Println("No environment running.")
			return nil
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

		infos, err := mgr.Status(ctx, env)
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tROLE\tSTATUS")
		for _, info := range infos {
			fmt.Fprintf(w, "%s\t%s\t%s\n", info.Name, info.Role, info.Status)
		}
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
