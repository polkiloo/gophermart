package auth

import "golang.org/x/crypto/bcrypt"

// PasswordHasher defines hashing strategy for credentials.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash string, password string) error
}

// BcryptHasher uses bcrypt to hash passwords.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher creates BcryptHasher with provided cost.
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

// Hash returns bcrypt hash for provided password.
func (h *BcryptHasher) Hash(password string) (string, error) {
	encoded, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// Compare checks password against stored hash.
func (h *BcryptHasher) Compare(hash string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
