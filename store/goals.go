package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var ErrGoalNotFound = errors.New("goal not found")

type GoalStore struct {
	db *sqlx.DB
}

func NewGoalStore(db *sql.DB) *GoalStore {
	return &GoalStore{
		db: sqlx.NewDb(db, "postgres"),
	}
}

type Goal struct {
	ID                   uuid.UUID `db:"id" json:"id"`
	UserID               uuid.UUID `db:"user_id" json:"user_id"`
	Name                 string    `db:"name" json:"name"`
	TargetAmountKopecks  int64     `db:"target_amount_kopecks" json:"target_amount_kopecks"`
	CurrentAmountKopecks int64     `db:"current_amount_kopecks" json:"current_amount_kopecks"`
	Currency             string    `db:"currency" json:"currency"`
	Status               string    `db:"status" json:"status"`
	CreatedAt            time.Time `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time `db:"updated_at" json:"updated_at"`
}

func (s *GoalStore) Create(ctx context.Context, userID uuid.UUID, name string, targetAmountKopecks int64) (*Goal, error) {
	const query = `
		INSERT INTO goals (user_id, name, target_amount_kopecks)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, name, target_amount_kopecks, current_amount_kopecks,
		          currency, status, created_at, updated_at`

	var goal Goal
	if err := s.db.GetContext(ctx, &goal, query, userID, strings.TrimSpace(name), targetAmountKopecks); err != nil {
		return nil, fmt.Errorf("failed to create goal: %w", err)
	}
	return &goal, nil
}

func (s *GoalStore) ListByUser(ctx context.Context, userID uuid.UUID) ([]Goal, error) {
	const query = `
		SELECT id, user_id, name, target_amount_kopecks, current_amount_kopecks,
		       currency, status, created_at, updated_at
		FROM goals
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC`

	goals := make([]Goal, 0)
	if err := s.db.SelectContext(ctx, &goals, query, userID); err != nil {
		return nil, fmt.Errorf("failed to list goals for user %s: %w", userID, err)
	}
	return goals, nil
}

func (s *GoalStore) ByID(ctx context.Context, id, userID uuid.UUID) (*Goal, error) {
	const query = `
		SELECT id, user_id, name, target_amount_kopecks, current_amount_kopecks,
		       currency, status, created_at, updated_at
		FROM goals
		WHERE id = $1 AND user_id = $2`

	var goal Goal
	if err := s.db.GetContext(ctx, &goal, query, id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id %s", ErrGoalNotFound, id)
		}
		return nil, fmt.Errorf("failed to get goal by id %s: %w", id, err)
	}
	return &goal, nil
}
