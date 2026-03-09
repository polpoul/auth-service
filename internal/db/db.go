package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*DB, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &DB{Pool: pool}, nil
}

// --- Users ---

type User struct {
	ID        uuid.UUID
	Email     string
	CreatedAt time.Time
}

// FindOrCreateUser returns existing user or creates one.
func (d *DB) FindOrCreateUser(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := d.Pool.QueryRow(ctx, `
		INSERT INTO users (id, email)
		VALUES (gen_random_uuid(), $1)
		ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		RETURNING id, email, created_at
	`, email).Scan(&u.ID, &u.Email, &u.CreatedAt)
	return u, err
}

func (d *DB) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := d.Pool.QueryRow(ctx, `
		SELECT id, email, created_at FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.CreatedAt)
	return u, err
}

// --- Devices ---

type Device struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  string
	DeviceName *string
	CreatedAt  time.Time
	LastSeen   *time.Time
	Revoked    bool
}

func (d *DB) CreateDevice(ctx context.Context, userID uuid.UUID, tokenHash string, deviceName *string) (*Device, error) {
	dev := &Device{}
	err := d.Pool.QueryRow(ctx, `
		INSERT INTO devices (id, user_id, token_hash, device_name)
		VALUES (gen_random_uuid(), $1, $2, $3)
		RETURNING id, user_id, token_hash, device_name, created_at, last_seen, revoked
	`, userID, tokenHash, deviceName).Scan(
		&dev.ID, &dev.UserID, &dev.TokenHash,
		&dev.DeviceName, &dev.CreatedAt, &dev.LastSeen, &dev.Revoked,
	)
	return dev, err
}

func (d *DB) FindDeviceByTokenHash(ctx context.Context, tokenHash string) (*Device, error) {
	dev := &Device{}
	err := d.Pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, device_name, created_at, last_seen, revoked
		FROM devices
		WHERE token_hash = $1 AND revoked = false
	`, tokenHash).Scan(
		&dev.ID, &dev.UserID, &dev.TokenHash,
		&dev.DeviceName, &dev.CreatedAt, &dev.LastSeen, &dev.Revoked,
	)
	return dev, err
}

func (d *DB) UpdateLastSeen(ctx context.Context, tokenHash string) error {
	_, err := d.Pool.Exec(ctx, `
		UPDATE devices SET last_seen = now() WHERE token_hash = $1
	`, tokenHash)
	return err
}

func (d *DB) RevokeDevice(ctx context.Context, tokenHash string) error {
	_, err := d.Pool.Exec(ctx, `
		UPDATE devices SET revoked = true WHERE token_hash = $1
	`, tokenHash)
	return err
}

// CleanupInactiveDevices removes devices inactive for more than 90 days.
func (d *DB) CleanupInactiveDevices(ctx context.Context) error {
	_, err := d.Pool.Exec(ctx, `
		DELETE FROM devices
		WHERE (last_seen IS NULL AND created_at < now() - INTERVAL '90 days')
		   OR last_seen < now() - INTERVAL '90 days'
	`)
	return err
}
