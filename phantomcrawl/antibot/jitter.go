package antibot

import (
	"math/rand"
	"time"
)

// Jitter adds randomness to throttle so requests
// never have predictable timing patterns
func Jitter(seconds int) time.Duration {
	base := float64(seconds)
	// Random value between -1 and +1
	offset := (rand.Float64() * 2) - 1
	jittered := base + offset
	if jittered < 0.5 {
		jittered = 0.5
	}
	return time.Duration(jittered * float64(time.Second))
}