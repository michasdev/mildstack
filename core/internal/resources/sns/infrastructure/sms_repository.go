package infrastructure

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

type SMSRepository struct {
	store *SQLiteStore
}

func NewSMSRepository(store *SQLiteStore) SMSRepository {
	return SMSRepository{store: store}
}

func (r SMSRepository) SetSMSAttributes(tenantKey string, attributes map[string]string, now time.Time) error {
	db, err := r.ensureDB()
	if err != nil {
		return err
	}
	attributesJSON, err := marshalStringMap(attributes)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
INSERT INTO sms_attributes (tenant_key, attributes_json, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(tenant_key) DO UPDATE SET
  attributes_json = excluded.attributes_json,
  updated_at = excluded.updated_at
`, strings.TrimSpace(tenantKey), attributesJSON, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("sns: set sms attributes: %w", err)
	}
	return nil
}

func (r SMSRepository) GetSMSAttributes(tenantKey string) (map[string]string, error) {
	db, err := r.ensureDB()
	if err != nil {
		return nil, err
	}

	row := db.QueryRow(`
SELECT attributes_json
FROM sms_attributes
WHERE tenant_key = ?
`, strings.TrimSpace(tenantKey))

	var raw string
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("sns: get sms attributes: %w", err)
	}
	return unmarshalStringMap(raw)
}

func (r SMSRepository) UpsertOptOutPhone(entry domain.OptOutPhoneNumber) error {
	db, err := r.ensureDB()
	if err != nil {
		return err
	}

	isOptedOut := 0
	if entry.IsOptedOut {
		isOptedOut = 1
	}

	_, err = db.Exec(`
INSERT INTO opt_out_phone_numbers (phone_number, tenant_key, is_opted_out, updated_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(phone_number, tenant_key) DO UPDATE SET
  is_opted_out = excluded.is_opted_out,
  updated_at = excluded.updated_at
`,
		entry.PhoneNumber,
		entry.TenantKey,
		isOptedOut,
		entry.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("sns: upsert opt-out phone number: %w", err)
	}
	return nil
}

func (r SMSRepository) IsOptedOut(tenantKey, phoneNumber string) (bool, error) {
	db, err := r.ensureDB()
	if err != nil {
		return false, err
	}

	row := db.QueryRow(`
SELECT is_opted_out
FROM opt_out_phone_numbers
WHERE tenant_key = ? AND phone_number = ?
`, strings.TrimSpace(tenantKey), strings.TrimSpace(phoneNumber))

	var isOptedOut int
	if err := row.Scan(&isOptedOut); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("sns: check opt-out phone number: %w", err)
	}
	return isOptedOut != 0, nil
}

func (r SMSRepository) ListOptedOutPhoneNumbers(tenantKey, nextToken string, limit int) ([]string, string, error) {
	db, err := r.ensureDB()
	if err != nil {
		return nil, "", err
	}
	limit = normalizeSNSPageLimit(limit)
	nextToken = strings.TrimSpace(nextToken)

	rows, err := db.Query(`
SELECT phone_number
FROM opt_out_phone_numbers
WHERE tenant_key = ? AND is_opted_out = 1 AND phone_number > ?
ORDER BY phone_number ASC
LIMIT ?
`, strings.TrimSpace(tenantKey), nextToken, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("sns: list opted-out phone numbers: %w", err)
	}
	defer rows.Close()

	phoneNumbers := make([]string, 0, limit+1)
	for rows.Next() {
		var phoneNumber string
		if err := rows.Scan(&phoneNumber); err != nil {
			return nil, "", fmt.Errorf("sns: scan opted-out phone number: %w", err)
		}
		phoneNumbers = append(phoneNumbers, phoneNumber)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("sns: iterate opted-out phone numbers: %w", err)
	}

	if len(phoneNumbers) <= limit {
		return phoneNumbers, "", nil
	}
	page := phoneNumbers[:limit]
	return page, page[len(page)-1], nil
}

func (r SMSRepository) CreateSMSSandboxPhoneNumber(phone domain.SMSSandboxPhoneNumber) (domain.SMSSandboxPhoneNumber, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.SMSSandboxPhoneNumber{}, err
	}

	if existing, err := r.GetSMSSandboxPhoneNumber(phone.TenantKey, phone.PhoneNumber); err == nil {
		return existing, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.SMSSandboxPhoneNumber{}, err
	}

	expiresAt := ""
	if phone.OTPExpires != nil {
		expiresAt = phone.OTPExpires.UTC().Format(time.RFC3339Nano)
	}
	verifiedAt := ""
	if phone.VerifiedAt != nil {
		verifiedAt = phone.VerifiedAt.UTC().Format(time.RFC3339Nano)
	}

	_, err = db.Exec(`
INSERT INTO sms_sandbox_phone_numbers (
  phone_number,
  tenant_key,
  status,
  language_code,
  otp_code,
  otp_expires_at,
  created_at,
  verified_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`,
		phone.PhoneNumber,
		phone.TenantKey,
		phone.Status,
		phone.Language,
		phone.OTPCode,
		nullableString(expiresAt),
		phone.CreatedAt.UTC().Format(time.RFC3339Nano),
		nullableString(verifiedAt),
	)
	if err != nil {
		return domain.SMSSandboxPhoneNumber{}, fmt.Errorf("sns: create sms sandbox phone number: %w", err)
	}
	return r.GetSMSSandboxPhoneNumber(phone.TenantKey, phone.PhoneNumber)
}

func (r SMSRepository) GetSMSSandboxPhoneNumber(tenantKey, phoneNumber string) (domain.SMSSandboxPhoneNumber, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.SMSSandboxPhoneNumber{}, err
	}

	row := db.QueryRow(`
SELECT
  phone_number,
  tenant_key,
  status,
  COALESCE(language_code, ''),
  COALESCE(otp_code, ''),
  COALESCE(otp_expires_at, ''),
  created_at,
  COALESCE(verified_at, '')
FROM sms_sandbox_phone_numbers
WHERE tenant_key = ? AND phone_number = ?
`, strings.TrimSpace(tenantKey), strings.TrimSpace(phoneNumber))

	phone, err := scanSMSSandboxPhoneRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.SMSSandboxPhoneNumber{}, domain.ErrNotFound
		}
		return domain.SMSSandboxPhoneNumber{}, err
	}
	return phone, nil
}

func (r SMSRepository) UpdateSMSSandboxPhoneNumber(phone domain.SMSSandboxPhoneNumber) error {
	db, err := r.ensureDB()
	if err != nil {
		return err
	}

	expiresAt := ""
	if phone.OTPExpires != nil {
		expiresAt = phone.OTPExpires.UTC().Format(time.RFC3339Nano)
	}
	verifiedAt := ""
	if phone.VerifiedAt != nil {
		verifiedAt = phone.VerifiedAt.UTC().Format(time.RFC3339Nano)
	}

	result, err := db.Exec(`
UPDATE sms_sandbox_phone_numbers
SET status = ?, language_code = ?, otp_code = ?, otp_expires_at = ?, verified_at = ?
WHERE tenant_key = ? AND phone_number = ?
`,
		phone.Status,
		nullableString(phone.Language),
		nullableString(phone.OTPCode),
		nullableString(expiresAt),
		nullableString(verifiedAt),
		phone.TenantKey,
		phone.PhoneNumber,
	)
	if err != nil {
		return fmt.Errorf("sns: update sms sandbox phone number: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("sns: update sms sandbox phone number affected rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r SMSRepository) DeleteSMSSandboxPhoneNumber(tenantKey, phoneNumber string) error {
	db, err := r.ensureDB()
	if err != nil {
		return err
	}

	result, err := db.Exec(`
DELETE FROM sms_sandbox_phone_numbers
WHERE tenant_key = ? AND phone_number = ?
`, strings.TrimSpace(tenantKey), strings.TrimSpace(phoneNumber))
	if err != nil {
		return fmt.Errorf("sns: delete sms sandbox phone number: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("sns: delete sms sandbox phone number affected rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r SMSRepository) ListSMSSandboxPhoneNumbers(tenantKey, nextToken string, limit int) ([]domain.SMSSandboxPhoneNumber, string, error) {
	db, err := r.ensureDB()
	if err != nil {
		return nil, "", err
	}
	limit = normalizeSNSPageLimit(limit)
	nextToken = strings.TrimSpace(nextToken)

	rows, err := db.Query(`
SELECT
  phone_number,
  tenant_key,
  status,
  COALESCE(language_code, ''),
  COALESCE(otp_code, ''),
  COALESCE(otp_expires_at, ''),
  created_at,
  COALESCE(verified_at, '')
FROM sms_sandbox_phone_numbers
WHERE tenant_key = ? AND phone_number > ?
ORDER BY phone_number ASC
LIMIT ?
`, strings.TrimSpace(tenantKey), nextToken, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("sns: list sms sandbox phone numbers: %w", err)
	}
	defer rows.Close()

	phones := make([]domain.SMSSandboxPhoneNumber, 0, limit+1)
	for rows.Next() {
		phone, err := scanSMSSandboxPhoneRows(rows)
		if err != nil {
			return nil, "", err
		}
		phones = append(phones, phone)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("sns: iterate sms sandbox phone numbers: %w", err)
	}

	if len(phones) <= limit {
		return phones, "", nil
	}
	page := phones[:limit]
	return page, page[len(page)-1].PhoneNumber, nil
}

func (r SMSRepository) CountVerifiedSMSSandboxPhoneNumbers(tenantKey string) (int, error) {
	db, err := r.ensureDB()
	if err != nil {
		return 0, err
	}

	row := db.QueryRow(`
SELECT COUNT(*)
FROM sms_sandbox_phone_numbers
WHERE tenant_key = ? AND status = ?
`, strings.TrimSpace(tenantKey), domain.SMSSandboxStatusVerified)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("sns: count verified sms sandbox phone numbers: %w", err)
	}
	return count, nil
}

func (r SMSRepository) ListVerifiedSMSSandboxPhoneNumbers(tenantKey, nextToken string, limit int) ([]string, string, error) {
	db, err := r.ensureDB()
	if err != nil {
		return nil, "", err
	}
	limit = normalizeSNSPageLimit(limit)
	nextToken = strings.TrimSpace(nextToken)

	rows, err := db.Query(`
SELECT phone_number
FROM sms_sandbox_phone_numbers
WHERE tenant_key = ? AND status = ? AND phone_number > ?
ORDER BY phone_number ASC
LIMIT ?
`, strings.TrimSpace(tenantKey), domain.SMSSandboxStatusVerified, nextToken, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("sns: list verified sms sandbox phone numbers: %w", err)
	}
	defer rows.Close()

	phones := make([]string, 0, limit+1)
	for rows.Next() {
		var phoneNumber string
		if err := rows.Scan(&phoneNumber); err != nil {
			return nil, "", fmt.Errorf("sns: scan verified sms sandbox phone number: %w", err)
		}
		phones = append(phones, phoneNumber)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("sns: iterate verified sms sandbox phone numbers: %w", err)
	}

	if len(phones) <= limit {
		return phones, "", nil
	}
	page := phones[:limit]
	return page, page[len(page)-1], nil
}

func (r SMSRepository) ensureDB() (*sql.DB, error) {
	if r.store == nil || r.store.db == nil {
		return nil, fmt.Errorf("sns: sms repository not initialized")
	}
	return r.store.db, nil
}

func scanSMSSandboxPhoneRow(row interface{ Scan(dest ...any) error }) (domain.SMSSandboxPhoneNumber, error) {
	var (
		phoneNumber   string
		tenantKey     string
		status        string
		languageCode  string
		otpCode       string
		expiresAtRaw  string
		createdAtRaw  string
		verifiedAtRaw string
	)
	if err := row.Scan(&phoneNumber, &tenantKey, &status, &languageCode, &otpCode, &expiresAtRaw, &createdAtRaw, &verifiedAtRaw); err != nil {
		return domain.SMSSandboxPhoneNumber{}, err
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return domain.SMSSandboxPhoneNumber{}, fmt.Errorf("sns: parse sms sandbox created_at: %w", err)
	}

	var expiresAt *time.Time
	if trimmed := strings.TrimSpace(expiresAtRaw); trimmed != "" {
		parsed, err := time.Parse(time.RFC3339Nano, trimmed)
		if err != nil {
			return domain.SMSSandboxPhoneNumber{}, fmt.Errorf("sns: parse sms sandbox otp_expires_at: %w", err)
		}
		parsed = parsed.UTC()
		expiresAt = &parsed
	}

	var verifiedAt *time.Time
	if trimmed := strings.TrimSpace(verifiedAtRaw); trimmed != "" {
		parsed, err := time.Parse(time.RFC3339Nano, trimmed)
		if err != nil {
			return domain.SMSSandboxPhoneNumber{}, fmt.Errorf("sns: parse sms sandbox verified_at: %w", err)
		}
		parsed = parsed.UTC()
		verifiedAt = &parsed
	}

	return domain.SMSSandboxPhoneNumber{
		PhoneNumber: phoneNumber,
		TenantKey:   tenantKey,
		Status:      status,
		Language:    languageCode,
		OTPCode:     otpCode,
		OTPExpires:  expiresAt,
		CreatedAt:   createdAt.UTC(),
		VerifiedAt:  verifiedAt,
	}, nil
}

func scanSMSSandboxPhoneRows(rows *sql.Rows) (domain.SMSSandboxPhoneNumber, error) {
	return scanSMSSandboxPhoneRow(rows)
}
