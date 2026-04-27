package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PlatformEndpoint models a concrete push target under a platform application.
type PlatformEndpoint struct {
	ARN                    string
	PlatformApplicationARN string
	TenantKey              string
	Token                  string
	CustomUserData         string
	Attributes             map[string]string
	Tags                   map[string]string
	Enabled                bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func NewPlatformEndpoint(tenant Tenant, applicationARN, platform, applicationName, token, customUserData string, attributes, tags map[string]string, now time.Time) (PlatformEndpoint, error) {
	applicationARN = strings.TrimSpace(applicationARN)
	if applicationARN == "" {
		return PlatformEndpoint{}, fmt.Errorf("%w: platform application arn is required", ErrValidation)
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return PlatformEndpoint{}, fmt.Errorf("%w: token is required", ErrValidation)
	}

	endpointARN := tenant.PlatformEndpointARN(platform, applicationName, uuid.NewString())
	if endpointARN == "" {
		return PlatformEndpoint{}, fmt.Errorf("%w: endpoint arn could not be generated", ErrValidation)
	}

	now = normalizeTimestamp(now)
	return PlatformEndpoint{
		ARN:                    endpointARN,
		PlatformApplicationARN: applicationARN,
		TenantKey:              tenant.Key(),
		Token:                  token,
		CustomUserData:         strings.TrimSpace(customUserData),
		Attributes:             cloneStringMap(attributes),
		Tags:                   cloneStringMap(tags),
		Enabled:                true,
		CreatedAt:              now,
		UpdatedAt:              now,
	}, nil
}

func (e PlatformEndpoint) WithAttributes(values map[string]string, now time.Time) PlatformEndpoint {
	updated := e
	updated.Attributes = cloneStringMap(e.Attributes)
	if updated.Attributes == nil {
		updated.Attributes = map[string]string{}
	}
	for key, value := range values {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		trimmedValue := strings.TrimSpace(value)
		switch trimmedKey {
		case "Token":
			if trimmedValue != "" {
				updated.Token = trimmedValue
			}
		case "CustomUserData":
			updated.CustomUserData = trimmedValue
		case "Enabled":
			updated.Enabled = parseTruthyString(trimmedValue)
		default:
			updated.Attributes[trimmedKey] = trimmedValue
		}
	}
	updated.UpdatedAt = normalizeTimestamp(now)
	return updated
}

func (e PlatformEndpoint) AttributesView() map[string]string {
	view := cloneStringMap(e.Attributes)
	if view == nil {
		view = map[string]string{}
	}
	view["Token"] = e.Token
	view["Enabled"] = boolString(e.Enabled)
	if strings.TrimSpace(e.CustomUserData) != "" {
		view["CustomUserData"] = e.CustomUserData
	}
	return view
}
