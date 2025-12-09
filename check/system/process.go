package system

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/alarmistdev/status/check"
)

// CheckProcessStatus creates a health check for process status by name.
func CheckProcessStatus(processName string) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		// Different commands for different OS
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "linux", "darwin":
			cmd = exec.CommandContext(ctx, "pgrep", "-f", processName)
		case "windows":
			cmd = exec.CommandContext(ctx, "tasklist", "/FI", "IMAGENAME eq "+processName)
		default:
			return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
		}

		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("process '%s' is not running: %w", processName, err)
		}

		if len(strings.TrimSpace(string(output))) == 0 {
			return fmt.Errorf("process '%s' not found", processName)
		}

		return nil
	})
}
