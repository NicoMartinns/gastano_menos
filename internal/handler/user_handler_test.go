package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NicoMartinns/gastano-menos/internal/domain"
	"github.com/NicoMartinns/gastano-menos/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

type managementStore struct {
	updated *domain.User
	err     error
	called  bool
}

func (s *managementStore) List(int, int) ([]domain.User, int, error) {
	s.called = true
	return []domain.User{{ID: testUserID, PasswordHash: "sensitive-hash", SessionVersion: 5}}, 1, s.err
}
func (s *managementStore) Create(u *domain.User) error {
	s.called = true
	u.ID = testUserID
	return s.err
}
func (s *managementStore) CreateInitialAdmin(*domain.User) error { return errors.New("not used") }
func (s *managementStore) Update(u *domain.User) error           { s.called = true; s.updated = u; return s.err }

func TestManagementHTTPAccess(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-used-only-in-unit-tests")
	for _, role := range []string{domain.RoleUser, domain.RoleAdmin} {
		t.Run(role, func(t *testing.T) {
			store := &managementStore{}
			h := NewUserHandler(service.NewUserService(store))
			router := chi.NewRouter()
			router.Use(NewAuthMiddleware(accessLookup{user: &domain.User{ID: testUserID, Role: role, IsActive: true}}))
			router.Route("/admin/users", func(r chi.Router) {
				r.Use(AdminMiddleware)
				r.Get("/", h.List)
				r.Post("/", h.Create)
				r.Put("/{id}", h.Update)
			})
			token := signedToken(t, jwt.SigningMethodHS256, jwt.MapClaims{"sub": testUserID, "exp": time.Now().Add(time.Hour).Unix()}, "test-secret-used-only-in-unit-tests")
			for _, path := range []string{"/admin/users", "/admin/users/"} {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.Header.Set("Authorization", "Bearer "+token)
				res := httptest.NewRecorder()
				router.ServeHTTP(res, req)
				expected := http.StatusForbidden
				if role == domain.RoleAdmin {
					expected = http.StatusOK
				}
				if res.Code != expected {
					t.Fatalf("%s: got %d, want %d: %s", path, res.Code, expected, res.Body.String())
				}
				if strings.Contains(res.Body.String(), "sensitive-hash") || strings.Contains(res.Body.String(), "session_version") {
					t.Fatal("secrets leaked")
				}
			}
			if store.called != (role == domain.RoleAdmin) {
				t.Fatal("unexpected storage access")
			}
		})
	}
}

func TestUserHandlerValidationAndErrors(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body string
		err                      error
		status                   int
		called                   bool
	}{
		{name: "invalid pagination", method: "GET", path: "/?limit=101", status: 400},
		{name: "negative offset", method: "GET", path: "/?offset=-1", status: 400},
		{name: "missing access decision", method: "POST", path: "/", body: `{"name":"Test","email":"test@example.com","role":"USER","password":"unit-test-password"}`, status: 400},
		{name: "cannot bind extra fields", method: "POST", path: "/", body: `{"name":"Test","email":"test@example.com","role":"USER","is_active":true,"password":"unit-test-password","session_version":5}`, status: 400},
		{name: "short password", method: "POST", path: "/", body: `{"name":"Test","email":"test@example.com","role":"USER","is_active":true,"password":"short"}`, status: 400},
		{name: "create", method: "POST", path: "/", body: `{"name":"Test","email":"test@example.com","role":"USER","is_active":true,"password":"unit-test-password"}`, status: 201, called: true},
		{name: "duplicate email", method: "POST", path: "/", body: `{"name":"Test","email":"test@example.com","role":"USER","is_active":true,"password":"unit-test-password"}`, err: domain.ErrEmailInUse, status: 409, called: true},
		{name: "invalid id", method: "PUT", path: "/bad-id", body: `{}`, status: 400},
		{name: "update", method: "PUT", path: "/" + testUserID, body: `{"name":"Test","email":"test@example.com","role":"USER","is_active":false}`, status: 200, called: true},
		{name: "missing account", method: "PUT", path: "/" + testUserID, body: `{"name":"Test","email":"test@example.com","role":"USER","is_active":false}`, err: domain.ErrUserNotFound, status: 404, called: true},
		{name: "internal error hidden", method: "GET", path: "/", err: errors.New("sensitive database detail"), status: 500, called: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &managementStore{err: tc.err}
			h := NewUserHandler(service.NewUserService(store))
			router := chi.NewRouter()
			router.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), UserIDKey, "other-admin")))
				})
			})
			router.Get("/", h.List)
			router.Post("/", h.Create)
			router.Put("/{id}", h.Update)
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("got %d, want %d: %s", res.Code, tc.status, res.Body.String())
			}
			if store.called != tc.called {
				t.Fatalf("storage called = %v", store.called)
			}
			if strings.Contains(res.Body.String(), "sensitive database detail") {
				t.Fatal("internal error exposed")
			}
			if tc.status == 201 {
				var data map[string]interface{}
				if err := json.Unmarshal(res.Body.Bytes(), &data); err != nil {
					t.Fatal(err)
				}
				if _, ok := data["password"]; ok {
					t.Fatal("password exposed")
				}
			}
		})
	}
}

func TestUppercaseIDCannotBypassSelfProtection(t *testing.T) {
	store := &managementStore{}
	h := NewUserHandler(service.NewUserService(store))
	router := chi.NewRouter()
	router.Put("/{id}", h.Update)
	req := httptest.NewRequest(http.MethodPut, "/"+strings.ToUpper(testUserID), strings.NewReader(`{"name":"Admin","email":"admin@example.com","role":"USER","is_active":false}`))
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, testUserID))
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusConflict || store.called {
		t.Fatalf("self protection bypassed: status=%d storage=%v", res.Code, store.called)
	}
}
