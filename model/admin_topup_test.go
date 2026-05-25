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

func TestLegacyAdminTopUpPaymentMethodIsTreatedAsRecharge(t *testing.T) {
	truncateTables(t)

	insertRankingUser(t, 4204, "legacy-admin-add-user")
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 4204).Update("quota", 1500).Error)
	legacyTopUp := &TopUp{
		UserId:        4204,
		Amount:        1500,
		TradeNo:       "LEGACY_ADMIN_ADD",
		PaymentMethod: PaymentMethodAdminAddLegacy,
		CreateTime:    1700,
		CompleteTime:  1700,
		Status:        common.TopUpStatusSuccess,
	}
	require.NoError(t, DB.Create(legacyTopUp).Error)

	rows, total, err := GetAllTopUpsWithOptions(TopUpQueryOptions{PaymentMethod: PaymentMethodAdminAdd}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "LEGACY_ADMIN_ADD", rows[0].TradeNo)
	assert.True(t, IsAdminTopUpRecord(rows[0]))

	refund, err := RefundAdminBalanceTopUp("LEGACY_ADMIN_ADD", 500, "legacy refund")
	require.NoError(t, err)
	assert.Equal(t, int64(500), refund.RefundQuota)
	assert.Equal(t, PaymentMethodAdminAddLegacy, refund.PaymentMethod)
	assert.Equal(t, 1000, getUserQuotaForPaymentGuardTest(t, 4204))
}

func TestMigrateLegacyAdminTopUpsToRechargeNormalizesPaymentFields(t *testing.T) {
	truncateTables(t)

	insertRankingUser(t, 4205, "legacy-admin-migrate-user")
	require.NoError(t, DB.Create(&TopUp{
		UserId:        4205,
		Amount:        900,
		TradeNo:       "LEGACY_ADMIN_MIGRATE",
		PaymentMethod: PaymentMethodAdminAddLegacy,
		CreateTime:    1800,
		CompleteTime:  1800,
		Status:        common.TopUpStatusSuccess,
	}).Error)
	require.NoError(t, DB.Create(&TopUpRefund{
		UserId:        4205,
		TradeNo:       "LEGACY_ADMIN_MIGRATE",
		OutRequestNo:  "LEGACY_ADMIN_REFUND",
		PaymentMethod: PaymentMethodAdminAddLegacy,
		Status:        TopUpRefundStatusSuccess,
	}).Error)
	require.NoError(t, DB.Create(&TopUpRefundRequest{
		UserId:        4205,
		TradeNo:       "LEGACY_ADMIN_MIGRATE",
		PaymentMethod: PaymentMethodAdminAddLegacy,
		Status:        TopUpRefundRequestStatusPending,
	}).Error)

	require.NoError(t, migrateLegacyAdminTopUpsToRecharge(DB))

	reloaded := GetTopUpByTradeNo("LEGACY_ADMIN_MIGRATE")
	require.NotNil(t, reloaded)
	assert.Equal(t, PaymentMethodAdminAdd, reloaded.PaymentMethod)
	assert.Equal(t, PaymentProviderAdmin, reloaded.PaymentProvider)

	var refund TopUpRefund
	require.NoError(t, DB.Where("trade_no = ?", "LEGACY_ADMIN_MIGRATE").First(&refund).Error)
	assert.Equal(t, PaymentMethodAdminAdd, refund.PaymentMethod)
	assert.Equal(t, PaymentProviderAdmin, refund.PaymentProvider)

	var request TopUpRefundRequest
	require.NoError(t, DB.Where("trade_no = ?", "LEGACY_ADMIN_MIGRATE").First(&request).Error)
	assert.Equal(t, PaymentMethodAdminAdd, request.PaymentMethod)
	assert.Equal(t, PaymentProviderAdmin, request.PaymentProvider)
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
