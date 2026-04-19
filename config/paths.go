// Package config resolves configuration and data directories for OpenHarness.
//
// Follows XDG-like conventions with ~/.openharness/ as the default base directory.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultBaseDir  = ".openharness"
	configFileName  = "settings.json"
	envConfigDirKey = "OPENHARNESS_CONFIG_DIR"
	envDataDirKey   = "OPENHARNESS_DATA_DIR"
	envLogsDirKey   = "OPENHARNESS_LOGS_DIR"
)

func GetConfigDir() (string, error) {
	if env := os.Getenv(envConfigDirKey); env != "" {
		if err := os.MkdirAll(env, 0o755); err != nil {
			return "", fmt.Errorf("create config dir %q: %w", env, err)
		}
		return env, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	dir := filepath.Join(home, defaultBaseDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create config dir %q: %w", dir, err)
	}
	return dir, nil
}

func GetDataDir() (string, error) {
	if env := os.Getenv(envDataDirKey); env != "" {
		if err := os.MkdirAll(env, 0o755); err != nil {
			return "", fmt.Errorf("create data dir %q: %w", env, err)
		}
		return env, nil
	}

	cfg, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cfg, "data")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create data dir %q: %w", dir, err)
	}
	return dir, nil
}

func GetLogsDir() (string, error) {
	if env := os.Getenv(envLogsDirKey); env != "" {
		if err := os.MkdirAll(env, 0o755); err != nil {
			return "", fmt.Errorf("create logs dir %q: %w", env, err)
		}
		return env, nil
	}

	cfg, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cfg, "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create logs dir %q: %w", dir, err)
	}
	return dir, nil
}
