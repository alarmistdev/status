package system

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/alarmistdev/status/check"
)

// CheckMemory creates a health check for memory usage
func CheckMemory(maxUsagePercent float64) check.Check {
	return check.CheckFunc(func(_ context.Context) error {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		usagePercent := float64(m.Alloc) / float64(m.Sys) * 100
		if usagePercent > maxUsagePercent {
			return fmt.Errorf("high memory usage: %.2f%% (maximum: %.2f%%)",
				usagePercent, maxUsagePercent)
		}

		return nil
	})
}

// CheckDiskSpace creates a health check for disk space
func CheckDiskSpace(path string, minFreeSpaceGB float64) check.Check {
	return check.CheckFunc(func(_ context.Context) error {
		var stat syscall.Statfs_t
		if err := syscall.Statfs(path, &stat); err != nil {
			return fmt.Errorf("failed to get disk stats: %w", err)
		}

		freeSpaceGB := float64(stat.Bavail*uint64(stat.Bsize)) / (1024 * 1024 * 1024)
		if freeSpaceGB < minFreeSpaceGB {
			return fmt.Errorf("insufficient disk space: %.2f GB free (minimum: %.2f GB)",
				freeSpaceGB, minFreeSpaceGB)
		}

		return nil
	})
}

// FileCheck creates a health check for file existence and permissions
func CheckFile(path string, requiredPerm os.FileMode) check.Check {
	return check.CheckFunc(func(_ context.Context) error {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", path, err)
		}

		if info.Mode()&requiredPerm != requiredPerm {
			return fmt.Errorf("file %s has insufficient permissions: got %v, want %v",
				path, info.Mode(), requiredPerm)
		}

		return nil
	})
}
