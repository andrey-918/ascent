package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
)

type SignupRequst struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r SignupRequst) Validate() error {
	if r.Email == "" {
		return errors.New("email is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

type ApiResponse[T any] struct {
	Data    *T     `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

func (s *ApiServer) signupHandler() http.HandlerFunc {
	return handler(func(w http.ResponseWriter, r *http.Request) error {
		req, err := decode[SignupRequst](r)
		if err != nil {
			return NewErrWihStatus(http.StatusBadRequest, err)
		}

		existingUser, err := s.store.Users.ByEmail(r.Context(), req.Email)
		switch {
		case err == nil && existingUser != nil:
			return NewErrWihStatus(http.StatusConflict, fmt.Errorf("email already registered"))
		case err != nil && !errors.Is(err, sql.ErrNoRows):
			return NewErrWihStatus(http.StatusInternalServerError, err)
		}

		if _, err := s.store.Users.CreateUser(r.Context(), req.Email, req.Password); err != nil {
			return NewErrWihStatus(http.StatusInternalServerError, err)
		}

		if err := encode(ApiResponse[struct{}]{
			Message: "successfully signed up user",
		}, http.StatusCreated, w); err != nil {
			return NewErrWihStatus(http.StatusInternalServerError, err)
		}

		return nil
	})
}

type SigningRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

type SigningResponse struct {
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (r SigningRequest) Validate() error {
	if r.Email == "" {
		return errors.New("email is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

func (s *ApiServer) signinHandler() http.HandlerFunc {
	return handler(func(w http.ResponseWriter, r *http.Request) error {
		req, err := decode[SigningRequest](r)
		if err != nil {
			return NewErrWihStatus(http.StatusBadRequest, err)
		}

		user, err := s.store.Users.ByEmail(r.Context(), req.Email)
		if err != nil {
			return NewErrWihStatus(http.StatusInternalServerError, err)
		}
		if err := user.ComparePassword(req.Password); err != nil {
			return NewErrWihStatus(http.StatusUnauthorized, err)
		}

		tokenPair, err := s.jwtManager.GenerateTokenPair(user.Id)
		if err != nil {
			return NewErrWihStatus(http.StatusInternalServerError, err)
		}

		_, err = s.store.RefreshTokenStore.DeleteUserTokens(r.Context(), user.Id)
		if err != nil {
			return NewErrWihStatus(http.StatusInternalServerError, err)
		}

		_, err = s.store.RefreshTokenStore.Create(r.Context(), user.Id, tokenPair.RefreshToken)
		if err != nil {
			return NewErrWihStatus(http.StatusInternalServerError, err)
		}

		if err := encode(ApiResponse[SigningResponse]{
			Data: &SigningResponse{
				AccessToken: tokenPair.AccessToken.Raw,
				RefreshToken: tokenPair.RefreshToken.Raw,
			},
		}, http.StatusOK, w); err != nil {
			return NewErrWihStatus(http.StatusInternalServerError, err)
		}

		return nil
	})
}