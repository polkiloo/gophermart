package errors

import "errors"

var (
	ErrAlreadyExists       = errors.New("already exists")
	ErrNotFound            = errors.New("not found")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInvalidOrderNumber  = errors.New("invalid order number")
	ErrInvalidAmount       = errors.New("invalid amount")
)
