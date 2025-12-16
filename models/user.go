package models

import "time"

type User struct {
	ID            string    `json:"id" gorm:"primaryKey"`
	Name          string    `json:"name"`
	Email         string    `json:"email" gorm:"uniqueIndex"`
	EmailVerified bool      `json:"email_verified"`
	Image         *string   `json:"image,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
