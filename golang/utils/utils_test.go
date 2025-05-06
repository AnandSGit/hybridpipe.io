package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRandomString(t *testing.T) {
	// Test generating random strings of different lengths
	lengths := []int{0, 1, 5, 10, 20, 50, 100}
	
	for _, length := range lengths {
		t.Run("Length_"+string(rune('0'+length)), func(t *testing.T) {
			// Generate a random string
			randomStr := RandomString(length)
			
			// Check that the string has the correct length
			assert.Equal(t, length, len(randomStr), "Random string length does not match requested length")
			
			// Generate another random string
			anotherRandomStr := RandomString(length)
			
			// Check that the strings are different (unless length is 0)
			if length > 0 {
				assert.NotEqual(t, randomStr, anotherRandomStr, "Random strings should be different")
			}
		})
	}
}

func TestRandomInt(t *testing.T) {
	// Test generating random integers within different ranges
	ranges := []struct {
		min int
		max int
	}{
		{0, 1},
		{0, 10},
		{-10, 10},
		{100, 200},
		{-200, -100},
	}
	
	for _, r := range ranges {
		t.Run("Range_"+string(rune('0'+r.min))+"_"+string(rune('0'+r.max)), func(t *testing.T) {
			// Generate a random integer
			randomInt := RandomInt(r.min, r.max)
			
			// Check that the integer is within the specified range
			assert.GreaterOrEqual(t, randomInt, r.min, "Random integer should be greater than or equal to min")
			assert.Less(t, randomInt, r.max, "Random integer should be less than max")
		})
	}
}

func TestTimeNow(t *testing.T) {
	// Get the current time
	now := TimeNow()
	
	// Check that the time is close to the current time
	assert.WithinDuration(t, time.Now(), now, 1*time.Second, "TimeNow should return a time close to the current time")
}

func TestTimeSince(t *testing.T) {
	// Get the current time
	now := time.Now()
	
	// Wait a short time
	time.Sleep(10 * time.Millisecond)
	
	// Get the time since now
	elapsed := TimeSince(now)
	
	// Check that the elapsed time is positive and reasonable
	assert.Greater(t, elapsed, time.Duration(0), "TimeSince should return a positive duration")
	assert.Less(t, elapsed, 1*time.Second, "TimeSince should return a reasonable duration")
}

func TestTimeUntil(t *testing.T) {
	// Get a future time
	future := time.Now().Add(1 * time.Hour)
	
	// Get the time until the future time
	remaining := TimeUntil(future)
	
	// Check that the remaining time is positive and reasonable
	assert.Greater(t, remaining, time.Duration(0), "TimeUntil should return a positive duration")
	assert.Less(t, remaining, 2*time.Hour, "TimeUntil should return a reasonable duration")
}

func TestFormatDuration(t *testing.T) {
	// Test formatting different durations
	durations := []struct {
		duration time.Duration
		expected string
	}{
		{1 * time.Nanosecond, "1ns"},
		{1 * time.Microsecond, "1µs"},
		{1 * time.Millisecond, "1ms"},
		{1 * time.Second, "1s"},
		{1 * time.Minute, "1m0s"},
		{1 * time.Hour, "1h0m0s"},
		{24 * time.Hour, "24h0m0s"},
		{1*time.Hour + 30*time.Minute + 15*time.Second, "1h30m15s"},
	}
	
	for _, d := range durations {
		t.Run(d.expected, func(t *testing.T) {
			// Format the duration
			formatted := FormatDuration(d.duration)
			
			// Check that the formatted string is correct
			assert.Equal(t, d.expected, formatted, "Formatted duration does not match expected string")
		})
	}
}

func TestParseDuration(t *testing.T) {
	// Test parsing different duration strings
	durations := []struct {
		str      string
		expected time.Duration
	}{
		{"1ns", 1 * time.Nanosecond},
		{"1µs", 1 * time.Microsecond},
		{"1ms", 1 * time.Millisecond},
		{"1s", 1 * time.Second},
		{"1m", 1 * time.Minute},
		{"1h", 1 * time.Hour},
		{"24h", 24 * time.Hour},
		{"1h30m15s", 1*time.Hour + 30*time.Minute + 15*time.Second},
	}
	
	for _, d := range durations {
		t.Run(d.str, func(t *testing.T) {
			// Parse the duration string
			parsed, err := ParseDuration(d.str)
			
			// Check that there was no error
			assert.NoError(t, err, "ParseDuration should not return an error")
			
			// Check that the parsed duration is correct
			assert.Equal(t, d.expected, parsed, "Parsed duration does not match expected duration")
		})
	}
	
	// Test parsing invalid duration strings
	invalidDurations := []string{
		"",
		"invalid",
		"1",
		"1x",
	}
	
	for _, str := range invalidDurations {
		t.Run("Invalid_"+str, func(t *testing.T) {
			// Parse the invalid duration string
			_, err := ParseDuration(str)
			
			// Check that there was an error
			assert.Error(t, err, "ParseDuration should return an error for invalid duration strings")
		})
	}
}
