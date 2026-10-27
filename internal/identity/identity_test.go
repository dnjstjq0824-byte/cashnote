package identity

import (
	"context"
	"testing"

	"merchant-os/internal/platform/auth"
)

func TestIdentityService(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	t.Run("Register and Login successfully", func(t *testing.T) {
		user, err := svc.Register(ctx, "owner@example.com", "SecurePassword123!", "홍길동")
		if err != nil {
			t.Fatalf("registration failed: %v", err)
		}

		if user.Email != "owner@example.com" {
			t.Errorf("expected email owner@example.com, got %s", user.Email)
		}

		// Login
		loggedIn, token, err := svc.Login(ctx, "owner@example.com", "SecurePassword123!")
		if err != nil {
			t.Fatalf("login failed: %v", err)
		}
		if loggedIn.ID != user.ID {
			t.Errorf("expected user ID %s, got %s", user.ID, loggedIn.ID)
		}
		if token == "" {
			t.Fatalf("expected non-empty token")
		}

		// Validate JWT Token
		claims, err := auth.ValidateToken(token)
		if err != nil {
			t.Fatalf("token validation failed: %v", err)
		}
		if claims.UserID != user.ID || claims.Email != user.Email {
			t.Errorf("token claims mismatch: %v", claims)
		}
	})

	t.Run("Duplicate email registration fails with conflict", func(t *testing.T) {
		_, err := svc.Register(ctx, "owner@example.com", "AnotherPassword!", "이순신")
		if err == nil {
			t.Fatalf("expected conflict on duplicate email registration")
		}
	})

	t.Run("Wrong password login fails with unauthorized", func(t *testing.T) {
		_, _, err := svc.Login(ctx, "owner@example.com", "WrongPassword!")
		if err == nil {
			t.Fatalf("expected unauthorized error on wrong password")
		}
	})
}
