package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var platformApplicationNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// PlatformApplication models a mobile push application namespace in SNS.
type PlatformApplication struct {
	ARN        string
	TenantKey  string
	Name       string
	Platform   string
	Attributes map[string]string
	Tags       map[string]string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewPlatformApplication(tenant Tenant, name, platform string, attributes, tags map[string]string, now time.Time) (PlatformApplication, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return PlatformApplication{}, fmt.Errorf("%w: platform application name is required", ErrValidation)
	}
	if len(name) > 256 || !platformApplicationNamePattern.MatchString(name) {
		return PlatformApplication{}, fmt.Errorf("%w: invalid platform application name", ErrValidation)
	}

	platform = strings.ToUpper(strings.TrimSpace(platform))
	if platform == "" {
		return PlatformApplication{}, fmt.Errorf("%w: platform is required", ErrValidation)
	}

	now = normalizeTimestamp(now)
	return PlatformApplication{
		ARN:        tenant.PlatformApplicationARN(platform, name),
		TenantKey:  tenant.Key(),
		Name:       name,
		Platform:   platform,
		Attributes: cloneStringMap(attributes),
		Tags:       cloneStringMap(tags),
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (a PlatformApplication) WithAttributes(values map[string]string, now time.Time) PlatformApplication {
	updated := a
	updated.Attributes = cloneStringMap(a.Attributes)
	if updated.Attributes == nil {
		updated.Attributes = map[string]string{}
	}
	for key, value := range values {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		updated.Attributes[trimmedKey] = strings.TrimSpace(value)
	}
	updated.UpdatedAt = normalizeTimestamp(now)
	return updated
}

func (a PlatformApplication) AttributesView() map[string]string {
	view := cloneStringMap(a.Attributes)
	if view == nil {
		view = map[string]string{}
	}
	view["PlatformApplicationArn"] = a.ARN
	return view
}
