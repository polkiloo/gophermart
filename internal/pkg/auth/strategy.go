package auth

import "time"

type Strategy interface {
	IssueToken(userID int64) (string, error)
	ParseToken(token string) (int64, error)
	Name() string
}

type Options struct {
	TTL time.Duration
}
