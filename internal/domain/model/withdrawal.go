package model

import "time"

// Withdrawal represents a loyalty points withdrawal transaction.
type Withdrawal struct {
	ID          int64
	UserID      int64
	OrderNumber string
	Sum         float64
	ProcessedAt time.Time
}
