package utils

import (
	"math/rand"
	"time"
)

// RandomString generates a random string of the specified length.
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// RandomInt generates a random integer between min (inclusive) and max (exclusive).
func RandomInt(min, max int) int {
	return min + rand.Intn(max-min)
}

// TimeNow returns the current time.
func TimeNow() time.Time {
	return time.Now()
}

// TimeSince returns the time elapsed since t.
func TimeSince(t time.Time) time.Duration {
	return time.Since(t)
}

// TimeUntil returns the time until t.
func TimeUntil(t time.Time) time.Duration {
	return time.Until(t)
}

// FormatDuration formats a duration as a string.
func FormatDuration(d time.Duration) string {
	return d.String()
}

// ParseDuration parses a duration string.
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

func init() {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())
}
