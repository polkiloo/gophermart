package test

import (
	"math/rand"
	"sync"
	"time"
)

const asciiLetters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var (
	rngMu sync.Mutex
	rng   = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// RandomASCIIString returns a pseudo-random ASCII string within the provided bounds.
// When maxLen equals minLen the resulting string always has that exact length.
func RandomASCIIString(minLen, maxLen int) string {
	if minLen <= 0 {
		minLen = 1
	}
	if maxLen < minLen {
		maxLen = minLen
	}
	length := minLen
	if maxLen > minLen {
		length += int(randomIntn(maxLen - minLen + 1))
	}
	buf := make([]byte, length)
	for i := range buf {
		buf[i] = asciiLetters[randomIntn(len(asciiLetters))]
	}
	return string(buf)
}

func randomIntn(n int) int {
	rngMu.Lock()
	defer rngMu.Unlock()
	return rng.Intn(n)
}
