package auth

import (
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	auth := NewAuth("test-secret-key")

	token, err := auth.GenerateToken(123, "paul@example.com", 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("expected token, got empty string")
	}

	t.Logf("✅ Generated token: %s", token)
}

func TestValidateToken(t *testing.T) {
	auth := NewAuth("test-secret-key")

	// Generate a JWT.
	token, _ := auth.GenerateToken(123, "paul@example.com", 24*time.Hour)

	// Validate the JWT.
	claims, err := auth.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	// Verify claims.
	if claims.UserId != 123 {
		t.Errorf("expected user_id 123, got %d", claims.UserId)
	}

	if claims.Email != "paul@example.com" {
		t.Errorf("expected email paul@example.com, got %s", claims.Email)
	}

	t.Log("✅ Token validated successfully")
}

func TestValidateInvalidToken(t *testing.T) {
	auth := NewAuth("test-secret-key")

	// Try to validate garbage token.
	_, err := auth.ValidateToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}

	t.Log("✅ Correctly rejected invalid token")
}

func TestValidateExpiredToken(t *testing.T) {
	auth := NewAuth("test-secret-key")

	// Generate JWT that expires immediately.
	token, _ := auth.GenerateToken(123, "paul@example.com", -1*time.Second)

	// Try to validate expired token.
	_, err := auth.ValidateToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}

	t.Log("✅ Correctly rejected expired token")
}

func TestValidateWrongSecret(t *testing.T) {
	auth1 := NewAuth("secret-1")
	auth2 := NewAuth("secret-2")

	// Generate with auth1.
	token, _ := auth1.GenerateToken(123, "paul@example.com", 24*time.Hour)

	// Try to validate with wrong secret.
	_, err := auth2.ValidateToken(token)
	if err == nil {
		t.Error("expected error for wrong secret")
	}

	t.Log("✅ Correctly rejected token signed with different secret")
}
