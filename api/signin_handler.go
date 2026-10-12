package api

import (
	"errors"
	"net/http"
)

type SigningRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SigningResponse struct {
	AccessToken  string `json:"access_token"`
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
				AccessToken:  tokenPair.AccessToken.Raw,
				RefreshToken: tokenPair.RefreshToken.Raw,
			},
		}, http.StatusOK, w); err != nil {
			return NewErrWihStatus(http.StatusInternalServerError, err)
		}

		return nil
	})
}
