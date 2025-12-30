package user

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// Test DB connection string. Assumes port-forward of 5432:5432 from localhost to postgres svc is in place.
const testDBConnStr = "postgres://taskboard:taskboard@localhost:5432/taskboard_test?sslmode=disable"

// setupTestDb creates fresh users table for each test.
func setupTestDb(t *testing.T) (*sql.DB, func()) {
	t.Helper() // Mark setupTestDb as a helper function.

	db, err := sql.Open("postgres", testDBConnStr)
	if err != nil {
		t.Skip("Skipping test: no test db available.")
		return nil, nil
	}

	// Verify connection.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Skip("Skipping test: cannot connect to test db")
		return nil, nil
	}

	// Create new schema for this test.
	schema := `
		DROP TABLE IF EXISTS users CASCADE;

		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX idx_users_email ON users(email);
		CREATE INDEX idx_users_username ON users(username);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// Instantiate cleanup function.
	cleanup := func() {
		db.Exec("DROP TABLE IF EXISTS users CASCADE")
		db.Close()
	}

	return db, cleanup
}

func TestStoreCreate(t *testing.T) {
	db, cleanup := setupTestDb(t)
	if db == nil {
		return // Test was skipped.
	}
	defer cleanup() // Cleanup in the test db after this test is done.

	store := NewStore(db)
	ctx := context.Background()

	t.Run("create valid user", func(t *testing.T) {
		// Create user.
		user, err := New("test@example.com", "testuser", "password123")
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		// Store in DB.
		err = store.Create(ctx, user)
		if err != nil {
			t.Fatalf("failed to store user in DB: %v", err)
		}

		t.Logf("✅ Stored user with ID: %s", user.ID())
	})

	t.Run("reject duplicate email", func(t *testing.T) {
		// Create first user.
		user1, _ := New("duplicate@example.com", "user1", "password123")
		store.Create(ctx, user1)

		// Try to create second user with identical email.
		user2, _ := New("duplicate@example.com", "user2", "password123")
		err := store.Create(ctx, user2)

		if err != ErrEmailTaken {
			t.Errorf("expected ErrEmailTaken for duplicate email, got: %v", err)
		}

		t.Log("✅ Correctly rejected duplicate email")
	})
}

func TestStoreQueryByEmail(t *testing.T) {
	db, cleanup := setupTestDb(t)
	if db == nil {
		return
	}
	defer cleanup()

	store := NewStore(db)
	ctx := context.Background()

	t.Run("find existing user", func(t *testing.T) {
		// Create and save a user.
		original, _ := New("test@example.com", "testuser", "password123")
		store.Create(ctx, original)

		// Query by email.
		found, err := store.QueryByEmail(ctx, "test@example.com")
		if err != nil {
			t.Fatalf("failed to query user: %v", err)
		}

		// Verify all fields match.
		if found.ID() != original.ID() {
			t.Errorf("ID mismatch: expected %s, got %s", original.ID(), found.ID())
		}

		if found.Email() != original.Email() {
			t.Errorf("Email mismatch: expected %s, got %s", original.Email(), found.Email())
		}

		if found.Username() != original.Username() {
			t.Errorf("Username mismatch: expected %s, got %s", original.Username(), found.Username())
		}

		// Verify password still works after DB roundtrip.
		if !found.Authenticate("password123") {
			t.Error("loaded user should authenticate with original password.")
		}

		t.Log("✅ Successfully queried and authenticated user")
	})

	t.Run("return error for missing user", func(t *testing.T) {
		_, err := store.QueryByEmail(ctx, "notfound@example.com")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}

		t.Log("✅ Correctly returned ErrNotFound for missing user")
	})
}

func TestStoreQueryById(t *testing.T) {
	db, cleanup := setupTestDb(t)
	if db == nil {
		return
	}
	defer cleanup()

	store := NewStore(db)
	ctx := context.Background()

	// Create and save user.
	original, _ := New("test@example.com", "testuser", "password123")
	store.Create(ctx, original)

	// Query by id.
	found, err := store.QueryById(ctx, original.ID())
	if err != nil {
		t.Fatalf("failed to query user: %v", err)
	}

	if found.Email() != original.Email() {
		t.Errorf("Email mismatch: expected %s, got %s", original.Email(), found.Email())
	}

	t.Log("✅ Successfully queried user by ID")
}

func TestStoreQueryByUsername(t *testing.T) {
	db, cleanup := setupTestDb(t)
	if db == nil {
		return
	}
	defer cleanup()

	store := NewStore(db)
	ctx := context.Background()

	// Create and save a user.
	original, _ := New("test@example.com", "testuser", "password123")
	store.Create(ctx, original)

	// Query by username.
	found, err := store.QueryByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("failed to query user: %v", err)
	}

	if found.Email() != original.Email() {
		t.Errorf("Email mismatch: expected %s, got %s", original.Email(), found.Email())
	}

	t.Log("✅ Successfully queried user by username")
}
