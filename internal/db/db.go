package db

import (
"context"
"database/sql"
"fmt"
"time"

_ "github.com/go-sql-driver/mysql"
"github.com/google/uuid"
)

type DB struct {
conn *sql.DB
}

func New(dsn string) (*DB, error) {
conn, err := sql.Open("mysql", dsn)
if err != nil {
return nil, fmt.Errorf("failed to open database: %w", err)
}
conn.SetMaxOpenConns(10)
conn.SetMaxIdleConns(5)
conn.SetConnMaxLifetime(5 * time.Minute)
if err := conn.Ping(); err != nil {
return nil, fmt.Errorf("failed to ping database: %w", err)
}
return &DB{conn: conn}, nil
}

func (d *DB) Close() {
_ = d.conn.Close()
}

type User struct {
ID        string
Email     string
CreatedAt time.Time
}

func (d *DB) FindOrCreateUser(ctx context.Context, email string) (*User, error) {
id := uuid.New().String()
_, err := d.conn.ExecContext(ctx, `
INSERT INTO users (id, email)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE email = email
`, id, email)
if err != nil {
return nil, err
}
return d.FindUserByEmail(ctx, email)
}

func (d *DB) FindUserByEmail(ctx context.Context, email string) (*User, error) {
u := &User{}
err := d.conn.QueryRowContext(ctx, `
SELECT id, email, created_at FROM users WHERE email = ?
`, email).Scan(&u.ID, &u.Email, &u.CreatedAt)
if err != nil {
return nil, err
}
return u, nil
}

type Device struct {
ID         string
UserID     string
TokenHash  string
DeviceName *string
CreatedAt  time.Time
LastSeen   *time.Time
Revoked    bool
}

func (d *DB) CreateDevice(ctx context.Context, userID, tokenHash string, deviceName *string) (*Device, error) {
id := uuid.New().String()
_, err := d.conn.ExecContext(ctx, `
INSERT INTO devices (id, user_id, token_hash, device_name)
VALUES (?, ?, ?, ?)
`, id, userID, tokenHash, deviceName)
if err != nil {
return nil, err
}
return &Device{
ID:         id,
UserID:     userID,
TokenHash:  tokenHash,
DeviceName: deviceName,
CreatedAt:  time.Now(),
}, nil
}

func (d *DB) FindDeviceByTokenHash(ctx context.Context, tokenHash string) (*Device, error) {
dev := &Device{}
err := d.conn.QueryRowContext(ctx, `
SELECT id, user_id, token_hash, device_name, created_at, last_seen, revoked
FROM devices
WHERE token_hash = ? AND revoked = 0
`, tokenHash).Scan(
&dev.ID, &dev.UserID, &dev.TokenHash,
&dev.DeviceName, &dev.CreatedAt, &dev.LastSeen, &dev.Revoked,
)
if err != nil {
return nil, err
}
return dev, nil
}

func (d *DB) UpdateLastSeen(ctx context.Context, tokenHash string) error {
_, err := d.conn.ExecContext(ctx, `
UPDATE devices SET last_seen = NOW() WHERE token_hash = ?
`, tokenHash)
return err
}

func (d *DB) RevokeDevice(ctx context.Context, tokenHash string) error {
_, err := d.conn.ExecContext(ctx, `
UPDATE devices SET revoked = 1 WHERE token_hash = ?
`, tokenHash)
return err
}

func (d *DB) CleanupInactiveDevices(ctx context.Context) error {
_, err := d.conn.ExecContext(ctx, `
DELETE FROM devices
WHERE (last_seen IS NULL AND created_at < NOW() - INTERVAL 90 DAY)
   OR last_seen < NOW() - INTERVAL 90 DAY
`)
return err
}
