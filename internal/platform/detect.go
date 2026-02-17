package platform

import (
	"fmt"
	"os/exec"
	"runtime"
)

func Detect() (Platform, error) {
	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("nerdctl"); err != nil {
			return nil, fmt.Errorf("nerdctl not found in PATH\nInstall: https://github.com/containerd/nerdctl#install")
		}
		return &Linux{}, nil
	case "darwin":
		if _, err := exec.LookPath("limactl"); err != nil {
			return nil, fmt.Errorf("lima not found in PATH\nInstall: brew install lima")
		}
		return &Darwin{}, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
