package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	SMSSandboxStatusPending  = "Pending"
	SMSSandboxStatusVerified = "Verified"
)

var e164PhonePattern = regexp.MustCompile(`^\+[1-9][0-9]{1,14}$`)

// SMSSandboxPhoneNumber models SNS sandbox state for a phone number.
type SMSSandboxPhoneNumber struct {
	PhoneNumber string
	TenantKey   string
	Status      string
	Language    string
	OTPCode     string
	OTPExpires  *time.Time
	CreatedAt   time.Time
	VerifiedAt  *time.Time
}

func NewSMSSandboxPhoneNumber(tenant Tenant, phoneNumber, languageCode string, now time.Time) (SMSSandboxPhoneNumber, error) {
	phoneNumber = normalizePhoneNumber(phoneNumber)
	if phoneNumber == "" {
		return SMSSandboxPhoneNumber{}, fmt.Errorf("%w: phone number is required", ErrValidation)
	}
	if !e164PhonePattern.MatchString(phoneNumber) {
		return SMSSandboxPhoneNumber{}, fmt.Errorf("%w: phone number must be in E.164 format", ErrValidation)
	}

	now = normalizeTimestamp(now)
	expiresAt := now.Add(10 * time.Minute)
	return SMSSandboxPhoneNumber{
		PhoneNumber: phoneNumber,
		TenantKey:   tenant.Key(),
		Status:      SMSSandboxStatusPending,
		Language:    strings.TrimSpace(languageCode),
		OTPCode:     "123456",
		OTPExpires:  &expiresAt,
		CreatedAt:   now,
	}, nil
}

func (p SMSSandboxPhoneNumber) Verify(oneTimePassword string, now time.Time) (SMSSandboxPhoneNumber, error) {
	if p.Status == SMSSandboxStatusVerified {
		return p, nil
	}
	if strings.TrimSpace(oneTimePassword) == "" {
		return SMSSandboxPhoneNumber{}, fmt.Errorf("%w: one-time password is required", ErrValidation)
	}
	if strings.TrimSpace(p.OTPCode) == "" || strings.TrimSpace(p.OTPCode) != strings.TrimSpace(oneTimePassword) {
		return SMSSandboxPhoneNumber{}, fmt.Errorf("%w: invalid one-time password", ErrValidation)
	}
	now = normalizeTimestamp(now)
	if p.OTPExpires != nil && now.After(p.OTPExpires.UTC()) {
		return SMSSandboxPhoneNumber{}, fmt.Errorf("%w: one-time password expired", ErrValidation)
	}

	updated := p
	updated.Status = SMSSandboxStatusVerified
	updated.OTPCode = ""
	updated.OTPExpires = nil
	updated.VerifiedAt = &now
	return updated, nil
}

func normalizePhoneNumber(phoneNumber string) string {
	return strings.TrimSpace(phoneNumber)
}
