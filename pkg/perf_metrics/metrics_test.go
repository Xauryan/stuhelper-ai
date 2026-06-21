package perfmetrics

import (
	"testing"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRecentSuccessRatesReturnsLatestBuckets(t *testing.T) {
	rates := recentSuccessRates(map[int64]counters{
		100: {requestCount: 10, successCount: 10},
		200: {requestCount: 4, successCount: 3},
		300: {requestCount: 3, successCount: 1},
		400: {requestCount: 2, successCount: 1},
	}, 3)

	require.Equal(t, []float64{75, 33.33, 50}, rates)
}

func TestQuerySummaryAllIncludesRecentSuccessRates(t *testing.T) {
	setupPerfMetricTestDB(t)
	resetHotBuckets(t)
	baseBucket := bucketStart(time.Now().Unix()) - 3*3600

	require.NoError(t, model.DB.Create([]model.PerfMetric{
		{ModelName: "gpt-test", Group: "default", BucketTs: baseBucket, RequestCount: 4, SuccessCount: 4, TotalLatencyMs: 400, OutputTokens: 100, GenerationMs: 1000},
		{ModelName: "gpt-test", Group: "vip", BucketTs: baseBucket + 3600, RequestCount: 4, SuccessCount: 3, TotalLatencyMs: 800, OutputTokens: 200, GenerationMs: 1000},
		{ModelName: "gpt-test", Group: "default", BucketTs: baseBucket + 7200, RequestCount: 3, SuccessCount: 1, TotalLatencyMs: 900, OutputTokens: 300, GenerationMs: 1000},
	}).Error)

	bucket := &atomicBucket{}
	bucket.add(Sample{Model: "gpt-test", Group: "default", LatencyMs: 100, Success: true, OutputTokens: 100, GenerationMs: 1000})
	bucket.add(Sample{Model: "gpt-test", Group: "default", LatencyMs: 100, Success: false, OutputTokens: 100, GenerationMs: 1000})
	hotBuckets.Store(bucketKey{model: "gpt-test", group: "default", bucketTs: baseBucket + 10800}, bucket)

	result, err := QuerySummaryAll(24*30, nil)
	require.NoError(t, err)
	require.Len(t, result.Models, 1)
	require.Equal(t, "gpt-test", result.Models[0].ModelName)
	require.Equal(t, int64(13), result.Models[0].RequestCount)
	require.Equal(t, float64(69.23), result.Models[0].SuccessRate)
	require.Equal(t, []float64{75, 33.33, 50}, result.Models[0].RecentSuccessRates)
}

func setupPerfMetricTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	previousDB := model.DB
	previousUsingSQLite := common.UsingSQLite
	previousRedisEnabled := common.RedisEnabled

	model.DB = db
	common.UsingSQLite = true
	common.RedisEnabled = false
	require.NoError(t, db.AutoMigrate(&model.PerfMetric{}))

	t.Cleanup(func() {
		model.DB = previousDB
		common.UsingSQLite = previousUsingSQLite
		common.RedisEnabled = previousRedisEnabled
		_ = sqlDB.Close()
	})
}

func resetHotBuckets(t *testing.T) {
	t.Helper()
	hotBuckets.Range(func(key, _ any) bool {
		hotBuckets.Delete(key)
		return true
	})
	t.Cleanup(func() {
		hotBuckets.Range(func(key, _ any) bool {
			hotBuckets.Delete(key)
			return true
		})
	})
}
