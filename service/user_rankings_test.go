package service

import (
	"fmt"
	"sync"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func resetUserRankingCache() {
	ClearUserRankingsCache()
}

func setupUserRankingsServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	originalDB := model.DB
	originalLogDB := model.LOG_DB

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Log{},
		&model.TopUp{},
		&model.Redemption{},
	))

	resetUserRankingCache()

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		resetUserRankingCache()
	})

	return db
}

func insertRankingTestUser(t *testing.T, db *gorm.DB, id int, username string) {
	t.Helper()
	require.NoError(t, db.Create(&model.User{
		Id:       id,
		Username: username,
		Status:   common.UserStatusEnabled,
		AffCode:  fmt.Sprintf("svc-ranking-%d", id),
	}).Error)
}

func insertConsumeLog(t *testing.T, db *gorm.DB, userId int, createdAt int64, quota int, prompt int, completion int) {
	t.Helper()
	require.NoError(t, db.Create(&model.Log{
		UserId:           userId,
		CreatedAt:        createdAt,
		Type:             model.LogTypeConsume,
		Quota:            quota,
		PromptTokens:     prompt,
		CompletionTokens: completion,
	}).Error)
}

func TestUserRankingsResponseDoesNotExposeAggregateTotalsOrShare(t *testing.T) {
	rows := []model.UserRankingTotal{
		{UserId: 1, Username: "alice", TotalQuota: 900, TotalTokens: 1500, RequestCount: 9},
		{UserId: 2, Username: "bob", TotalQuota: 100, TotalTokens: 200, RequestCount: 2},
	}

	ranked := buildRankedUsers(rows)

	require.Len(t, ranked, 2)
	assert.Equal(t, 1, ranked[0].Rank)
	assert.Equal(t, "al***ce", ranked[0].Display)
	assert.Equal(t, int64(1500), ranked[0].TotalTokens)
	assert.Equal(t, int64(9), ranked[0].RequestCount)

	data, err := common.Marshal(UserRankingsResponse{
		Period:      "week",
		Consumption: ranked,
		Recharge:    ranked,
		UpdatedAt:   1700000000,
	})
	require.NoError(t, err)
	body := string(data)
	assert.NotContains(t, body, "share")
	assert.NotContains(t, body, "consumption_total")
	assert.NotContains(t, body, "recharge_total")
	assert.NotContains(t, body, "user_id")
	assert.NotContains(t, body, "username")
	assert.NotContains(t, body, "alice")
	assert.NotContains(t, body, "bob")
	assert.Contains(t, body, "total_tokens")
	assert.Contains(t, body, "request_count")
	// Me is optional and absent here -> must be omitted by omitempty
	assert.NotContains(t, body, `"me"`)
}

func TestGetUserRankingsSnapshotReturnsThreeMetricsForAnonymous(t *testing.T) {
	db := setupUserRankingsServiceTestDB(t)
	insertRankingTestUser(t, db, 1, "alice")
	insertConsumeLog(t, db, 1, 1100, 100, 30, 70)

	resp, err := GetUserRankingsSnapshot("all", 0)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Nil(t, resp.Me)
	require.Len(t, resp.Consumption, 1)
	assert.Equal(t, int64(100), resp.Consumption[0].TotalQuota)
	assert.Equal(t, int64(100), resp.Consumption[0].TotalTokens)
	assert.Equal(t, int64(1), resp.Consumption[0].RequestCount)
}

func TestGetUserRankingsSnapshotIncludesMeInsideTopN(t *testing.T) {
	db := setupUserRankingsServiceTestDB(t)
	insertRankingTestUser(t, db, 1, "alice")
	insertRankingTestUser(t, db, 2, "bob")
	insertConsumeLog(t, db, 1, 1100, 100, 50, 50)
	insertConsumeLog(t, db, 2, 1200, 200, 100, 100)

	resp, err := GetUserRankingsSnapshot("all", 1)
	require.NoError(t, err)
	require.NotNil(t, resp.Me)
	// bob (id=2) has more tokens -> alice should be rank #2
	assert.Equal(t, 2, resp.Me.Rank)
	// Me display must NOT be masked
	assert.Equal(t, "alice", resp.Me.Display)
	assert.Equal(t, int64(100), resp.Me.TotalTokens)
	assert.Equal(t, int64(1), resp.Me.RequestCount)

	// top-N rows remain masked
	require.Len(t, resp.Consumption, 2)
	assert.Equal(t, "al***ce", resp.Consumption[1].Display)
}

func TestGetUserRankingsSnapshotIncludesMeOutsideTopN(t *testing.T) {
	db := setupUserRankingsServiceTestDB(t)
	for i := 1; i <= 21; i++ {
		insertRankingTestUser(t, db, i, fmt.Sprintf("user%02d", i))
		// Larger user_id => fewer tokens so user 21 ends up last.
		tokens := 22 - i
		insertConsumeLog(t, db, i, int64(1000+i), tokens*10, tokens, 0)
	}

	resp, err := GetUserRankingsSnapshot("all", 21)
	require.NoError(t, err)
	require.NotNil(t, resp.Me)
	assert.Equal(t, 21, resp.Me.Rank)
	assert.Equal(t, "user21", resp.Me.Display)
	assert.Len(t, resp.Consumption, userRankingLimit)
}

func TestUserRankingsCacheIsolatesMeBetweenViewers(t *testing.T) {
	db := setupUserRankingsServiceTestDB(t)
	insertRankingTestUser(t, db, 1, "alice")
	insertRankingTestUser(t, db, 2, "bob")
	insertConsumeLog(t, db, 1, 1100, 100, 100, 100)
	insertConsumeLog(t, db, 2, 1200, 200, 50, 50)

	respA, err := GetUserRankingsSnapshot("all", 1)
	require.NoError(t, err)
	require.NotNil(t, respA.Me)
	assert.Equal(t, "alice", respA.Me.Display)

	respB, err := GetUserRankingsSnapshot("all", 2)
	require.NoError(t, err)
	require.NotNil(t, respB.Me)
	assert.Equal(t, "bob", respB.Me.Display)

	// Confirm A's Me row was not mutated by B's query
	assert.Equal(t, "alice", respA.Me.Display)
	// Public rows are equal between the two viewers (same cached snapshot)
	require.Len(t, respA.Consumption, 2)
	require.Len(t, respB.Consumption, 2)
	for i := range respA.Consumption {
		assert.Equal(t, respA.Consumption[i].Display, respB.Consumption[i].Display)
	}
}

func TestUserRankingsCacheConcurrentReadsDoNotRace(t *testing.T) {
	db := setupUserRankingsServiceTestDB(t)
	insertRankingTestUser(t, db, 1, "alice")
	insertConsumeLog(t, db, 1, 1100, 100, 50, 50)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := GetUserRankingsSnapshot("all", 1)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}
