package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var projectDir string

var rootCmd = &cobra.Command{
	Use:   "envclone",
	Short: "Containerized dev environments with sidecar services",
	Long:  `envclone provides containerized dev shells with sidecar services, host filesystem mounts, VS Code SSH integration, and devcontainer.json compatibility.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&projectDir, "project-dir", "", "project directory (defaults to current directory)")
}

func getProjectDir() (string, error) {
	if projectDir != "" {
		return projectDir, nil
	}
	return os.Getwd()
}
