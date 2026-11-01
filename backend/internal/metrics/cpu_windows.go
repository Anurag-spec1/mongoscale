//go:build windows

package metrics

import "time"

func processCPUTime() time.Duration {
	return 0
}
