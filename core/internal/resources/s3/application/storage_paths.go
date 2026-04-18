package application

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type StorageConfig struct {
	BaseDir    string
	InstanceID string
}

func ResolveStoragePath(config StorageConfig) (string, error) {
	instanceID := strings.TrimSpace(config.InstanceID)
	if instanceID == "" {
		return "", fmt.Errorf("s3: instance id is required")
	}

	baseDir := strings.TrimSpace(config.BaseDir)
	if baseDir == "" {
		homeDir, _ := os.UserHomeDir()
		baseDir = resolveBaseDir(homeDir)
	}

	return filepath.Join(baseDir, "instances", instanceID, "s3"), nil
}

func resolveBaseDir(homeDir string) string {
	if homeDir != "" {
		return filepath.Join(homeDir, ".mildstack")
	}
	return filepath.Join(".", ".mildstack")
}
