package infrastructure

import (
	"path/filepath"
	"strings"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/resources/instancepath"
)

const sqliteFileName = "state.db"

// ResolveStatePath returns the instance-scoped SNS storage directory.
func ResolveStatePath(baseDir, instanceID string) (string, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		paths := runtime.ResolvePaths()
		baseDir = paths.BaseDir
	}
	return instancepath.Resolve(baseDir, instanceID, "sns")
}

func ResolveSQLitePath(baseDir, instanceID string) (string, error) {
	statePath, err := ResolveStatePath(baseDir, instanceID)
	if err != nil {
		return "", err
	}
	return filepath.Join(statePath, sqliteFileName), nil
}
