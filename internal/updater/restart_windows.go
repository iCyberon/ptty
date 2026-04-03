//go:build windows

package updater

import (
	"os"
)

func Restart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	proc, err := os.StartProcess(exe, os.Args, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		return err
	}
	_ = proc.Release()
	os.Exit(0)
	return nil
}
