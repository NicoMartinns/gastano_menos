package service

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/NicoMartinns/gastano-menos/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type userStoreStub struct {
	saved *domain.User
	err   error
}

func (s *userStoreStub) List(int, int) ([]domain.User, int, error) { return nil, 0, s.err }
func (s *userStoreStub) Create(u *domain.User) error               { s.saved = u; return s.err }
func (s *userStoreStub) CreateInitialAdmin(u *domain.User) error   { s.saved = u; return s.err }
func (s *userStoreStub) Update(u *domain.User) error               { s.saved = u; return s.err }

type loginStoreStub struct {
	user *domain.User
	err  error
}

func (s loginStoreStub) FindByEmail(string) (*domain.User, error) { return s.user, s.err }

func TestCreateUserHashesPasswordAndHidesSecrets(t *testing.T) {
	repo := &userStoreStub{}
	u := &domain.User{Name: " Test User ", Email: "test@example.com", Role: domain.RoleUser, IsActive: false}
	password := "unit-test-password"
	if err := NewUserService(repo).Create(u, password); err != nil {
		t.Fatal(err)
	}
	if repo.saved.Name != "Test User" || repo.saved.IsActive {
		t.Fatal("name or access changed unexpectedly")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.saved.PasswordHash), []byte(password)); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(repo.saved)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), repo.saved.PasswordHash) || strings.Contains(string(data), "session_version") || strings.Contains(string(data), "PasswordHash") {
		t.Fatal("sensitive fields in response")
	}
}

func TestInvalidUserDoesNotReachRepository(t *testing.T) {
	for _, tc := range []struct{ name, email, role, password string }{
		{"", "test@example.com", "USER", "unit-test-password"},
		{"Test", "invalid", "USER", "unit-test-password"},
		{"Test", "test@example.com", "OWNER", "unit-test-password"},
		{"Test", "test@example.com", "USER", "short"},
		{"Test", "test@example.com", "USER", strings.Repeat("x", 73)},
		{"Test", "Display <test@example.com>", "USER", "unit-test-password"},
	} {
		repo := &userStoreStub{}
		err := NewUserService(repo).Create(&domain.User{Name: tc.name, Email: tc.email, Role: tc.role}, tc.password)
		if !errors.Is(err, domain.ErrInvalidUser) || repo.saved != nil {
			t.Fatalf("invalid payload reached storage: %v", err)
		}
	}
}

func TestInitialAdminAndProtectedSelfAccess(t *testing.T) {
	repo := &userStoreStub{}
	s := NewUserService(repo)
	if err := s.CreateInitialAdmin("Admin", "admin@example.com", "unit-test-password"); err != nil {
		t.Fatal(err)
	}
	if repo.saved.Role != domain.RoleAdmin || !repo.saved.IsActive {
		t.Fatal("bootstrap did not create active admin")
	}
	repo.saved = nil
	for _, u := range []*domain.User{
		{ID: "actor", Name: "Admin", Email: "admin@example.com", Role: domain.RoleUser, IsActive: true},
		{ID: "actor", Name: "Admin", Email: "admin@example.com", Role: domain.RoleAdmin, IsActive: false},
	} {
		if err := s.Update("actor", u); !errors.Is(err, domain.ErrSelfAccessChange) {
			t.Fatalf("self access change allowed: %v", err)
		}
		if repo.saved != nil {
			t.Fatal("self access change reached storage")
		}
	}
	repo.err = domain.ErrLastAdmin
	err := s.Update("other", &domain.User{ID: "target", Name: "Admin", Email: "admin@example.com", Role: domain.RoleUser, IsActive: true})
	if !errors.Is(err, domain.ErrLastAdmin) {
		t.Fatal("last admin conflict swallowed")
	}
}

func TestLoginAccessAndSessionVersion(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-secret")
	hash, err := bcrypt.GenerateFromPassword([]byte("unit-test-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	u := &domain.User{ID: "actor", Role: domain.RoleAdmin, IsActive: true, PasswordHash: string(hash), SessionVersion: 3}
	token, err := NewAuthService(loginStoreStub{user: u}).Login("admin@example.com", "unit-test-password")
	if err != nil {
		t.Fatal(err)
	}
	claims := &domain.AccessClaims{}
	_, err = jwt.ParseWithClaims(token, claims, func(*jwt.Token) (interface{}, error) { return []byte("unit-test-secret"), nil })
	if err != nil || claims.Subject != "actor" || claims.SessionVersion != 3 || claims.ExpiresAt.Time.Before(time.Now()) {
		t.Fatalf("invalid token claims: %v", err)
	}
	for _, tc := range []struct {
		user     *domain.User
		password string
	}{
		{nil, "unit-test-password"},
		{&domain.User{IsActive: false, PasswordHash: string(hash)}, "unit-test-password"},
		{u, "incorrect-password"},
	} {
		if _, err := NewAuthService(loginStoreStub{user: tc.user}).Login("admin@example.com", tc.password); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("unauthorized login accepted: %v", err)
		}
	}
	t.Setenv("JWT_SECRET", "")
	if _, err := generateToken(u); err == nil {
		t.Fatal("empty signing secret accepted")
	}
}
