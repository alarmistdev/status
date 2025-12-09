package system

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/alarmistdev/status/check"
)

// CheckProcessStatus creates a health check for process status by name
func CheckProcessStatus(processName string) check.Check {
	return check.CheckFunc(func(_ context.Context) error {
		// Different commands for different OS
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "linux", "darwin":
			cmd = exec.Command("pgrep", "-f", processName)
		case "windows":
			cmd = exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", processName))
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
