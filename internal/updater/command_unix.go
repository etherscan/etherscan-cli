//go:build !windows

package updater

import "os/exec"

func configureBackgroundCommand(cmd *exec.Cmd) {}
