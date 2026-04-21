package auth

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type Middleware struct {
	jwtSecret []byte
}

func NewMiddleware(secret string) *Middleware {
	return &Middleware{jwtSecret: []byte(secret)}
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
		tokenStr, err := ExtractToken(r)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := m.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type contextKey string

const claimsContextKey contextKey = "admin_claims"

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	v := ctx.Value(claimsContextKey)
	if v == nil {
		return nil, false
	}
	c, ok := v.(*Claims)
	return c, ok
}

type Claims struct {
	AdminID int64
	Login   string
	Role    string
	jwt.RegisteredClaims
}

func (m *Middleware) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return m.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	sub, _ := mapClaims["sub"].(string)
	adminID, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		return nil, errors.New("invalid subject")
	}

	login, _ := mapClaims["login"].(string)
	role, _ := mapClaims["role"].(string)

	return &Claims{AdminID: adminID, Login: login, Role: role}, nil
}
