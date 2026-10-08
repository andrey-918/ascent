package api

import (
	"ascent/config"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var signingMethod = jwt.SigningMethodHS256

type JWTManager struct {
	config *config.Config
}

func NewJWTManager(config *config.Config) *JWTManager {
	return &JWTManager{config: config}
}

type TokenPair struct {
	AccessToken  *jwt.Token
	RefreshToken *jwt.Token
}

type CustomClaims struct {
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

func (j *JWTManager) Parse(token string) (*jwt.Token, error) {
	parser := jwt.NewParser()
	jwtToken, err := parser.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if t.Method != signingMethod {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(j.config.JWTSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}
	return jwtToken, nil
}

func (j *JWTManager) IsAccessToken(token *jwt.Token) bool {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		if cc, ok := token.Claims.(*CustomClaims); ok {
			return cc.TokenType == "access"
		}
		return false
	}
	tt, _ := claims["token_type"].(string)
	return tt == "access"
}

func (j *JWTManager) GenerateTokenPair(userId uuid.UUID) (*TokenPair, error) {
	now := time.Now()
	issuer := "http://" + j.config.ApiServerHost + ":" + j.config.ApiServerPort
	key := []byte(j.config.JWTSecret)

	access := jwt.NewWithClaims(signingMethod, CustomClaims{
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userId.String(),
			Issuer:    issuer,
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})
	accessStr, err := access.SignedString(key)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}
	accessToken, err := j.Parse(accessStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse access token %w", err)
	}

	refresh := jwt.NewWithClaims(signingMethod, CustomClaims{
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userId.String(),
			Issuer:    issuer,
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})
	refreshStr, err := refresh.SignedString(key)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}
	refreshToken, err := j.Parse(refreshStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}

	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}
