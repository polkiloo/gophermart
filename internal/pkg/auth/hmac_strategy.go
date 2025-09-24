package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid auth token")

// HMACStrategy implements auth token creation/verification using HMAC signatures.
type HMACStrategy struct {
	secret []byte
	ttl    time.Duration
}

// NewHMACStrategy builds HMACStrategy with provided secret and options.
func NewHMACStrategy(secret string, opts Options) *HMACStrategy {
	ttl := opts.TTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &HMACStrategy{secret: []byte(secret), ttl: ttl}
}

// IssueToken generates signed auth token for the user.
func (s *HMACStrategy) IssueToken(userID int64) (string, error) {
	expires := time.Now().Add(s.ttl).Unix()
	payload := fmt.Sprintf("%d:%d", userID, expires)
	sig := s.sign(payload)
	token := fmt.Sprintf("%s:%s", payload, sig)
	return base64.StdEncoding.EncodeToString([]byte(token)), nil
}

// ParseToken validates token and returns encoded user ID.
func (s *HMACStrategy) ParseToken(token string) (int64, error) {
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return 0, ErrInvalidToken
	}

	parts := strings.Split(string(raw), ":")
	if len(parts) != 3 {
		return 0, ErrInvalidToken
	}

	payload := strings.Join(parts[:2], ":")
	expectedSig := s.sign(payload)
	if !hmac.Equal([]byte(expectedSig), []byte(parts[2])) {
		return 0, ErrInvalidToken
	}

	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, ErrInvalidToken
	}

	expires, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, ErrInvalidToken
	}

	if time.Unix(expires, 0).Before(time.Now()) {
		return 0, ErrInvalidToken
	}

	return userID, nil
}

func (s *HMACStrategy) Name() string {
	return "hmac"
}

func (s *HMACStrategy) sign(payload string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
