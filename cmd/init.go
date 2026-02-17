package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed init_template.json
var defaultTemplate []byte

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new devcontainer configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := getProjectDir()
		if err != nil {
			return err
		}

		destDir := filepath.Join(dir, ".devcontainer")
		destFile := filepath.Join(destDir, "devcontainer.json")

		if _, err := os.Stat(destFile); err == nil {
			return fmt.Errorf("%s already exists", destFile)
		}

		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return fmt.Errorf("creating .devcontainer directory: %w", err)
		}

		if err := os.WriteFile(destFile, defaultTemplate, 0o644); err != nil {
			return fmt.Errorf("writing devcontainer.json: %w", err)
		}

		fmt.Printf("Created %s\n", destFile)
		fmt.Println("Edit the file to configure your dev environment, then run: envclone up")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
