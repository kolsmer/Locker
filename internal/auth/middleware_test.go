package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func signedToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return tokenStr
}

func TestExtractToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer abc.def")

		token, err := ExtractToken(r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if token != "abc.def" {
			t.Fatalf("expected token abc.def, got %q", token)
		}
	})

	t.Run("missing header", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if _, err := ExtractToken(r); err == nil {
			t.Fatal("expected error for missing header")
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Token abc.def")
		if _, err := ExtractToken(r); err == nil {
			t.Fatal("expected error for invalid format")
		}
	})
}

func TestValidateToken(t *testing.T) {
	secret := "test-secret"
	m := NewMiddleware(secret)

	t.Run("success", func(t *testing.T) {
		tok := signedToken(t, secret, jwt.MapClaims{
			"sub":   "42",
			"login": "admin",
			"role":  "superadmin",
		})

		claims, err := m.ValidateToken(tok)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if claims.AdminID != 42 {
			t.Fatalf("expected AdminID=42, got %d", claims.AdminID)
		}
		if claims.Login != "admin" {
			t.Fatalf("expected Login=admin, got %q", claims.Login)
		}
		if claims.Role != "superadmin" {
			t.Fatalf("expected Role=superadmin, got %q", claims.Role)
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		tok := signedToken(t, "other-secret", jwt.MapClaims{"sub": "42"})
		if _, err := m.ValidateToken(tok); err == nil {
			t.Fatal("expected error for wrong secret")
		}
	})

	t.Run("invalid subject", func(t *testing.T) {
		tok := signedToken(t, secret, jwt.MapClaims{"sub": "not-a-number"})
		if _, err := m.ValidateToken(tok); err == nil {
			t.Fatal("expected error for invalid subject")
		}
	})
}

func TestRequireAdmin(t *testing.T) {
	secret := "test-secret"
	m := NewMiddleware(secret)

	t.Run("unauthorized without token", func(t *testing.T) {
		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
		})

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		rr := httptest.NewRecorder()

		m.RequireAdmin(next).ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rr.Code)
		}
		if nextCalled {
			t.Fatal("next handler must not be called")
		}
	})

	t.Run("passes request with valid token and stores claims", func(t *testing.T) {
		tok := signedToken(t, secret, jwt.MapClaims{
			"sub":   "7",
			"login": "ops",
			"role":  "admin",
		})

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				t.Fatal("expected claims in context")
			}
			if claims.AdminID != 7 {
				t.Fatalf("expected AdminID=7, got %d", claims.AdminID)
			}
			w.WriteHeader(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()

		m.RequireAdmin(next).ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rr.Code)
		}
	})
}
