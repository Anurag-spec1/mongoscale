package loadtest

import (
	"sync/atomic"
	"time"
)

const (
	bucketsPerMillisecond  = 2
	maxTrackedMilliseconds = 30000
	bucketCount            = maxTrackedMilliseconds * bucketsPerMillisecond
)

type histogram struct {
	buckets [bucketCount]atomic.Int64
	count   atomic.Int64
}

func (h *histogram) record(duration time.Duration) {
	index := int(float64(duration.Microseconds()) / 1000 * bucketsPerMillisecond)
	if index >= bucketCount {
		index = bucketCount - 1
	}
	if index < 0 {
		index = 0
	}
	h.buckets[index].Add(1)
	h.count.Add(1)
}

func (h *histogram) percentile(fraction float64) float64 {
	total := h.count.Load()
	if total == 0 {
		return 0
	}
	target := int64(float64(total-1)*fraction) + 1
	var seen int64
	for index := 0; index < bucketCount; index++ {
		seen += h.buckets[index].Load()
		if seen >= target {
			return float64(index) / bucketsPerMillisecond
		}
	}
	return float64(maxTrackedMilliseconds)
}
