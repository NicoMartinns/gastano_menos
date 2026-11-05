package config

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NicoMartinns/gastano-menos/internal/handler"
	"github.com/go-chi/chi/v5"
)

func TestCORS(t *testing.T) {
	for _, tc := range []struct {
		name, env, origin string
		allowed           bool
	}{
		{"tunnel", "", "http://localhost:5580", true},
		{"public", "", "http://201.23.83.151", true},
		{"legacy localhost", "", "http://localhost:3000", true},
		{"legacy loopback", "", "http://127.0.0.1:3000", true},
		{"untrusted", "", "http://untrusted.example", false},
		{"environment with whitespace", " , http://custom.example, ,", "http://custom.example", true},
		{"environment replaces defaults", "http://custom.example", "http://localhost:5580", false},
		{"empty entries use defaults", " , , ", "http://localhost:5580", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CORS_ALLOWED_ORIGINS", tc.env)
			r := chi.NewRouter()
			r.Use(CORSHandler())
			// Invalid JSON exercises the real login handler without a database.
			r.Post("/auth/login", handler.NewAuthHandler(nil).Login)
			r.Get("/categories", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) })
			for _, path := range []string{"/auth/login", "/categories"} {
				method := http.MethodPost
				if path == "/categories" {
					method = http.MethodGet
				}
				req := httptest.NewRequest(http.MethodOptions, path, nil)
				req.Header.Set("Origin", tc.origin)
				req.Header.Set("Access-Control-Request-Method", method)
				req.Header.Set("Access-Control-Request-Headers", "content-type, authorization")
				res := httptest.NewRecorder()
				r.ServeHTTP(res, req)
				if res.Code != http.StatusOK {
					t.Fatalf("preflight: %d", res.Code)
				}
				wantOrigin := ""
				if tc.allowed {
					wantOrigin = tc.origin
				}
				if got := res.Header().Get("Access-Control-Allow-Origin"); got != wantOrigin {
					t.Fatalf("origin: %q, want %q", got, wantOrigin)
				}
				if tc.allowed {
					if res.Header().Get("Access-Control-Allow-Methods") != method {
						t.Fatal("missing allowed method")
					}
					headers := strings.ToLower(strings.Join(res.Header().Values("Access-Control-Allow-Headers"), ","))
					for _, h := range []string{"content-type", "authorization"} {
						if !strings.Contains(headers, h) {
							t.Fatalf("missing header %s", h)
						}
					}
					if res.Header().Get("Access-Control-Allow-Credentials") != "true" {
						t.Fatal("missing credentials header")
					}
				}
				req = httptest.NewRequest(method, path, strings.NewReader("{"))
				req.Header.Set("Origin", tc.origin)
				res = httptest.NewRecorder()
				r.ServeHTTP(res, req)
				wantStatus := http.StatusBadRequest
				if path == "/categories" {
					wantStatus = http.StatusUnauthorized
				}
				if res.Code != wantStatus {
					t.Fatalf("route status: %d, want %d", res.Code, wantStatus)
				}
				if res.Header().Get("Access-Control-Allow-Origin") != wantOrigin {
					t.Fatal("unexpected origin on actual response")
				}
			}
		})
	}
}
