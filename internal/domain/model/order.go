package model

import "time"

// OrderStatus describes processing lifecycle.
type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusInvalid    OrderStatus = "INVALID"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
)

// Order describes purchase order registered by user.
type Order struct {
	ID         int64
	UserID     int64
	Number     string
	Status     OrderStatus
	Accrual    *float64
	UploadedAt time.Time
	UpdatedAt  time.Time
}
