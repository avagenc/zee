package middleware

import (
	"net/http"

	"github.com/avagenc/zee-api/pkg/api"
)

func AuthenticateAPIKey(validKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("x-avagenc-api-key")
			if key == "" {
				api.Respond(w, http.StatusUnauthorized, api.NewErrorResponse("UNAUTHORIZED", "Missing API key", nil))
				return
			}

			if key != validKey {
				api.Respond(w, http.StatusUnauthorized, api.NewErrorResponse("UNAUTHORIZED", "Invalid API key", nil))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
