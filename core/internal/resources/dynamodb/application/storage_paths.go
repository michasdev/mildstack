package application

import (
	"strings"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/resources/instancepath"
)

type StorageConfig struct {
	BaseDir    string
	InstanceID string
}

func ResolveStoragePath(config StorageConfig) (string, error) {
	baseDir := strings.TrimSpace(config.BaseDir)
	if baseDir == "" {
		paths := runtime.ResolvePaths()
		baseDir = paths.BaseDir
	}

	return instancepath.Resolve(baseDir, config.InstanceID, "dynamodb")
}
