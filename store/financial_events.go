package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrInsufficientGoalProgress = errors.New("withdrawal exceeds current goal progress")
	ErrFinancialEventConflict   = errors.New("financial event id already used with different data")
)

type FinancialEventType string

const (
	FinancialEventDeposit    FinancialEventType = "deposit"
	FinancialEventWithdrawal FinancialEventType = "withdrawal"
)

type FinancialEvent struct {
	ID            uuid.UUID          `db:"id" json:"id"`
	UserID        uuid.UUID          `db:"user_id" json:"user_id"`
	GoalID        uuid.UUID          `db:"goal_id" json:"goal_id"`
	DiaryEntryID  uuid.UUID          `db:"diary_entry_id" json:"diary_entry_id"`
	Type          FinancialEventType `db:"event_type" json:"type"`
	AmountKopecks int64              `db:"amount_kopecks" json:"amount_kopecks"`
	Currency      string             `db:"currency" json:"currency"`
	SourceText    string             `db:"source_text" json:"source_text"`
	CreatedAt     time.Time          `db:"created_at" json:"created_at"`
}

type FinancialEventStore struct {
	db *sqlx.DB
}

func NewFinancialEventStore(db *sql.DB) *FinancialEventStore {
	return &FinancialEventStore{db: sqlx.NewDb(db, "postgres")}
}

// Apply atomically stores an event and adjusts its goal. The event ID makes retries idempotent.
func (s *FinancialEventStore) Apply(ctx context.Context, event FinancialEvent) (bool, error) {
	if event.ID == uuid.Nil {
		return false, errors.New("financial event id is required")
	}
	if event.AmountKopecks <= 0 {
		return false, errors.New("financial event amount must be positive")
	}
	if event.Type != FinancialEventDeposit && event.Type != FinancialEventWithdrawal {
		return false, fmt.Errorf("unsupported financial event type %q", event.Type)
	}
	if event.Currency != "RUB" {
		return false, fmt.Errorf("unsupported financial event currency %q", event.Currency)
	}
	if event.SourceText == "" {
		return false, errors.New("financial event source text is required")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("failed to begin financial event transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var goal Goal
	const getGoal = `
		SELECT id, user_id, name, target_amount_kopecks, current_amount_kopecks,
		       currency, status, created_at, updated_at
		FROM goals
		WHERE id = $1 AND user_id = $2
		FOR UPDATE`
	if err := tx.GetContext(ctx, &goal, getGoal, event.GoalID, event.UserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("%w: id %s", ErrGoalNotFound, event.GoalID)
		}
		return false, fmt.Errorf("failed to lock goal %s: %w", event.GoalID, err)
	}
	if goal.Currency != event.Currency {
		return false, fmt.Errorf("financial event currency %s does not match goal currency %s", event.Currency, goal.Currency)
	}

	const insertEvent = `
		INSERT INTO financial_events (
			id, user_id, goal_id, diary_entry_id, event_type, amount_kopecks, currency, source_text
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING`
	result, err := tx.ExecContext(
		ctx,
		insertEvent,
		event.ID,
		event.UserID,
		event.GoalID,
		event.DiaryEntryID,
		event.Type,
		event.AmountKopecks,
		event.Currency,
		event.SourceText,
	)
	if err != nil {
		return false, fmt.Errorf("failed to save financial event %s: %w", event.ID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to inspect insert result for financial event %s: %w", event.ID, err)
	}
	if rowsAffected == 0 {
		var existing FinancialEvent
		const getEvent = `
			SELECT id, user_id, goal_id, diary_entry_id, event_type, amount_kopecks,
			       currency, source_text, created_at
			FROM financial_events
			WHERE id = $1`
		if err := tx.GetContext(ctx, &existing, getEvent, event.ID); err != nil {
			return false, fmt.Errorf("failed to check duplicate financial event %s: %w", event.ID, err)
		}
		if existing.UserID != event.UserID ||
			existing.GoalID != event.GoalID ||
			existing.DiaryEntryID != event.DiaryEntryID ||
			existing.Type != event.Type ||
			existing.AmountKopecks != event.AmountKopecks ||
			existing.Currency != event.Currency ||
			existing.SourceText != event.SourceText {
			return false, fmt.Errorf("%w: id %s", ErrFinancialEventConflict, event.ID)
		}
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("failed to commit duplicate financial event check: %w", err)
		}
		return false, nil
	}

	var newAmount int64
	switch event.Type {
	case FinancialEventDeposit:
		if goal.CurrentAmountKopecks > math.MaxInt64-event.AmountKopecks {
			return false, errors.New("financial event would overflow goal progress")
		}
		newAmount = goal.CurrentAmountKopecks + event.AmountKopecks
	case FinancialEventWithdrawal:
		if event.AmountKopecks > goal.CurrentAmountKopecks {
			return false, ErrInsufficientGoalProgress
		}
		newAmount = goal.CurrentAmountKopecks - event.AmountKopecks
	}

	const updateGoal = `
		UPDATE goals
		SET current_amount_kopecks = $2,
		    status = CASE WHEN $2 >= target_amount_kopecks THEN 'completed' ELSE 'active' END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`
	if _, err := tx.ExecContext(ctx, updateGoal, goal.ID, newAmount); err != nil {
		return false, fmt.Errorf("failed to update progress for goal %s: %w", goal.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("failed to commit financial event %s: %w", event.ID, err)
	}
	return true, nil
}
