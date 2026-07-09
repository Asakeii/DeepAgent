package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrArtifactShareForbidden = errors.New("artifact not found or forbidden")

type ArtifactShareRecord struct {
	TokenHash  string
	Token      string
	ArtifactID int64
	UserID     string
	CreatedAt  sql.NullTime
	ExpiresAt  sql.NullTime
	RevokedAt  sql.NullTime
}

func CreateArtifactShare(ctx context.Context, db *sql.DB, artifactID int64, userID string, expiresAt sql.NullTime) (ArtifactShareRecord, error) {
	if db == nil {
		return ArtifactShareRecord{}, fmt.Errorf("db is nil")
	}
	if artifactID <= 0 {
		return ArtifactShareRecord{}, fmt.Errorf("artifact id is required")
	}
	userID = NormalizeUserID(userID)
	for attempt := 0; attempt < 3; attempt++ {
		token, err := NewArtifactShareToken()
		if err != nil {
			return ArtifactShareRecord{}, err
		}
		record := ArtifactShareRecord{
			TokenHash:  ArtifactShareTokenHash(token),
			Token:      token,
			ArtifactID: artifactID,
			UserID:     userID,
			CreatedAt:  sql.NullTime{Time: time.Now(), Valid: true},
			ExpiresAt:  expiresAt,
		}
		res, err := db.ExecContext(ctx,
			`INSERT INTO artifact_shares (token_hash, artifact_id, user_id, expires_at)
			 SELECT ?, id, user_id, ?
			 FROM artifacts
			 WHERE id=? AND user_id=?`,
			record.TokenHash, nullableTime(expiresAt), artifactID, userID,
		)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				continue
			}
			return ArtifactShareRecord{}, fmt.Errorf("create artifact share: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return ArtifactShareRecord{}, fmt.Errorf("create artifact share rows affected: %w", err)
		}
		if n == 0 {
			return ArtifactShareRecord{}, ErrArtifactShareForbidden
		}
		return record, nil
	}
	return ArtifactShareRecord{}, fmt.Errorf("create artifact share token collision")
}

func RevokeArtifactShare(ctx context.Context, db *sql.DB, token, userID string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("db is nil")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return false, fmt.Errorf("token is required")
	}
	userID = NormalizeUserID(userID)
	res, err := db.ExecContext(ctx,
		`UPDATE artifact_shares
		 SET revoked_at=CURRENT_TIMESTAMP
		 WHERE token_hash=? AND user_id=? AND revoked_at IS NULL`,
		ArtifactShareTokenHash(token), userID,
	)
	if err != nil {
		return false, fmt.Errorf("revoke artifact share: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("revoke artifact share rows affected: %w", err)
	}
	return n > 0, nil
}

func GetSharedArtifact(ctx context.Context, db *sql.DB, token string) (ArtifactRecord, ArtifactShareRecord, bool, error) {
	if db == nil {
		return ArtifactRecord{}, ArtifactShareRecord{}, false, fmt.Errorf("db is nil")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return ArtifactRecord{}, ArtifactShareRecord{}, false, nil
	}
	var artifact ArtifactRecord
	var share ArtifactShareRecord
	share.TokenHash = ArtifactShareTokenHash(token)
	err := db.QueryRowContext(ctx,
		`SELECT s.token_hash, s.artifact_id, s.user_id, s.created_at, s.expires_at, s.revoked_at,
		 a.id, a.user_id, a.thread_id, a.run_id, a.kind, a.title, a.format, a.content,
		 COALESCE(a.metadata, JSON_OBJECT()), a.version, a.source, a.created_at, a.updated_at
		 FROM artifact_shares s
		 JOIN artifacts a ON a.id=s.artifact_id
		 WHERE s.token_hash=?
		 AND s.revoked_at IS NULL
		 AND (s.expires_at IS NULL OR s.expires_at>CURRENT_TIMESTAMP)`,
		share.TokenHash,
	).Scan(
		&share.TokenHash, &share.ArtifactID, &share.UserID, &share.CreatedAt, &share.ExpiresAt, &share.RevokedAt,
		&artifact.ID, &artifact.UserID, &artifact.ThreadID, &artifact.RunID, &artifact.Kind, &artifact.Title, &artifact.Format, &artifact.Content,
		&artifact.Metadata, &artifact.Version, &artifact.Source, &artifact.CreatedAt, &artifact.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return ArtifactRecord{}, ArtifactShareRecord{}, false, nil
	}
	if err != nil {
		return ArtifactRecord{}, ArtifactShareRecord{}, false, fmt.Errorf("get shared artifact: %w", err)
	}
	return artifact, share, true, nil
}

func NewArtifactShareToken() (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate artifact share token: %w", err)
	}
	return "as_" + hex.EncodeToString(b[:]), nil
}

func ArtifactShareTokenHash(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func nullableTime(value sql.NullTime) any {
	if !value.Valid {
		return nil
	}
	return value.Time
}
