package repository

import (
	"time"

	"github.com/uptrace/bun"
)

type TOTPRecord struct {
	bun.BaseModel `bun:"table:totp"`
	ID            string    `bun:"column:id,pk"`
	UserID        string    `bun:"column:user_id"`
	Secret        string    `bun:"column:secret"`
	BackupCodes   string    `bun:"column:backup_codes"`
	Enabled       bool      `bun:"column:enabled"`
	CreatedAt     time.Time `bun:"column:created_at,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"column:updated_at,default:current_timestamp"`
}

type TrustedDevice struct {
	bun.BaseModel `bun:"table:trusted_devices"`
	ID            string    `bun:"column:id,pk"`
	UserID        string    `bun:"column:user_id"`
	Token         string    `bun:"column:token"`
	UserAgent     string    `bun:"column:user_agent"`
	ExpiresAt     time.Time `bun:"column:expires_at"`
	CreatedAt     time.Time `bun:"column:created_at,default:current_timestamp"`
}
