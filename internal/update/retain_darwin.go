package update

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/yetone/magpie/internal/proc"
)

func bundleOwnerExited(pid int) bool {
	// EPERM is still a live process. PID reuse defers cleanup, safely.
	return errors.Is(syscall.Kill(pid, 0), syscall.ESRCH)
}

func bundleUnused(bundle string) bool {
	exe := filepath.Join(bundle, "Contents", "MacOS", "magpie")
	if _, err := os.Stat(exe); err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// The CLI and GUI can run from the same image. Checking just the
	// updater's PID would remove the CLI's image while it is still running.
	// No -t: it implies -w, hiding warnings we must treat as unknown.
	out, err := proc.CommandContext(ctx, "/usr/sbin/lsof", "-F", "p", "--", exe).CombinedOutput()
	var exit *exec.ExitError
	return ctx.Err() == nil && errors.As(err, &exit) && exit.ExitCode() == 1 && len(out) == 0
}
