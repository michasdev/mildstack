package infrastructure

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

type PlatformRepository struct {
	store *SQLiteStore
}

func NewPlatformRepository(store *SQLiteStore) PlatformRepository {
	return PlatformRepository{store: store}
}

func (r PlatformRepository) CreateApplication(application domain.PlatformApplication) (domain.PlatformApplication, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.PlatformApplication{}, err
	}

	if existing, err := r.GetApplicationByName(application.TenantKey, application.Name); err == nil {
		return existing, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.PlatformApplication{}, err
	}

	attributesJSON, err := marshalStringMap(application.Attributes)
	if err != nil {
		return domain.PlatformApplication{}, err
	}
	tagsJSON, err := marshalStringMap(application.Tags)
	if err != nil {
		return domain.PlatformApplication{}, err
	}

	_, err = db.Exec(`
INSERT INTO platform_applications (
  platform_application_arn,
  tenant_key,
  name,
  platform,
  attributes_json,
  tags_json,
  created_at,
  updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`,
		application.ARN,
		application.TenantKey,
		application.Name,
		application.Platform,
		attributesJSON,
		tagsJSON,
		application.CreatedAt.UTC().Format(time.RFC3339Nano),
		application.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return domain.PlatformApplication{}, fmt.Errorf("sns: create platform application: %w", err)
	}
	return r.GetApplicationByARN(application.TenantKey, application.ARN)
}

func (r PlatformRepository) GetApplicationByName(tenantKey, name string) (domain.PlatformApplication, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.PlatformApplication{}, err
	}

	row := db.QueryRow(`
SELECT
  platform_application_arn,
  tenant_key,
  name,
  platform,
  attributes_json,
  tags_json,
  created_at,
  updated_at
FROM platform_applications
WHERE tenant_key = ? AND name = ?
ORDER BY created_at ASC
LIMIT 1
`, strings.TrimSpace(tenantKey), strings.TrimSpace(name))

	application, err := scanPlatformApplicationRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.PlatformApplication{}, domain.ErrNotFound
		}
		return domain.PlatformApplication{}, err
	}
	return application, nil
}

func (r PlatformRepository) GetApplicationByARN(tenantKey, platformApplicationARN string) (domain.PlatformApplication, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.PlatformApplication{}, err
	}

	row := db.QueryRow(`
SELECT
  platform_application_arn,
  tenant_key,
  name,
  platform,
  attributes_json,
  tags_json,
  created_at,
  updated_at
FROM platform_applications
WHERE tenant_key = ? AND platform_application_arn = ?
`, strings.TrimSpace(tenantKey), strings.TrimSpace(platformApplicationARN))

	application, err := scanPlatformApplicationRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.PlatformApplication{}, domain.ErrNotFound
		}
		return domain.PlatformApplication{}, err
	}
	return application, nil
}

func (r PlatformRepository) ListApplicationsByTenant(tenantKey, nextToken string, limit int) ([]domain.PlatformApplication, string, error) {
	db, err := r.ensureDB()
	if err != nil {
		return nil, "", err
	}

	limit = normalizeSNSPageLimit(limit)
	nextToken = strings.TrimSpace(nextToken)

	rows, err := db.Query(`
SELECT
  platform_application_arn,
  tenant_key,
  name,
  platform,
  attributes_json,
  tags_json,
  created_at,
  updated_at
FROM platform_applications
WHERE tenant_key = ? AND platform_application_arn > ?
ORDER BY platform_application_arn ASC
LIMIT ?
`, strings.TrimSpace(tenantKey), nextToken, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("sns: list platform applications: %w", err)
	}
	defer rows.Close()

	applications := make([]domain.PlatformApplication, 0, limit+1)
	for rows.Next() {
		application, err := scanPlatformApplicationRows(rows)
		if err != nil {
			return nil, "", err
		}
		applications = append(applications, application)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("sns: iterate platform applications: %w", err)
	}

	if len(applications) <= limit {
		return applications, "", nil
	}
	page := applications[:limit]
	return page, page[len(page)-1].ARN, nil
}

func (r PlatformRepository) UpdateApplication(application domain.PlatformApplication) error {
	db, err := r.ensureDB()
	if err != nil {
		return err
	}

	attributesJSON, err := marshalStringMap(application.Attributes)
	if err != nil {
		return err
	}
	tagsJSON, err := marshalStringMap(application.Tags)
	if err != nil {
		return err
	}

	result, err := db.Exec(`
UPDATE platform_applications
SET platform = ?, attributes_json = ?, tags_json = ?, updated_at = ?
WHERE tenant_key = ? AND platform_application_arn = ?
`,
		application.Platform,
		attributesJSON,
		tagsJSON,
		application.UpdatedAt.UTC().Format(time.RFC3339Nano),
		strings.TrimSpace(application.TenantKey),
		strings.TrimSpace(application.ARN),
	)
	if err != nil {
		return fmt.Errorf("sns: update platform application: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("sns: update platform application affected rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r PlatformRepository) DeleteApplicationByARN(tenantKey, platformApplicationARN string) error {
	db, err := r.ensureDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("sns: begin platform application delete: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`
DELETE FROM platform_endpoints
WHERE tenant_key = ? AND platform_application_arn = ?
`, strings.TrimSpace(tenantKey), strings.TrimSpace(platformApplicationARN)); err != nil {
		return fmt.Errorf("sns: cascade delete platform endpoints: %w", err)
	}

	result, err := tx.Exec(`
DELETE FROM platform_applications
WHERE tenant_key = ? AND platform_application_arn = ?
`, strings.TrimSpace(tenantKey), strings.TrimSpace(platformApplicationARN))
	if err != nil {
		return fmt.Errorf("sns: delete platform application: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("sns: delete platform application affected rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sns: commit platform application delete: %w", err)
	}
	return nil
}

func (r PlatformRepository) CreateEndpoint(endpoint domain.PlatformEndpoint) (domain.PlatformEndpoint, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.PlatformEndpoint{}, err
	}

	if existing, err := r.GetEndpointByToken(endpoint.TenantKey, endpoint.PlatformApplicationARN, endpoint.Token); err == nil {
		return existing, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.PlatformEndpoint{}, err
	}

	attributes := cloneStringMap(endpoint.Attributes)
	if attributes == nil {
		attributes = map[string]string{}
	}
	if strings.TrimSpace(endpoint.CustomUserData) != "" {
		attributes["CustomUserData"] = endpoint.CustomUserData
	}
	attributesJSON, err := marshalStringMap(attributes)
	if err != nil {
		return domain.PlatformEndpoint{}, err
	}
	tagsJSON, err := marshalStringMap(endpoint.Tags)
	if err != nil {
		return domain.PlatformEndpoint{}, err
	}

	enabled := 0
	if endpoint.Enabled {
		enabled = 1
	}

	_, err = db.Exec(`
INSERT INTO platform_endpoints (
  endpoint_arn,
  platform_application_arn,
  tenant_key,
  token,
  attributes_json,
  tags_json,
  enabled,
  created_at,
  updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		endpoint.ARN,
		endpoint.PlatformApplicationARN,
		endpoint.TenantKey,
		endpoint.Token,
		attributesJSON,
		tagsJSON,
		enabled,
		endpoint.CreatedAt.UTC().Format(time.RFC3339Nano),
		endpoint.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return domain.PlatformEndpoint{}, fmt.Errorf("sns: create platform endpoint: %w", err)
	}
	return r.GetEndpointByARN(endpoint.TenantKey, endpoint.ARN)
}

func (r PlatformRepository) GetEndpointByToken(tenantKey, platformApplicationARN, token string) (domain.PlatformEndpoint, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.PlatformEndpoint{}, err
	}

	row := db.QueryRow(`
SELECT
  endpoint_arn,
  platform_application_arn,
  tenant_key,
  token,
  attributes_json,
  tags_json,
  enabled,
  created_at,
  updated_at
FROM platform_endpoints
WHERE tenant_key = ? AND platform_application_arn = ? AND token = ?
ORDER BY created_at ASC
LIMIT 1
`, strings.TrimSpace(tenantKey), strings.TrimSpace(platformApplicationARN), strings.TrimSpace(token))

	endpoint, err := scanPlatformEndpointRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.PlatformEndpoint{}, domain.ErrNotFound
		}
		return domain.PlatformEndpoint{}, err
	}
	return endpoint, nil
}

func (r PlatformRepository) GetEndpointByARN(tenantKey, endpointARN string) (domain.PlatformEndpoint, error) {
	db, err := r.ensureDB()
	if err != nil {
		return domain.PlatformEndpoint{}, err
	}

	row := db.QueryRow(`
SELECT
  endpoint_arn,
  platform_application_arn,
  tenant_key,
  token,
  attributes_json,
  tags_json,
  enabled,
  created_at,
  updated_at
FROM platform_endpoints
WHERE tenant_key = ? AND endpoint_arn = ?
`, strings.TrimSpace(tenantKey), strings.TrimSpace(endpointARN))

	endpoint, err := scanPlatformEndpointRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.PlatformEndpoint{}, domain.ErrNotFound
		}
		return domain.PlatformEndpoint{}, err
	}
	return endpoint, nil
}

func (r PlatformRepository) ListEndpointsByApplication(tenantKey, platformApplicationARN, nextToken string, limit int) ([]domain.PlatformEndpoint, string, error) {
	db, err := r.ensureDB()
	if err != nil {
		return nil, "", err
	}

	limit = normalizeSNSPageLimit(limit)
	nextToken = strings.TrimSpace(nextToken)

	rows, err := db.Query(`
SELECT
  endpoint_arn,
  platform_application_arn,
  tenant_key,
  token,
  attributes_json,
  tags_json,
  enabled,
  created_at,
  updated_at
FROM platform_endpoints
WHERE tenant_key = ? AND platform_application_arn = ? AND endpoint_arn > ?
ORDER BY endpoint_arn ASC
LIMIT ?
`, strings.TrimSpace(tenantKey), strings.TrimSpace(platformApplicationARN), nextToken, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("sns: list platform endpoints: %w", err)
	}
	defer rows.Close()

	endpoints := make([]domain.PlatformEndpoint, 0, limit+1)
	for rows.Next() {
		endpoint, err := scanPlatformEndpointRows(rows)
		if err != nil {
			return nil, "", err
		}
		endpoints = append(endpoints, endpoint)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("sns: iterate platform endpoints: %w", err)
	}

	if len(endpoints) <= limit {
		return endpoints, "", nil
	}
	page := endpoints[:limit]
	return page, page[len(page)-1].ARN, nil
}

func (r PlatformRepository) UpdateEndpoint(endpoint domain.PlatformEndpoint) error {
	db, err := r.ensureDB()
	if err != nil {
		return err
	}

	attributes := cloneStringMap(endpoint.Attributes)
	if attributes == nil {
		attributes = map[string]string{}
	}
	if strings.TrimSpace(endpoint.CustomUserData) != "" {
		attributes["CustomUserData"] = endpoint.CustomUserData
	}
	attributesJSON, err := marshalStringMap(attributes)
	if err != nil {
		return err
	}
	tagsJSON, err := marshalStringMap(endpoint.Tags)
	if err != nil {
		return err
	}
	enabled := 0
	if endpoint.Enabled {
		enabled = 1
	}

	result, err := db.Exec(`
UPDATE platform_endpoints
SET token = ?, attributes_json = ?, tags_json = ?, enabled = ?, updated_at = ?
WHERE tenant_key = ? AND endpoint_arn = ?
`,
		endpoint.Token,
		attributesJSON,
		tagsJSON,
		enabled,
		endpoint.UpdatedAt.UTC().Format(time.RFC3339Nano),
		strings.TrimSpace(endpoint.TenantKey),
		strings.TrimSpace(endpoint.ARN),
	)
	if err != nil {
		return fmt.Errorf("sns: update platform endpoint: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("sns: update platform endpoint affected rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r PlatformRepository) DeleteEndpointByARN(tenantKey, endpointARN string) error {
	db, err := r.ensureDB()
	if err != nil {
		return err
	}

	result, err := db.Exec(`
DELETE FROM platform_endpoints
WHERE tenant_key = ? AND endpoint_arn = ?
`, strings.TrimSpace(tenantKey), strings.TrimSpace(endpointARN))
	if err != nil {
		return fmt.Errorf("sns: delete platform endpoint: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("sns: delete platform endpoint affected rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r PlatformRepository) ensureDB() (*sql.DB, error) {
	if r.store == nil || r.store.db == nil {
		return nil, fmt.Errorf("sns: platform repository not initialized")
	}
	return r.store.db, nil
}

func scanPlatformApplicationRow(row interface{ Scan(dest ...any) error }) (domain.PlatformApplication, error) {
	var (
		arn           string
		tenantKey     string
		name          string
		platform      string
		attributesRaw string
		tagsRaw       string
		createdAtRaw  string
		updatedAtRaw  string
	)
	if err := row.Scan(&arn, &tenantKey, &name, &platform, &attributesRaw, &tagsRaw, &createdAtRaw, &updatedAtRaw); err != nil {
		return domain.PlatformApplication{}, err
	}

	attributes, err := unmarshalStringMap(attributesRaw)
	if err != nil {
		return domain.PlatformApplication{}, err
	}
	tags, err := unmarshalStringMap(tagsRaw)
	if err != nil {
		return domain.PlatformApplication{}, err
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return domain.PlatformApplication{}, fmt.Errorf("sns: parse platform application created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return domain.PlatformApplication{}, fmt.Errorf("sns: parse platform application updated_at: %w", err)
	}

	return domain.PlatformApplication{
		ARN:        arn,
		TenantKey:  tenantKey,
		Name:       name,
		Platform:   platform,
		Attributes: attributes,
		Tags:       tags,
		CreatedAt:  createdAt.UTC(),
		UpdatedAt:  updatedAt.UTC(),
	}, nil
}

func scanPlatformApplicationRows(rows *sql.Rows) (domain.PlatformApplication, error) {
	return scanPlatformApplicationRow(rows)
}

func scanPlatformEndpointRow(row interface{ Scan(dest ...any) error }) (domain.PlatformEndpoint, error) {
	var (
		arn                    string
		platformApplicationARN string
		tenantKey              string
		token                  string
		attributesRaw          string
		tagsRaw                string
		enabledRaw             int
		createdAtRaw           string
		updatedAtRaw           string
	)
	if err := row.Scan(&arn, &platformApplicationARN, &tenantKey, &token, &attributesRaw, &tagsRaw, &enabledRaw, &createdAtRaw, &updatedAtRaw); err != nil {
		return domain.PlatformEndpoint{}, err
	}

	attributes, err := unmarshalStringMap(attributesRaw)
	if err != nil {
		return domain.PlatformEndpoint{}, err
	}
	tags, err := unmarshalStringMap(tagsRaw)
	if err != nil {
		return domain.PlatformEndpoint{}, err
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return domain.PlatformEndpoint{}, fmt.Errorf("sns: parse platform endpoint created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return domain.PlatformEndpoint{}, fmt.Errorf("sns: parse platform endpoint updated_at: %w", err)
	}

	customUserData := strings.TrimSpace(attributes["CustomUserData"])
	delete(attributes, "CustomUserData")

	return domain.PlatformEndpoint{
		ARN:                    arn,
		PlatformApplicationARN: platformApplicationARN,
		TenantKey:              tenantKey,
		Token:                  token,
		CustomUserData:         customUserData,
		Attributes:             attributes,
		Tags:                   tags,
		Enabled:                enabledRaw != 0,
		CreatedAt:              createdAt.UTC(),
		UpdatedAt:              updatedAt.UTC(),
	}, nil
}

func scanPlatformEndpointRows(rows *sql.Rows) (domain.PlatformEndpoint, error) {
	return scanPlatformEndpointRow(rows)
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}
