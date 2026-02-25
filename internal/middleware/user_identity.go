package middleware

import (
	"net/http"

	"github.com/avagenc/zee-agent/pkg/api"
)

func RequireUserIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("x-user-id")
		if userID == "" {
			api.WriteError(w, api.NewError(http.StatusUnauthorized, "UNAUTHORIZED", "Missing user identity"))
			return
		}

		ctx, err := api.NewContextWithUserID(r.Context(), userID)
		if err != nil {
			api.WriteError(w, api.NewError(http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user identity"))
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
