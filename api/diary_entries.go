package api

import (
	"ascent/store"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxDiaryEntryLength = 10000

type CreateDiaryEntryRequest struct {
	Content string `json:"content"`
}

func (r CreateDiaryEntryRequest) Validate() error {
	if strings.TrimSpace(r.Content) == "" {
		return errors.New("content is required")
	}
	if utf8.RuneCountInString(r.Content) > maxDiaryEntryLength {
		return fmt.Errorf("content must be at most %d characters", maxDiaryEntryLength)
	}
	return nil
}

func (s *ApiServer) diaryEntriesHandler() http.HandlerFunc {
	return handler(func(w http.ResponseWriter, r *http.Request) error {
		user, ok := r.Context().Value(userCtxKey{}).(*store.User)
		if !ok {
			return NewErrWihStatus(http.StatusUnauthorized, errors.New("authentication required"))
		}

		switch r.Method {
		case http.MethodPost:
			req, err := decode[CreateDiaryEntryRequest](r)
			if err != nil {
				return NewErrWihStatus(http.StatusBadRequest, err)
			}
			entry, err := s.store.DiaryEntries.Create(r.Context(), user.Id, req.Content)
			if err != nil {
				return NewErrWihStatus(http.StatusInternalServerError, err)
			}
			if err := encode(ApiResponse[store.DiaryEntry]{Data: entry}, http.StatusCreated, w); err != nil {
				return NewErrWihStatus(http.StatusInternalServerError, err)
			}
			return nil
		case http.MethodGet:
			query := r.URL.Query()
			limit := 20
			if rawLimit := query.Get("limit"); rawLimit != "" {
				parsedLimit, err := strconv.Atoi(rawLimit)
				if err != nil {
					return NewErrWihStatus(http.StatusBadRequest, fmt.Errorf("invalid limit: %w", err))
				}
				limit = parsedLimit
			}
			if limit >= 0 && limit > 100 {
				return NewErrWihStatus(http.StatusBadRequest, fmt.Errorf("limit must be between 1 and 100"))
			}
			offset := 0
			if rawOffset := query.Get("offset"); rawOffset != "" {
				parsedOffset, err := strconv.Atoi(rawOffset)
				if err != nil {
					return NewErrWihStatus(http.StatusBadRequest, fmt.Errorf("invalid offset: %w", err))
				}
				offset = parsedOffset
			}
			if offset < 0 {
				return NewErrWihStatus(http.StatusBadRequest, fmt.Errorf("offset must be non-negative"))
			}

			entries, err := s.store.DiaryEntries.ListByUser(r.Context(), user.Id, limit, offset)
			if err != nil {
				return NewErrWihStatus(http.StatusInternalServerError, err)
			}
			if err := encode(ApiResponse[[]store.DiaryEntry]{Data: &entries}, http.StatusOK, w); err != nil {
				return NewErrWihStatus(http.StatusInternalServerError, err)
			}
			return nil
		default:
			return NewErrWihStatus(http.StatusMethodNotAllowed, errors.New("method not allowed"))
		}
	})
}
