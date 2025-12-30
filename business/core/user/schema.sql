-- Users table schema.
-- This is what userdb.go expects to exist.

CREATE TABLE IF NOT EXISTS users (
	id            TEXT PRIMARY KEY,                -- UUID as string.
	email         TEXT UNIQUE NOT NULL,            -- Email login.
	username      TEXT UNIQUE NOT NULL,            -- Display name.
	password_hash TEXT NOT NULL,                   -- Bcrypt hash.
	created_at    TIMESTAMP DEFAULT NOW(),
	updated_at    TIMESTAMP DEFAULT NOW()
);

-- Index for fast lookups.
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- Example queries this schema supports:
-- SELECT * FROM users WHERE email = 'paul@example.com';   -- Fast (indexed).
-- SELECT * FROM users WHERE username = 'paul';            -- Fast (indexed).
-- SELECT * FROM users WHERE id = 'uuid-123';              -- Fast (primary key).
