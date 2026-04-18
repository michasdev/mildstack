package application

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
)

type StorageConfig struct {
	BaseDir    string
	InstanceID string
}

func ResolveStoragePath(config StorageConfig) (string, error) {
	instanceID := strings.TrimSpace(config.InstanceID)
	if instanceID == "" {
		return "", fmt.Errorf("dynamodb: instance id is required")
	}

	baseDir := strings.TrimSpace(config.BaseDir)
	if baseDir == "" {
		paths := runtime.ResolvePaths()
		baseDir = paths.BaseDir
	}

	return filepath.Join(baseDir, "instances", instanceID, "dynamodb"), nil
}
