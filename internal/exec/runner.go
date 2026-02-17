package exec

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Runner struct {
	DryRun bool
}

func (r *Runner) Run(ctx context.Context, name string, args ...string) (string, error) {
	log.Printf("exec: %s %s", name, strings.Join(args, " "))

	if r.DryRun {
		return "", nil
	}

	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (r *Runner) RunInteractive(ctx context.Context, name string, args ...string) error {
	log.Printf("exec (interactive): %s %s", name, strings.Join(args, " "))

	if r.DryRun {
		return nil
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
