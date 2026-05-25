package model

import (
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAdminBalanceTopUpCreatesBillAndCreditsQuota(t *testing.T) {
	truncateTables(t)

	insertRankingUser(t, 4201, "admin-recharge-user")

	topUp, err := CreateAdminBalanceTopUp(4201, 1200)
	require.NoError(t, err)
	require.NotNil(t, topUp)
	assert.Equal(t, int64(1200), topUp.Amount)
	assert.Equal(t, PaymentMethodAdminAdd, topUp.PaymentMethod)
	assert.Equal(t, PaymentProviderAdmin, topUp.PaymentProvider)
	assert.Equal(t, common.TopUpStatusSuccess, topUp.Status)
	assert.NotZero(t, topUp.CompleteTime)
	assert.Equal(t, 1200, getUserQuotaForPaymentGuardTest(t, 4201))

	rows, total, err := GetAllTopUpsWithOptions(TopUpQueryOptions{PaymentMethod: PaymentMethodAdminAdd}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, topUp.TradeNo, rows[0].TradeNo)
}

func TestRefundAdminBalanceTopUpRecordsRefundAndDeductsQuota(t *testing.T) {
	truncateTables(t)

	insertRankingUser(t, 4202, "admin-refund-user")
	topUp, err := CreateAdminBalanceTopUp(4202, 1000)
	require.NoError(t, err)

	refund, err := RefundAdminBalanceTopUp(topUp.TradeNo, 300, "manual refund")
	require.NoError(t, err)
	require.NotNil(t, refund)
	assert.Equal(t, TopUpRefundStatusSuccess, refund.Status)
	assert.Equal(t, int64(300), refund.RefundQuota)
	assert.Equal(t, "manual refund", refund.Reason)
	assert.Equal(t, 700, getUserQuotaForPaymentGuardTest(t, 4202))

	reloaded := GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, reloaded)
	assert.Equal(t, common.TopUpStatusPartialRefunded, reloaded.Status)
	assert.Equal(t, int64(300), reloaded.RefundedQuota)

	refund, err = RefundAdminBalanceTopUp(topUp.TradeNo, 700, "full refund")
	require.NoError(t, err)
	assert.Equal(t, int64(700), refund.RefundQuota)
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 4202))

	reloaded = GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, reloaded)
	assert.Equal(t, common.TopUpStatusRefunded, reloaded.Status)
	assert.Equal(t, int64(1000), reloaded.RefundedQuota)
}

func TestRefundAdminBalanceTopUpRejectsInsufficientUserQuota(t *testing.T) {
	truncateTables(t)

	insertRankingUser(t, 4203, "admin-refund-low-balance-user")
	topUp, err := CreateAdminBalanceTopUp(4203, 1000)
	require.NoError(t, err)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 4203).Update("quota", 100).Error)

	_, err = RefundAdminBalanceTopUp(topUp.TradeNo, 500, "too much")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "余额不足")
	assert.Equal(t, 100, getUserQuotaForPaymentGuardTest(t, 4203))

	reloaded := GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, reloaded)
	assert.Equal(t, common.TopUpStatusSuccess, reloaded.Status)
	assert.Equal(t, int64(0), reloaded.RefundedQuota)
}
