package instancepath

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Resolve joins a base directory with the canonical instance-scoped resource layout.
func Resolve(baseDir, instanceID, service string) (string, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return "", fmt.Errorf("instancepath: base dir is required")
	}

	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return "", fmt.Errorf("instancepath: instance id is required")
	}

	service = strings.TrimSpace(service)
	if service == "" {
		return "", fmt.Errorf("instancepath: service is required")
	}

	return filepath.Join(baseDir, "instances", instanceID, service), nil
}

// ResolveRoot joins a base directory with the canonical instance-scoped root.
func ResolveRoot(baseDir, instanceID string) (string, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return "", fmt.Errorf("instancepath: base dir is required")
	}

	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return "", fmt.Errorf("instancepath: instance id is required")
	}

	return filepath.Join(baseDir, "instances", instanceID), nil
}
