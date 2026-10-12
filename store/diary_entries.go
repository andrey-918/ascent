package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DiaryEntryStore struct {
	db *sqlx.DB
}

func NewDiaryEntryStore(db *sql.DB) *DiaryEntryStore {
	return &DiaryEntryStore{
		db: sqlx.NewDb(db, "postgres"),
	}
}

type DiaryEntry struct {
	ID        uuid.UUID `db:"id" json:"id"`
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	Content   string    `db:"content" json:"content"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (s *DiaryEntryStore) Create(ctx context.Context, userID uuid.UUID, content string) (*DiaryEntry, error) {
	const query = `
		INSERT INTO diary_entries (user_id, content)
		VALUES ($1, $2)
		RETURNING id, user_id, content, created_at`

	var entry DiaryEntry
	if err := s.db.GetContext(ctx, &entry, query, userID, content); err != nil {
		return nil, fmt.Errorf("failed to create diary entry: %w", err)
	}
	return &entry, nil
}

func (s *DiaryEntryStore) ListByUser(ctx context.Context, userID uuid.UUID) ([]DiaryEntry, error) {
	const query = `
		SELECT id, user_id, content, created_at
		FROM diary_entries
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC`

	entries := make([]DiaryEntry, 0)
	if err := s.db.SelectContext(ctx, &entries, query, userID); err != nil {
		return nil, fmt.Errorf("failed to list diary entries for user %s: %w", userID, err)
	}
	return entries, nil
}
