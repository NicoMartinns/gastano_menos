package service

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/NicoMartinns/gastano-menos/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthUserStore interface {
	FindByEmail(string) (*domain.User, error)
}

type AuthService struct{ userRepo AuthUserStore }

func NewAuthService(userRepo AuthUserStore) *AuthService { return &AuthService{userRepo: userRepo} }

func (s *AuthService) Login(email, password string) (string, error) {
	user, err := s.userRepo.FindByEmail(strings.TrimSpace(email))
	if err != nil {
		return "", err
	}
	if user == nil || !user.IsActive {
		return "", domain.ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", domain.ErrInvalidCredentials
	}
	return generateToken(user)
}

func generateToken(user *domain.User) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("JWT_SECRET não configurado")
	}
	claims := domain.AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
		SessionVersion: user.SessionVersion,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
