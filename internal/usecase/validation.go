package usecase

import "unicode"

// ValidateOrderNumber checks order number using Luhn algorithm.
func ValidateOrderNumber(number string) bool {
	if number == "" {
		return false
	}

	var sum int
	var alt bool
	for i := len(number) - 1; i >= 0; i-- {
		r := rune(number[i])
		if !unicode.IsDigit(r) {
			return false
		}
		digit := int(r - '0')
		if alt {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		alt = !alt
	}

	return sum%10 == 0
}
