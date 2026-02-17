package cmd

import (
	"fmt"

	"github.com/matoval/envclone/internal/state"
	"github.com/spf13/cobra"
)

var sshConfigCmd = &cobra.Command{
	Use:   "ssh-config",
	Short: "Print SSH config for the dev environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := getProjectDir()
		if err != nil {
			return err
		}

		env, err := state.Load(dir)
		if err != nil {
			return fmt.Errorf("no environment found (run 'envclone up' first): %w", err)
		}

		fmt.Printf("Host envclone-%s\n", env.ProjectName)
		fmt.Printf("  HostName localhost\n")
		fmt.Printf("  Port %d\n", env.SSHPort)
		fmt.Printf("  User %s\n", env.RemoteUser)
		fmt.Printf("  StrictHostKeyChecking no\n")
		fmt.Printf("  UserKnownHostsFile /dev/null\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(sshConfigCmd)
}
