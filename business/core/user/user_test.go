package user

import "testing"

func TestNew(t *testing.T) {
	t.Run("valid user", func(t *testing.T) {
		user, err := New("test@example.com", "testuser", "password123")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if user.Email() != "test@example.com" {
			t.Errorf("expected email test@example.com, got: %s", user.Email())
		}

		if user.Username() != "testuser" {
			t.Errorf("expected username testuser, got: %s", user.Username())
		}

		if user.ID() == "" {
			t.Error("expected ID to be generated")
		}

		t.Logf("✅ Created user with ID: %s", user.ID())
	})

	t.Run("invalid email", func(t *testing.T) {
		_, err := New("invalid-email", "testuser", "password123")

		if err != ErrInvalidEmail {
			t.Errorf("expected ErrInvalidEmail, got: %v", err)
		}

		t.Log("✅ Correctly rejected invalid email")
	})

	t.Run("weak password", func(t *testing.T) {
		_, err := New("test@example.com", "testuser", "short")

		if err != ErrWeakPassword {
			t.Errorf("expected ErrWeakPassword, got: %v", err)
		}

		t.Log("✅ Correctly rejected weak password")
	})

	t.Run("empty username", func(t *testing.T) {
		_, err := New("test@example.com", "", "password123")

		if err != ErrEmptyUsername {
			t.Error("expected error for empty username")
		}

		t.Log("✅ Correctly rejected empty username")
	})
}

func TestAuthenticate(t *testing.T) {
	// Create a user.
	user, err := New("test@example.com", "testuser", "correct-password")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	t.Run("correct password", func(t *testing.T) {
		if !user.Authenticate("correct-password") {
			t.Error("authentication should succeed with correct password")
		}

		t.Log("✅ Correct password authenticated")
	})

	t.Run("wrong password", func(t *testing.T) {
		if user.Authenticate("wrong-password") {
			t.Error("authentication should fail with wrong password")
		}

		t.Log("✅ Wrong password rejected")
	})

	t.Run("empty password", func(t *testing.T) {
		if user.Authenticate("") {
			t.Error("authentication should fail with empty password")
		}

		t.Log("✅ Empty password rejected")
	})
}

func TestUnmarshal(t *testing.T) {
	// Create a user to get a real hash.
	original, err := New("test@example.com", "testuser", "password123")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Unmarshal. Reconstructs from DB data.
	reconstructed, err := Unmarshal(
		original.ID(),
		original.Email(),
		original.Username(),
		original.PasswordHash(),
	)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify fields.
	if reconstructed.ID() != original.ID() {
		t.Error("IDs don't match")
	}

	if reconstructed.Email() != original.Email() {
		t.Error("Emails don't match")
	}

	if reconstructed.Username() != original.Username() {
		t.Error("Usernames don't match")
	}

	// Verify password works.
	if !reconstructed.Authenticate("password123") {
		t.Error("reconstructed user should authenticate")
	}

	t.Log("✅ Successfully reconstructed user from database format")
}

func TestUnmarshalInvalidData(t *testing.T) {
	t.Run("empty id", func(t *testing.T) {
		_, err := Unmarshal("", "test@example.com", "testuser", "hash")
		if err == nil {
			t.Error("expected error for empty id")
		}
	})

	t.Run("empty email", func(t *testing.T) {
		_, err := Unmarshal("123", "", "testuser", "hash")
		if err == nil {
			t.Error("expected error from empty email")
		}
	})

	t.Log("✅ Correctly rejected invalid unmarshal data")
}
