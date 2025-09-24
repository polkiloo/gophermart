package model

import "testing"

func TestOrderStatusValues(t *testing.T) {
	cases := []struct {
		name  string
		got   OrderStatus
		value string
	}{
		{"new", OrderStatusNew, "NEW"},
		{"processing", OrderStatusProcessing, "PROCESSING"},
		{"invalid", OrderStatusInvalid, "INVALID"},
		{"processed", OrderStatusProcessed, "PROCESSED"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.got) != tc.value {
				t.Fatalf("expected %s, got %s", tc.value, tc.got)
			}
		})
	}
}

func TestAccrualStatusValues(t *testing.T) {
	cases := []struct {
		status AccrualStatus
		value  string
	}{
		{AccrualStatusRegistered, "REGISTERED"},
		{AccrualStatusInvalid, "INVALID"},
		{AccrualStatusProcessing, "PROCESSING"},
		{AccrualStatusProcessed, "PROCESSED"},
	}

	for _, tc := range cases {
		if string(tc.status) != tc.value {
			t.Fatalf("expected %s, got %s", tc.value, tc.status)
		}
	}
}
