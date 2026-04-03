//go:build !windows

package updater

import (
	"os"
	"syscall"
)

func Restart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return syscall.Exec(exe, os.Args, os.Environ())
}
