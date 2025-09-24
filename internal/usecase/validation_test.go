package usecase

import "testing"

func TestValidateOrderNumber(t *testing.T) {
	valid := []string{
		"79927398713",
		"2718281828459045",
		"6011111111111117",
	}
	for _, number := range valid {
		if !ValidateOrderNumber(number) {
			t.Fatalf("expected number %s to be valid", number)
		}
	}

	invalid := []string{"", "123456", "abcdef", "79927398710"}
	for _, number := range invalid {
		if ValidateOrderNumber(number) {
			t.Fatalf("expected number %s to be invalid", number)
		}
	}
}
