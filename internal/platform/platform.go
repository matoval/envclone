package platform

import "context"

type Platform interface {
	Name() string
	NerdctlArgs(args ...string) []string
	EnsureRuntime(ctx context.Context) error
	MountArgs(hostPath, containerPath string) []string
	SSHPort() int
	Cleanup(ctx context.Context) error
}
