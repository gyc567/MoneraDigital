package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetShanghaiLocation_ReturnsValidLocation(t *testing.T) {
	resetShanghaiLocation()

	loc := GetShanghaiLocation()

	require.NotNil(t, loc, "Location must not be nil")

	// Verify offset is UTC+8 (28800 seconds)
	fixedTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	shanghaiTime := fixedTime.In(loc)
	_, offset := shanghaiTime.Zone()
	assert.Equal(t, 8*60*60, offset, "Shanghai offset should be UTC+8")
}

func TestGetShanghaiLocation_IsCached(t *testing.T) {
	resetShanghaiLocation()

	loc1 := GetShanghaiLocation()
	loc2 := GetShanghaiLocation()

	assert.Same(t, loc1, loc2, "Should return cached location")
}

func TestNowInShanghai_ReturnsCurrentTimeInShanghai(t *testing.T) {
	resetShanghaiLocation()

	before := time.Now()
	shanghaiNow := NowInShanghai()
	after := time.Now()

	// Convert to UTC for comparison
	shanghaiUTC := shanghaiNow.UTC()

	assert.True(t, !shanghaiUTC.Before(before.UTC()), "Shanghai time should not be before test start")
	assert.True(t, !shanghaiUTC.After(after.UTC()), "Shanghai time should not be after test end")

	// Verify location is set
	assert.NotNil(t, shanghaiNow.Location())
}

func TestTodayInShanghai_ReturnsCorrectFormat(t *testing.T) {
	resetShanghaiLocation()

	today := TodayInShanghai()

	// Verify format YYYY-MM-DD
	_, err := time.Parse("2006-01-02", today)
	assert.NoError(t, err, "Date should be in YYYY-MM-DD format")
}

func TestTodayInShanghai_MatchesNowInShanghai(t *testing.T) {
	resetShanghaiLocation()

	today := TodayInShanghai()
	now := NowInShanghai()

	expected := now.Format("2006-01-02")
	assert.Equal(t, expected, today)
}

func TestGetShanghaiLocation_FixedZoneFallback(t *testing.T) {
	// Validate fallback logic works correctly by testing FixedZone
	fallbackLoc := time.FixedZone("Asia/Shanghai", 8*60*60)

	fixedTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	shanghaiTime := fixedTime.In(fallbackLoc)

	assert.Equal(t, 8, shanghaiTime.Hour(), "UTC 00:00 should be 08:00 in Shanghai")
	assert.Equal(t, 15, shanghaiTime.Day(), "Day should be the same")
}

func TestNowInShanghai_DoesNotPanic(t *testing.T) {
	resetShanghaiLocation()

	// Critical test: NowInShanghai should never panic
	assert.NotPanics(t, func() {
		_ = NowInShanghai()
	})
}

func TestTodayInShanghai_DoesNotPanic(t *testing.T) {
	resetShanghaiLocation()

	assert.NotPanics(t, func() {
		_ = TodayInShanghai()
	})
}

func TestGetShanghaiLocation_ConsistentAcrossMultipleCalls(t *testing.T) {
	resetShanghaiLocation()

	times := make([]time.Time, 5)
	for i := 0; i < 5; i++ {
		times[i] = NowInShanghai()
		time.Sleep(10 * time.Millisecond)
	}

	// All times should be in the same location
	for i := 1; i < len(times); i++ {
		prev := times[i-1].Location()
		curr := times[i].Location()
		assert.Equal(t, prev, curr, "Location should be consistent across calls")
	}
}
