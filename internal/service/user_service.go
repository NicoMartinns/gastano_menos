package service

import (
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/NicoMartinns/gastano-menos/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type UserStore interface {
	List(limit, offset int) ([]domain.User, int, error)
	Create(*domain.User) error
	CreateInitialAdmin(*domain.User) error
	Update(*domain.User) error
}

type UserService struct{ repo UserStore }

func NewUserService(repo UserStore) *UserService { return &UserService{repo: repo} }

func validUser(u *domain.User) bool {
	u.Name = strings.TrimSpace(u.Name)
	u.Email = strings.TrimSpace(u.Email)
	email, err := mail.ParseAddress(u.Email)
	return u.Name != "" && utf8.RuneCountInString(u.Name) <= 255 &&
		utf8.RuneCountInString(u.Email) <= 255 && err == nil && email.Address == u.Email &&
		(u.Role == domain.RoleUser || u.Role == domain.RoleAdmin)
}

func prepareUser(u *domain.User, password string) error {
	if !validUser(u) || len(password) < 12 || len(password) > 72 {
		return domain.ErrInvalidUser
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

func (s *UserService) List(limit, offset int) ([]domain.User, int, error) {
	return s.repo.List(limit, offset)
}

func (s *UserService) Create(u *domain.User, password string) error {
	if err := prepareUser(u, password); err != nil {
		return err
	}
	return s.repo.Create(u)
}

func (s *UserService) CreateInitialAdmin(name, email, password string) error {
	u := &domain.User{Name: name, Email: email, Role: domain.RoleAdmin, IsActive: true}
	if err := prepareUser(u, password); err != nil {
		return err
	}
	return s.repo.CreateInitialAdmin(u)
}

func (s *UserService) Update(actorID string, u *domain.User) error {
	if !validUser(u) {
		return domain.ErrInvalidUser
	}
	if actorID == u.ID && (!u.IsActive || u.Role != domain.RoleAdmin) {
		return domain.ErrSelfAccessChange
	}
	return s.repo.Update(u)
}
