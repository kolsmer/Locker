package config

import "testing"

func TestNewConfig_OverridesFromEnv(t *testing.T) {
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PORT", "6543")
	t.Setenv("DB_USER", "locker_user")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "locker_test")
	t.Setenv("PORT", "9999")
	t.Setenv("JWT_SECRET", "jwt-test-secret")
	t.Setenv("ENVIRONMENT", "test")
	t.Setenv("YOOKASSA_SHOP_ID", "shop-id")
	t.Setenv("YOOKASSA_API_KEY", "api-key")

	c := NewConfig()

	if c.DBHost != "db" {
		t.Fatalf("expected DBHost=db, got %q", c.DBHost)
	}
	if c.DBPort != "6543" {
		t.Fatalf("expected DBPort=6543, got %q", c.DBPort)
	}
	if c.DBUser != "locker_user" {
		t.Fatalf("expected DBUser=locker_user, got %q", c.DBUser)
	}
	if c.DBPassword != "secret" {
		t.Fatalf("expected DBPassword=secret, got %q", c.DBPassword)
	}
	if c.DBName != "locker_test" {
		t.Fatalf("expected DBName=locker_test, got %q", c.DBName)
	}
	if c.Port != "9999" {
		t.Fatalf("expected Port=9999, got %q", c.Port)
	}
	if c.JWTSecret != "jwt-test-secret" {
		t.Fatalf("expected JWTSecret=jwt-test-secret, got %q", c.JWTSecret)
	}
	if c.Environment != "test" {
		t.Fatalf("expected Environment=test, got %q", c.Environment)
	}
	if c.YookassaShopID != "shop-id" {
		t.Fatalf("expected YookassaShopID=shop-id, got %q", c.YookassaShopID)
	}
	if c.YookassaAPIKey != "api-key" {
		t.Fatalf("expected YookassaAPIKey=api-key, got %q", c.YookassaAPIKey)
	}
}

func TestGetEnv(t *testing.T) {
	t.Run("returns env value when present", func(t *testing.T) {
		key := "LOCKER_CONFIG_TEST_KEY"
		t.Setenv(key, "value")

		got := getEnv(key, "fallback")
		if got != "value" {
			t.Fatalf("expected value, got %q", got)
		}
	})

	t.Run("returns default when missing", func(t *testing.T) {
		key := "LOCKER_CONFIG_TEST_MISSING_KEY"
		got := getEnv(key, "fallback")
		if got != "fallback" {
			t.Fatalf("expected fallback, got %q", got)
		}
	})
}
