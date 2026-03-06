package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dbehnke/trindex/internal/db"
	"github.com/google/uuid"
)

// APIKey represents a stored API key without its raw secret string.
type APIKey struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	IsRevoked  bool       `json:"is_revoked"`
}

// Service handles authentication and audit logging.
type Service struct {
	db *db.DB
}

// NewService creates a new auth service tied to the database.
func NewService(database *db.DB) *Service {
	return &Service{
		db: database,
	}
}

// hashKey generates a SHA-256 hash of the raw string to store securely.
func hashKey(raw string) string {
	h := sha256.New()
	h.Write([]byte(raw))
	return hex.EncodeToString(h.Sum(nil))
}

// generateSecureRandomString creates a cryptographic random string prefixed with trindex_.
func generateSecureRandomString() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("trindex_%s", hex.EncodeToString(b)), nil
}

// CreateKey generates a new API Key, hashes it, stores the record,
// and returns the raw secret string ONCE.
func (s *Service) CreateKey(ctx context.Context, name string) (APIKey, string, error) {
	rawSecret, err := generateSecureRandomString()
	if err != nil {
		return APIKey{}, "", fmt.Errorf("failed to generate secure key: %w", err)
	}

	hash := hashKey(rawSecret)

	var key APIKey
	query := `
		INSERT INTO api_keys (name, key_hash) 
		VALUES ($1, $2) 
		RETURNING id, name, created_at, last_used_at, is_revoked`

	err = s.db.Pool().QueryRow(ctx, query, name, hash).Scan(
		&key.ID, &key.Name, &key.CreatedAt, &key.LastUsedAt, &key.IsRevoked,
	)
	if err != nil {
		return APIKey{}, "", fmt.Errorf("failed to insert API key: %w", err)
	}

	return key, rawSecret, nil
}

// ValidateKey checks if the raw secret maps to a valid, unrevoked hash.
// If successful, it updates the last_used_at timestamp and returns the key UUID.
func (s *Service) ValidateKey(ctx context.Context, rawSecret string) (*uuid.UUID, bool, error) {
	hash := hashKey(rawSecret)

	var id uuid.UUID
	var isRevoked bool

	query := `SELECT id, is_revoked FROM api_keys WHERE key_hash = $1`
	err := s.db.Pool().QueryRow(ctx, query, hash).Scan(&id, &isRevoked)
	if err != nil {
		// pgx.ErrNoRows or query err usually means invalid key
		return nil, false, nil
	}

	if isRevoked {
		return nil, false, nil
	}

	// Update last_used_at asynchronously
	go func(keyID uuid.UUID) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = s.db.Pool().Exec(bgCtx, `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, keyID)
	}(id)

	return &id, true, nil
}

// ListKeys retrieves all known keys excluding hashes for management UI.
func (s *Service) ListKeys(ctx context.Context) ([]APIKey, error) {
	query := `SELECT id, name, created_at, last_used_at, is_revoked FROM api_keys ORDER BY created_at DESC`
	rows, err := s.db.Pool().Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.CreatedAt, &k.LastUsedAt, &k.IsRevoked); err != nil {
			return nil, fmt.Errorf("failed to scan key: %w", err)
		}
		keys = append(keys, k)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}

// RevokeKey updates an API key to block future usage.
func (s *Service) RevokeKey(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_keys SET is_revoked = TRUE WHERE id = $1`
	tag, err := s.db.Pool().Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("API key not found")
	}
	return nil
}
