package model

import (
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertRankingUser(t *testing.T, id int, username string) {
	t.Helper()
	require.NoError(t, DB.Create(&User{
		Id:       id,
		Username: username,
		Status:   common.UserStatusEnabled,
		AffCode:  "ranking-aff-" + username,
	}).Error)
}

func TestGetUserConsumptionRankingTotals(t *testing.T) {
	truncateTables(t)

	insertRankingUser(t, 1, "alice")
	insertRankingUser(t, 2, "bob")

	require.NoError(t, LOG_DB.Create(&[]Log{
		{UserId: 1, Username: "alice", CreatedAt: 1100, Type: LogTypeConsume, Quota: 100},
		{UserId: 1, Username: "alice-renamed", CreatedAt: 1200, Type: LogTypeConsume, Quota: 300},
		{UserId: 2, Username: "bob", CreatedAt: 1300, Type: LogTypeConsume, Quota: 500},
		{UserId: 2, Username: "bob", CreatedAt: 1400, Type: LogTypeError, Quota: 900},
		{UserId: 1, Username: "alice", CreatedAt: 900, Type: LogTypeConsume, Quota: 700},
	}).Error)

	rows, total, err := GetUserConsumptionRankingTotals(1000, 2000, 20)
	require.NoError(t, err)

	require.Len(t, rows, 2)
	assert.Equal(t, int64(900), total)
	assert.Equal(t, 2, rows[0].UserId)
	assert.Equal(t, "bob", rows[0].Username)
	assert.Equal(t, int64(500), rows[0].TotalQuota)
	assert.Equal(t, 1, rows[1].UserId)
	assert.Equal(t, "alice", rows[1].Username)
	assert.Equal(t, int64(400), rows[1].TotalQuota)

	limitedRows, limitedTotal, err := GetUserConsumptionRankingTotals(1000, 2000, 1)
	require.NoError(t, err)
	require.Len(t, limitedRows, 1)
	assert.Equal(t, int64(900), limitedTotal)
}

func TestGetUserRechargeRankingTotalsIncludesAllRechargeSources(t *testing.T) {
	truncateTables(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 500000

	insertRankingUser(t, 1, "alice")
	insertRankingUser(t, 2, "bob")
	insertRankingUser(t, 3, "carol")

	require.NoError(t, DB.Create(&[]TopUp{
		{
			UserId:          1,
			Amount:          2,
			Money:           2,
			TradeNo:         "epay-success",
			PaymentMethod:   "alipay",
			PaymentProvider: PaymentProviderEpay,
			CreateTime:      1100,
			CompleteTime:    1200,
			Status:          common.TopUpStatusSuccess,
		},
		{
			UserId:          2,
			Amount:          700000,
			Money:           8,
			TradeNo:         "creem-success",
			PaymentMethod:   PaymentMethodCreem,
			PaymentProvider: PaymentProviderCreem,
			CreateTime:      1200,
			CompleteTime:    1300,
			Status:          common.TopUpStatusSuccess,
		},
		{
			UserId:          3,
			Amount:          2,
			Money:           2,
			RefundedQuota:   500000,
			TradeNo:         "partial-refunded-topup",
			PaymentMethod:   "alipay",
			PaymentProvider: PaymentProviderEpay,
			CreateTime:      1300,
			CompleteTime:    1400,
			Status:          common.TopUpStatusPartialRefunded,
		},
		{
			UserId:          3,
			Amount:          99,
			Money:           99,
			TradeNo:         "failed-topup",
			PaymentMethod:   "alipay",
			PaymentProvider: PaymentProviderEpay,
			CreateTime:      1300,
			CompleteTime:    1400,
			Status:          common.TopUpStatusFailed,
		},
		{
			UserId:          1,
			Amount:          99,
			Money:           99,
			RefundedQuota:   49500000,
			TradeNo:         "refunded-topup",
			PaymentMethod:   "alipay",
			PaymentProvider: PaymentProviderEpay,
			CreateTime:      1300,
			CompleteTime:    1400,
			Status:          common.TopUpStatusRefunded,
		},
	}).Error)

	require.NoError(t, DB.Create(&[]Redemption{
		{UsedUserId: 1, Key: "redeem-alice", Status: common.RedemptionCodeStatusUsed, Quota: 300000, RedeemedTime: 1500},
		{UsedUserId: 3, Key: "redeem-unused", Status: common.RedemptionCodeStatusEnabled, Quota: 900000, RedeemedTime: 1500},
	}).Error)

	require.NoError(t, LOG_DB.Create(&[]Log{
		{UserId: 2, Username: "bob", CreatedAt: 1600, Type: LogTypeManage, Quota: 400000, Content: "管理员增加用户额度"},
		{UserId: 3, Username: "carol", CreatedAt: 1700, Type: LogTypeManage, Content: "管理员增加用户额度 ＄1.600000 额度"},
		{UserId: 1, Username: "alice", CreatedAt: 1600, Type: LogTypeManage, Quota: 250000, Content: "管理员减少用户额度"},
		{UserId: 3, Username: "carol", CreatedAt: 900, Type: LogTypeManage, Quota: 800000, Content: "管理员增加用户额度"},
	}).Error)

	rows, total, err := GetUserRechargeRankingTotals(1000, 2000, 20)
	require.NoError(t, err)

	require.Len(t, rows, 3)
	assert.Equal(t, int64(3700000), total)
	assert.Equal(t, 1, rows[0].UserId)
	assert.Equal(t, "alice", rows[0].Username)
	assert.Equal(t, int64(1300000), rows[0].TotalQuota)
	assert.Equal(t, 3, rows[1].UserId)
	assert.Equal(t, "carol", rows[1].Username)
	assert.Equal(t, int64(1300000), rows[1].TotalQuota)
	assert.Equal(t, 2, rows[2].UserId)
	assert.Equal(t, "bob", rows[2].Username)
	assert.Equal(t, int64(1100000), rows[2].TotalQuota)

	limitedRows, limitedTotal, err := GetUserRechargeRankingTotals(1000, 2000, 1)
	require.NoError(t, err)
	require.Len(t, limitedRows, 1)
	assert.Equal(t, int64(3700000), limitedTotal)
}
