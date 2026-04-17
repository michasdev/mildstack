package runtime

import (
	"os"
	"path/filepath"
)

type Paths struct {
	BaseDir      string
	ConfigDir    string
	InstancesDir string
	LogsDir      string
	CacheDir     string
}

func ResolvePaths() Paths {
	homeDir, _ := os.UserHomeDir()
	configDir, _ := os.UserConfigDir()
	return ResolvePathsFrom(homeDir, configDir)
}

func ResolvePathsFrom(homeDir, configDir string) Paths {
	baseDir := resolveBaseDir(homeDir, configDir)
	return Paths{
		BaseDir:      baseDir,
		ConfigDir:    filepath.Join(baseDir, "config"),
		InstancesDir: filepath.Join(baseDir, "instances"),
		LogsDir:      filepath.Join(baseDir, "logs"),
		CacheDir:     filepath.Join(baseDir, "cache"),
	}
}

func LegacyBaseDirFrom(homeDir, configDir string) string {
	if configDir != "" {
		return filepath.Join(configDir, "mildstack")
	}
	if homeDir != "" {
		return filepath.Join(homeDir, ".config", "mildstack")
	}
	return ""
}

func resolveBaseDir(homeDir, configDir string) string {
	switch {
	case homeDir != "":
		return filepath.Join(homeDir, ".mildstack")
	case configDir != "":
		return filepath.Join(configDir, ".mildstack")
	default:
		return filepath.Join(".", ".mildstack")
	}
}
