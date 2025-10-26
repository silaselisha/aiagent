package analytics

import (
	"sort"
	"time"

	"starseed/internal/model"
)

// HourlyEngagement aggregates events into per-hour buckets.
func HourlyEngagement(events []model.EngagementEvent) map[time.Time]map[string]int {
	buckets := make(map[time.Time]map[string]int)
	for _, e := range events {
		key := time.Date(e.Timestamp.Year(), e.Timestamp.Month(), e.Timestamp.Day(), e.Timestamp.Hour(), 0, 0, 0, time.UTC)
		if _, ok := buckets[key]; !ok {
			buckets[key] = make(map[string]int)
		}
		buckets[key][e.Type]++
	}
	return buckets
}

// SortedBucketKeys returns sorted hour keys.
func SortedBucketKeys(m map[time.Time]map[string]int) []time.Time {
	keys := make([]time.Time, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Before(keys[j]) })
	return keys
}
