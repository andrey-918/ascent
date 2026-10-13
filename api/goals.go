package api

import (
	"ascent/store"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const maxGoalNameLength = 120

type CreateGoalRequest struct {
	Name                string `json:"name"`
	TargetAmountKopecks int64  `json:"target_amount_kopecks"`
}

func (r CreateGoalRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if utf8.RuneCountInString(strings.TrimSpace(r.Name)) > maxGoalNameLength {
		return fmt.Errorf("name must be at most %d characters", maxGoalNameLength)
	}
	if r.TargetAmountKopecks <= 0 {
		return errors.New("target_amount_kopecks must be positive")
	}
	return nil
}

func (s *ApiServer) goalsHandler() http.HandlerFunc {
	return handler(func(w http.ResponseWriter, r *http.Request) error {
		user, ok := r.Context().Value(userCtxKey{}).(*store.User)
		if !ok {
			return NewErrWihStatus(http.StatusUnauthorized, errors.New("authentication required"))
		}

		switch r.Method {
		case http.MethodPost:
			req, err := decode[CreateGoalRequest](r)
			if err != nil {
				return NewErrWihStatus(http.StatusBadRequest, err)
			}
			goal, err := s.store.Goals.Create(r.Context(), user.Id, req.Name, req.TargetAmountKopecks)
			if err != nil {
				return NewErrWihStatus(http.StatusInternalServerError, err)
			}
			if err := encode(ApiResponse[store.Goal]{Data: goal}, http.StatusCreated, w); err != nil {
				return NewErrWihStatus(http.StatusInternalServerError, err)
			}
			return nil
		case http.MethodGet:
			goals, err := s.store.Goals.ListByUser(r.Context(), user.Id)
			if err != nil {
				return NewErrWihStatus(http.StatusInternalServerError, err)
			}
			if err := encode(ApiResponse[[]store.Goal]{Data: &goals}, http.StatusOK, w); err != nil {
				return NewErrWihStatus(http.StatusInternalServerError, err)
			}
			return nil
		default:
			return NewErrWihStatus(http.StatusMethodNotAllowed, errors.New("method not allowed"))
		}
	})
}

func (s *ApiServer) getGoalHandler() http.HandlerFunc {
	return handler(func(w http.ResponseWriter, r *http.Request) error {
		user, ok := r.Context().Value(userCtxKey{}).(*store.User)
		if !ok {
			return NewErrWihStatus(http.StatusUnauthorized, errors.New("authentication required"))
		}

		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			return NewErrWihStatus(http.StatusBadRequest, fmt.Errorf("invalid goal id: %w", err))
		}

		goal, err := s.store.Goals.ByID(r.Context(), id, user.Id)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, store.ErrGoalNotFound) {
				status = http.StatusNotFound
			}
			return NewErrWihStatus(status, err)
		}

		if err := encode(ApiResponse[store.Goal]{Data: goal}, http.StatusOK, w); err != nil {
			return NewErrWihStatus(http.StatusInternalServerError, err)
		}
		return nil
	})
}
