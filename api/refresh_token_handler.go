package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type TokenRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type TokenRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (r TokenRefreshRequest) Validate() error {
	if r.RefreshToken == "" {
		return errors.New("refresh token is required")
	}

	return nil
}

func (s *ApiServer) tokenRefreshHandler() http.HandlerFunc {
	return handler(func(w http.ResponseWriter, r *http.Request) error {
		req, err := decode[TokenRefreshRequest](r)
		if err != nil {
			return NewErrWihStatus(http.StatusBadRequest, err)
		}

		currentRefreshToken, err := s.jwtManager.Parse(req.RefreshToken)
		if err != nil {
			return NewErrWihStatus(http.StatusUnauthorized, err)
		}

		userIdStr, err := currentRefreshToken.Claims.GetSubject()
		if err != nil {
			return NewErrWihStatus(http.StatusUnauthorized, err)
		}

		userId, err := uuid.Parse(userIdStr)
		if err != nil {
			return NewErrWihStatus(http.StatusUnauthorized, err)
		}

		currentRefreshTokenRecord, err := s.store.RefreshTokenStore.ByPrimaryKey(r.Context(), userId, currentRefreshToken)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, sql.ErrNoRows) {
				status = http.StatusUnauthorized
			}
			return NewErrWihStatus(status, err)
		}

		if currentRefreshTokenRecord.ExpiresAt.Before(time.Now()) {
			return NewErrWihStatus(http.StatusUnauthorized, fmt.Errorf("refresh token expired"))
		}

		tokenPair, err := s.jwtManager.GenerateTokenPair(userId)
		if err != nil {
			return NewErrWihStatus(http.StatusInternalServerError, err)
		}

		if _, err := s.store.RefreshTokenStore.DeleteUserTokens(r.Context(), userId); err != nil {
			return NewErrWihStatus(http.StatusInternalServerError, err)
		}

		if _, err := s.store.RefreshTokenStore.Create(r.Context(), userId, tokenPair.RefreshToken); err != nil {
			return NewErrWihStatus(http.StatusInternalServerError, err)
		}

		if err := encode(ApiResponse[TokenRefreshResponse]{
			Data: &TokenRefreshResponse{
				AccessToken:  tokenPair.AccessToken.Raw,
				RefreshToken: tokenPair.RefreshToken.Raw,
			},
		}, http.StatusOK, w); err != nil {
			return NewErrWihStatus(http.StatusInternalServerError, err)
		}

		return nil
	})
}
