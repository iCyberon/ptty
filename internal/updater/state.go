package updater

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type stateFile struct {
	UpdatedTo string `json:"updated_to,omitempty"`
}

func configDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ptty"), nil
}

func statePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

func WriteUpdatedVersion(version string) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	path, err := statePath()
	if err != nil {
		return err
	}

	data, err := json.Marshal(stateFile{UpdatedTo: version})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func ReadAndClearUpdatedVersion() string {
	path, err := statePath()
	if err != nil {
		return ""
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var s stateFile
	if err := json.Unmarshal(data, &s); err != nil {
		return ""
	}

	if s.UpdatedTo != "" {
		os.Remove(path)
	}

	return s.UpdatedTo
}
