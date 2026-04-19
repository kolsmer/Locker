package auth

import (
	"errors"
	"net/http"
	"strings"
)

type Middleware struct {
	// Will inject repo
}

func NewMiddleware() *Middleware {
	return &Middleware{}
}

// ExtractToken из Authorization header
func ExtractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid authorization header")
	}

	return parts[1], nil
}

// RequireAuth middleware для админ endpoints
func (m *Middleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := ExtractToken(r)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// TODO: validate JWT
		_ = token

		next.ServeHTTP(w, r)
	})
}

type Claims struct {
	AdminID int64
	Role    string
}

func (m *Middleware) ValidateToken(token string) (*Claims, error) {
	// TODO: parse JWT and validate
	// For now just placeholder
	return &Claims{
		AdminID: 1,
		Role:    "admin",
	}, nil
}
