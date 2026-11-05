package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NicoMartinns/gastano-menos/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

const testUserID = "c365a5e1-6805-45c2-b193-f40e306995ab"

type accessLookup struct {
	user *domain.User
	err  error
}

func (s accessLookup) FindByID(string) (*domain.User, error) { return s.user, s.err }

func signedToken(t *testing.T, method jwt.SigningMethod, claims jwt.MapClaims, secret string) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestAccessMiddleware(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-used-only-in-unit-tests")
	baseClaims := func() jwt.MapClaims {
		return jwt.MapClaims{"sub": testUserID, "exp": time.Now().Add(time.Hour).Unix(), "session_version": 0}
	}
	active := &domain.User{ID: testUserID, Role: domain.RoleUser, IsActive: true}
	for _, tc := range []struct {
		name   string
		user   *domain.User
		err    error
		modify func(jwt.MapClaims)
		method jwt.SigningMethod
		secret string
		admin  bool
		status int
	}{
		{name: "active", user: active, status: 200},
		{name: "ordinary user cannot administer", user: active, admin: true, status: 403},
		{name: "admin", user: &domain.User{ID: testUserID, Role: domain.RoleAdmin, IsActive: true}, admin: true, status: 200},
		{name: "blocked", user: &domain.User{ID: testUserID, Role: domain.RoleAdmin, IsActive: false}, status: 401},
		{name: "deleted", status: 401},
		{name: "revoked even after reactivation", user: &domain.User{ID: testUserID, IsActive: true, SessionVersion: 2}, status: 401},
		{name: "expired", user: active, modify: func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Hour).Unix() }, status: 401},
		{name: "expiration required", user: active, modify: func(c jwt.MapClaims) { delete(c, "exp") }, status: 401},
		{name: "invalid subject", user: active, modify: func(c jwt.MapClaims) { c["sub"] = "not-a-uuid" }, status: 401},
		{name: "wrong signature", user: active, secret: "another-secret", status: 401},
		{name: "different HMAC algorithm", user: active, method: jwt.SigningMethodHS512, status: 401},
		{name: "lookup failure", err: errors.New("database unavailable"), status: 500},
		// Old tokens have no role claim; current database permissions control access.
		{name: "forged role claim ignored", user: active, modify: func(c jwt.MapClaims) { c["role"] = "ADMIN" }, admin: true, status: 403},
		{name: "legacy token", user: active, modify: func(c jwt.MapClaims) { delete(c, "session_version") }, status: 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := baseClaims()
			if tc.modify != nil {
				tc.modify(claims)
			}
			method := tc.method
			if method == nil {
				method = jwt.SigningMethodHS256
			}
			secret := tc.secret
			if secret == "" {
				secret = "test-secret-used-only-in-unit-tests"
			}
			endpoint := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Context().Value(UserIDKey) != testUserID {
					t.Error("actor missing from context")
				}
				w.WriteHeader(http.StatusOK)
			}))
			if tc.admin {
				endpoint = AdminMiddleware(endpoint)
			}
			endpoint = NewAuthMiddleware(accessLookup{tc.user, tc.err})(endpoint)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+signedToken(t, method, claims, secret))
			res := httptest.NewRecorder()
			endpoint.ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("got %d, want %d: %s", res.Code, tc.status, res.Body.String())
			}
		})
	}
}

func TestMiddlewareRejectsMissingTokenAndSecret(t *testing.T) {
	for _, secret := range []string{"", "configured"} {
		t.Setenv("JWT_SECRET", secret)
		handler := NewAuthMiddleware(accessLookup{})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("request should not reach endpoint") }))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if secret == "" {
			req.Header.Set("Authorization", "Bearer placeholder")
		}
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		expected := http.StatusUnauthorized
		if secret == "" {
			expected = http.StatusInternalServerError
		}
		if res.Code != expected {
			t.Fatalf("got %d, want %d", res.Code, expected)
		}
	}
}
