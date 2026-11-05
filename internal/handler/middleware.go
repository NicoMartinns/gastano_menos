package handler

import (
	"context"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/NicoMartinns/gastano-menos/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "user_id"
const UserKey contextKey = "user"

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type AuthUserLookup interface {
	FindByID(string) (*domain.User, error)
}

func NewAuthMiddleware(users AuthUserLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.Fields(r.Header.Get("Authorization"))
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, "token não informado ou inválido", http.StatusUnauthorized)
				return
			}
			secret := os.Getenv("JWT_SECRET")
			if strings.TrimSpace(secret) == "" {
				http.Error(w, "autenticação indisponível", http.StatusInternalServerError)
				return
			}
			claims := &domain.AccessClaims{}
			token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
			if err != nil || !token.Valid || !uuidPattern.MatchString(claims.Subject) {
				http.Error(w, "token inválido ou expirado", http.StatusUnauthorized)
				return
			}
			user, err := users.FindByID(claims.Subject)
			if err != nil {
				log.Printf("erro ao verificar acesso do usuário: %v", err)
				http.Error(w, "erro ao verificar acesso", http.StatusInternalServerError)
				return
			}
			if user == nil || !user.IsActive || user.SessionVersion != claims.SessionVersion {
				http.Error(w, "sessão inválida ou acesso não liberado", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, user.ID)
			ctx = context.WithValue(ctx, UserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(UserKey).(*domain.User)
		if !ok || !user.IsActive || user.Role != domain.RoleAdmin {
			http.Error(w, "acesso restrito ao administrador", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
