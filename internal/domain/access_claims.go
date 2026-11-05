package domain

import "github.com/golang-jwt/jwt/v5"

type AccessClaims struct {
	jwt.RegisteredClaims
	SessionVersion int `json:"session_version"`
}
