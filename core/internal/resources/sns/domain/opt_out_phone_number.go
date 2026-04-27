package domain

import (
	"fmt"
	"strings"
	"time"
)

// OptOutPhoneNumber models SNS opt-out status for one phone number.
type OptOutPhoneNumber struct {
	PhoneNumber string
	TenantKey   string
	IsOptedOut  bool
	UpdatedAt   time.Time
}

func NewOptOutPhoneNumber(tenant Tenant, phoneNumber string, isOptedOut bool, now time.Time) (OptOutPhoneNumber, error) {
	phoneNumber = normalizePhoneNumber(phoneNumber)
	if phoneNumber == "" {
		return OptOutPhoneNumber{}, fmt.Errorf("%w: phone number is required", ErrValidation)
	}
	if !e164PhonePattern.MatchString(phoneNumber) {
		return OptOutPhoneNumber{}, fmt.Errorf("%w: phone number must be in E.164 format", ErrValidation)
	}

	return OptOutPhoneNumber{
		PhoneNumber: phoneNumber,
		TenantKey:   strings.TrimSpace(tenant.Key()),
		IsOptedOut:  isOptedOut,
		UpdatedAt:   normalizeTimestamp(now),
	}, nil
}
