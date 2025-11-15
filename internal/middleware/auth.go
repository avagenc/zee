package middleware

import (
	"log"
	"net/http"
)

type Authenticator struct {
	apiKey string
}

func NewAuthenticator(apiKey string) *Authenticator {
	return &Authenticator{apiKey: apiKey}
}

func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientAPIKey := r.Header.Get("x-avagenc-api-key")
		if clientAPIKey != a.apiKey {
			log.Printf("Authentication failed: Invalid API Key. Request from %s", r.RemoteAddr)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
