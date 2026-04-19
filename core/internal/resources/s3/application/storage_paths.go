package application

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/michasdev/mildstack/core/internal/resources/instancepath"
)

type StorageConfig struct {
	BaseDir    string
	InstanceID string
}

func ResolveStoragePath(config StorageConfig) (string, error) {
	baseDir := strings.TrimSpace(config.BaseDir)
	if baseDir == "" {
		homeDir, _ := os.UserHomeDir()
		baseDir = resolveBaseDir(homeDir)
	}

	return instancepath.Resolve(baseDir, config.InstanceID, "s3")
}

func resolveBaseDir(homeDir string) string {
	if homeDir != "" {
		return filepath.Join(homeDir, ".mildstack")
	}
	return filepath.Join(".", ".mildstack")
}
