package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type User struct {
	ID              int64
	Username        string
	PasswordHash    string
	UploadTokenName string
	UploadToken     string
	IsAdmin         bool
	CreatedAt       time.Time
}

type DB struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
    id               SERIAL PRIMARY KEY,
    username         TEXT NOT NULL UNIQUE,
    password_hash    TEXT NOT NULL,
    upload_token_name TEXT NOT NULL UNIQUE,
    upload_token     TEXT NOT NULL,
    is_admin         BOOLEAN NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

func New(dsn string) (*DB, error) {
	d, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := d.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if _, err := d.Exec(schema); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &DB{db: d}, nil
}

func (d *DB) HasAnyUsers(ctx context.Context) (bool, error) {
	var n int
	err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n > 0, err
}

func (d *DB) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (d *DB) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

func (d *DB) CreateUser(ctx context.Context, username, passwordHash, tokenName, token string, isAdmin bool) (*User, error) {
	var u User
	err := d.db.QueryRowContext(ctx,
		`INSERT INTO users (username, password_hash, upload_token_name, upload_token, is_admin)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, username, password_hash, upload_token_name, upload_token, is_admin, created_at`,
		username, passwordHash, tokenName, token, isAdmin,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.UploadTokenName, &u.UploadToken, &u.IsAdmin, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

func (d *DB) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := d.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, upload_token_name, upload_token, is_admin, created_at
		 FROM users WHERE username = $1`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.UploadTokenName, &u.UploadToken, &u.IsAdmin, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

func (d *DB) GetUserByID(ctx context.Context, id int64) (*User, error) {
	var u User
	err := d.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, upload_token_name, upload_token, is_admin, created_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.UploadTokenName, &u.UploadToken, &u.IsAdmin, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

func (d *DB) ListUsers(ctx context.Context) ([]*User, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT id, username, upload_token_name, is_admin, created_at FROM users ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	var users []*User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.UploadTokenName, &u.IsAdmin, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

func (d *DB) UpdatePassword(ctx context.Context, userID int64, passwordHash string) error {
	_, err := d.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1 WHERE id = $2`,
		passwordHash, userID,
	)
	return err
}

func (d *DB) DeleteUser(ctx context.Context, id int64) (string, error) {
	var tokenName string
	err := d.db.QueryRowContext(ctx,
		`DELETE FROM users WHERE id = $1 RETURNING upload_token_name`, id,
	).Scan(&tokenName)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("delete user: %w", err)
	}
	return tokenName, nil
}

func (d *DB) CreateSession(ctx context.Context, userID int64, expiresAt time.Time) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	id := hex.EncodeToString(b)
	_, err := d.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`,
		id, userID, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return id, nil
}

func (d *DB) GetSessionUser(ctx context.Context, sessionID string) (*User, error) {
	var u User
	err := d.db.QueryRowContext(ctx,
		`SELECT u.id, u.username, u.password_hash, u.upload_token_name, u.upload_token, u.is_admin, u.created_at
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.id = $1 AND s.expires_at > NOW()`,
		sessionID,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.UploadTokenName, &u.UploadToken, &u.IsAdmin, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session user: %w", err)
	}
	return &u, nil
}

func (d *DB) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := d.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
	return err
}

func (d *DB) DeleteExpiredSessions(ctx context.Context) error {
	_, err := d.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= NOW()`)
	return err
}
