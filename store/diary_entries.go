package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var ErrDiaryEntryNotFound = errors.New("diary entry not found")

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

func (s *DiaryEntryStore) Create(ctx context.Context, userId uuid.UUID, content string) (*DiaryEntry, error) {
	const query = `
		INSERT INTO diary_entries (user_id, content)
		VALUES ($1, $2)
		RETURNING id, user_id, content, created_at`

	var entry DiaryEntry
	if err := s.db.GetContext(ctx, &entry, query, userId, content); err != nil {
		return nil, fmt.Errorf("failed to create diary entry: %w", err)
	}
	return &entry, nil
}

func (s *DiaryEntryStore) ListByUser(ctx context.Context, userId uuid.UUID, limit, offset int) ([]DiaryEntry, error) {
	if limit <= 0 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}
	if offset < 0 {
		return nil, fmt.Errorf("offset must be non-negative")
	}

	const query = `
		SELECT id, user_id, content, created_at
		FROM diary_entries
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3`

	entries := make([]DiaryEntry, 0)
	if err := s.db.SelectContext(ctx, &entries, query, userId, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list diary entries for user %s: %w", userId, err)
	}
	return entries, nil
}

func (s *DiaryEntryStore) ById(ctx context.Context, id, userId uuid.UUID) (*DiaryEntry, error) {
	const query = `
		SELECT id, user_id, content, created_at
		FROM diary_entries
		WHERE id = $1 AND user_id = $2`

	var entry DiaryEntry
	if err := s.db.GetContext(ctx, &entry, query, id, userId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id %s", ErrDiaryEntryNotFound, id)
		}
		return nil, fmt.Errorf("failed to get diary entry by id %s: %w", id, err)
	}
	return &entry, nil
}

func (s *DiaryEntryStore) Delete(ctx context.Context, id, userId uuid.UUID) error {
	const query = `
		DELETE FROM diary_entries
		WHERE id = $1 AND user_id = $2`

	result, err := s.db.ExecContext(ctx, query, id, userId)
	if err != nil {
		return fmt.Errorf("failed to delete diary entry with id %s: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for diary entry deletion with id %s: %w", id, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%w: id %s", ErrDiaryEntryNotFound, id)
	}

	return nil
}
