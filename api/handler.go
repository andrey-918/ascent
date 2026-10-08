package api

import (
	"database/sql"
	"encoding/json"
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

func (s *ApiServer) signupHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc{
	return handler(func(w http.ResponseWriter, r *http.Request) error {
		var req SignupRequst
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return NewErrWihStatus(http.StatusBadRequest, fmt.Errorf("invalid request body: %v", err))
		}
		defer r.Body.Close()

		if err := req.Validate(); err != nil {
			return NewErrWihStatus(http.StatusBadRequest, fmt.Errorf("invalid request: %v", err))
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

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(ApiResponse[struct{}]{
			Message: "successfully signed up user",
		}); err != nil {
			return NewErrWihStatus(http.StatusInternalServerError, err)
		}
		
		return nil
	})
}
