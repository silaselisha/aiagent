package schedule

import (
	"time"
)

// NextWindow returns the next suitable interaction time avoiding quiet hours.
func NextWindow(now time.Time, quietHours []int) time.Time {
	isQuiet := func(h int) bool {
		for _, q := range quietHours {
			if q == h { return true }
		}
		return false
	}
	for i := 0; i < 48; i++ { // search up to 2 days ahead
		cand := now.Add(time.Duration(i) * time.Hour)
		if !isQuiet(cand.Hour()) {
			return cand
		}
	}
	return now.Add(15 * time.Minute)
}
