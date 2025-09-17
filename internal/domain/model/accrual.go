package model

// AccrualStatus describes loyalty accrual calculation status returned by external service.
type AccrualStatus string

const (
	AccrualStatusRegistered AccrualStatus = "REGISTERED"
	AccrualStatusInvalid    AccrualStatus = "INVALID"
	AccrualStatusProcessing AccrualStatus = "PROCESSING"
	AccrualStatusProcessed  AccrualStatus = "PROCESSED"
)

// Accrual encapsulates order accrual calculation details.
type Accrual struct {
	Order   string
	Status  AccrualStatus
	Accrual *float64
}
