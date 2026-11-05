package config

import (
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/cors"
)

// CORSHandler reads a comma-separated allowlist at startup. A configured list
// replaces the defaults; empty entries are ignored.
func CORSHandler() func(http.Handler) http.Handler {
	origins := make([]string, 0)
	for _, origin := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		origins = []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://localhost:5580", "http://201.23.83.151"}
	}
	return cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	})
}
