package api

import (
	"database/sql"
	"encoding/json"
	"errors"
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

func (s *ApiServer) signupHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req SignupRequst
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	existingUser, err := s.store.Users.ByEmail(r.Context(), req.Email)
	switch {
	case err == nil && existingUser != nil:
		http.Error(w, "user already exists", http.StatusConflict)
		return
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := s.store.Users.CreateUser(r.Context(), req.Email, req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(ApiResponse[struct{}]{
		Message: "successfully signed up user",
	})
}
