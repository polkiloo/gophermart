package errors

import (
	stdErrors "errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"already exists", ErrAlreadyExists},
		{"not found", ErrNotFound},
		{"invalid credentials", ErrInvalidCredentials},
		{"insufficient balance", ErrInsufficientBalance},
		{"invalid order", ErrInvalidOrderNumber},
		{"invalid amount", ErrInvalidAmount},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !stdErrors.Is(tc.err, tc.err) {
				t.Fatalf("expected error to match itself: %v", tc.err)
			}
		})
	}
}
