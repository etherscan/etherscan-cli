//go:build windows

package updater

import (
	"os/exec"
	"syscall"
)

func configureBackgroundCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
