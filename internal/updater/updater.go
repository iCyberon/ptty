package updater

import (
	"context"
	"fmt"
	"os"
	"strings"

	selfupdate "github.com/creativeprojects/go-selfupdate"
)

const repo = "iCyberon/ptty"

type CheckResult struct {
	Version string
	Release *selfupdate.Release
}

type Updater struct {
	currentVersion string
	updater        *selfupdate.Updater
}

func New(currentVersion string) (*Updater, error) {
	up, err := selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create updater: %w", err)
	}
	return &Updater{
		currentVersion: currentVersion,
		updater:        up,
	}, nil
}

func (u *Updater) CheckLatest(ctx context.Context) (*CheckResult, error) {
	release, found, err := u.updater.DetectLatest(ctx, selfupdate.ParseSlug(repo))
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	current := strings.TrimPrefix(u.currentVersion, "v")
	if !release.GreaterThan(current) {
		return nil, nil
	}

	return &CheckResult{
		Version: release.Version(),
		Release: release,
	}, nil
}

func (u *Updater) Apply(ctx context.Context, release *selfupdate.Release) error {
	if release == nil {
		return fmt.Errorf("no release to apply")
	}

	cmdPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find executable path: %w", err)
	}

	return u.updater.UpdateTo(ctx, release, cmdPath)
}

func CanWrite() (string, bool) {
	path, err := os.Executable()
	if err != nil {
		return "", false
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return path, false
	}
	f.Close()
	return path, true
}
