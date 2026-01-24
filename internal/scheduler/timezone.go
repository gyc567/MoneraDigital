package scheduler

import (
	"sync"
	"time"

	"monera-digital/internal/logger"
)

// Shanghai timezone: UTC+8 (China does not observe DST)
const shanghaiOffsetSeconds = 8 * 60 * 60

var (
	shanghaiLocation     *time.Location
	shanghaiLocationOnce sync.Once
)

// GetShanghaiLocation returns Asia/Shanghai timezone location.
// Falls back to a fixed UTC+8 offset if timezone database is unavailable.
// This is critical for deployment environments that lack tzdata.
func GetShanghaiLocation() *time.Location {
	shanghaiLocationOnce.Do(func() {
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			logger.Warn("[Timezone] Failed to load Asia/Shanghai timezone, using UTC+8 fallback",
				"error", err.Error())
			loc = time.FixedZone("Asia/Shanghai", shanghaiOffsetSeconds)
		}
		shanghaiLocation = loc
	})
	return shanghaiLocation
}

// NowInShanghai returns the current time in Asia/Shanghai timezone.
func NowInShanghai() time.Time {
	return time.Now().In(GetShanghaiLocation())
}

// TodayInShanghai returns today's date string (YYYY-MM-DD) in Asia/Shanghai timezone.
func TodayInShanghai() string {
	return NowInShanghai().Format("2006-01-02")
}

// resetShanghaiLocation resets the cached location for testing.
// This is unexported intentionally - only used in tests.
func resetShanghaiLocation() {
	shanghaiLocationOnce = sync.Once{}
	shanghaiLocation = nil
}
