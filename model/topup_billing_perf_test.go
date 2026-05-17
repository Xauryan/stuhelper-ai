package model

import (
	"testing"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetUserTopUpsWithOptionsDoesNotFillUsername 锁定 self 路径行为：
// 普通用户查自己的账单时不再回填 Username（前端不展示该列，省一次 SELECT）。
// 若以后 self API 被导出/邮件等功能复用，本测试会暴露契约变化。
func TestGetUserTopUpsWithOptionsDoesNotFillUsername(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       4101,
		Username: "self-path-user",
		Status:   common.UserStatusEnabled,
		AffCode:  "self-path-user-aff",
	}).Error)
	require.NoError(t, DB.Create(&TopUp{
		UserId:          4101,
		Amount:          1,
		Money:           1,
		TradeNo:         "TOPUP_SELF_NO_USERNAME",
		PaymentMethod:   PaymentMethodAlipayOfficial,
		PaymentProvider: PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}).Error)

	rows, total, err := GetUserTopUpsWithOptions(4101, TopUpQueryOptions{}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Empty(t, rows[0].Username, "self path should not fill Username to avoid extra users-table query")
}

// TestGetAllTopUpsWithOptionsStillFillsUsername 锁定 admin 路径行为：
// 全平台账单仍然回填 Username（管理员视图要展示用户名列）。
func TestGetAllTopUpsWithOptionsStillFillsUsername(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       4102,
		Username: "admin-path-user",
		Status:   common.UserStatusEnabled,
		AffCode:  "admin-path-user-aff",
	}).Error)
	require.NoError(t, DB.Create(&TopUp{
		UserId:          4102,
		Amount:          1,
		Money:           1,
		TradeNo:         "TOPUP_ADMIN_HAS_USERNAME",
		PaymentMethod:   PaymentMethodAlipayOfficial,
		PaymentProvider: PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}).Error)

	rows, total, err := GetAllTopUpsWithOptions(TopUpQueryOptions{}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "admin-path-user", rows[0].Username, "admin path must still fill Username for the username column")
}

// TestTopUpCompositeIndexExists 确认 AutoMigrate 产出了 (user_id, create_time)
// 复合索引；这是账单页性能优化的核心索引，迁移路径异常时需要立即暴露。
func TestTopUpCompositeIndexExists(t *testing.T) {
	assert.True(t,
		DB.Migrator().HasIndex(&TopUp{}, "idx_topup_user_create"),
		"composite index idx_topup_user_create on (user_id, create_time) must exist after AutoMigrate",
	)
}
