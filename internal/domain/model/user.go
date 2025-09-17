package model

import "time"

// User represents a registered customer of loyalty program.
type User struct {
	ID           int64
	Login        string
	PasswordHash string
	CreatedAt    time.Time
}
