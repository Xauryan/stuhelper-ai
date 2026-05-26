package model

import (
	"fmt"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/logger"
	"github.com/Xauryan/stuhelper-ai/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setAdminTopUpMoneySettings(t *testing.T, unitPrice float64, serviceFeePercent float64) {
	t.Helper()
	originalUnitPrice := setting.AlipayOfficialUnitPrice
	originalServiceFeePercent := setting.AlipayOfficialServiceFeePercent
	setting.AlipayOfficialUnitPrice = unitPrice
	setting.AlipayOfficialServiceFeePercent = serviceFeePercent
	t.Cleanup(func() {
		setting.AlipayOfficialUnitPrice = originalUnitPrice
		setting.AlipayOfficialServiceFeePercent = originalServiceFeePercent
	})
}

func TestCreateAdminBalanceTopUpCreatesBillAndCreditsQuota(t *testing.T) {
	truncateTables(t)
	setAdminTopUpMoneySettings(t, 1, 0.6)

	insertRankingUser(t, 4201, "admin-recharge-user")

	quota := int(common.QuotaPerUnit * 10)
	topUp, err := CreateAdminBalanceTopUp(4201, quota)
	require.NoError(t, err)
	require.NotNil(t, topUp)
	assert.Equal(t, int64(quota), topUp.Amount)
	assert.Equal(t, 10.0, topUp.Money)
	assert.Equal(t, 0.06, topUp.Fee)
	assert.Equal(t, PaymentMethodAdminAdd, topUp.PaymentMethod)
	assert.Equal(t, PaymentProviderAdmin, topUp.PaymentProvider)
	assert.Equal(t, common.TopUpStatusSuccess, topUp.Status)
	assert.NotZero(t, topUp.CompleteTime)
	assert.Equal(t, quota, getUserQuotaForPaymentGuardTest(t, 4201))

	rows, total, err := GetAllTopUpsWithOptions(TopUpQueryOptions{PaymentMethod: PaymentMethodAdminAdd}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, topUp.TradeNo, rows[0].TradeNo)
	assert.Equal(t, 10.0, rows[0].Money)
	assert.Equal(t, 0.06, rows[0].Fee)
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

func TestMigrateLegacyAdminAddLogsToTopUpsBackfillsBills(t *testing.T) {
	truncateTables(t)
	setAdminTopUpMoneySettings(t, 1, 0.6)

	insertRankingUser(t, 4206, "legacy-admin-log-user")
	quota := int(common.QuotaPerUnit * 16)
	legacyLog := &Log{
		UserId:    4206,
		Username:  "legacy-admin-log-user",
		CreatedAt: 1900,
		Type:      LogTypeManage,
		Content:   "管理员增加用户额度",
		Quota:     quota,
	}
	require.NoError(t, LOG_DB.Create(legacyLog).Error)

	require.NoError(t, migrateAdminAddedQuotaLogsToRecharge(LOG_DB))
	require.NoError(t, migrateLegacyAdminAddLogsToTopUps(LOG_DB, DB))
	require.NoError(t, migrateLegacyAdminAddLogsToTopUps(LOG_DB, DB))
	require.NoError(t, BackfillAdminTopUpMoneyFromAlipayOfficial(DB))

	rows, total, err := GetAllTopUpsWithOptions(TopUpQueryOptions{PaymentMethod: PaymentMethodAdminAdd}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, fmt.Sprintf("ADMIN_LEGACY_LOG_%d", legacyLog.Id), rows[0].TradeNo)
	assert.Equal(t, int64(quota), rows[0].Amount)
	assert.Equal(t, 16.0, rows[0].Money)
	assert.Equal(t, 0.1, rows[0].Fee)
	assert.Equal(t, PaymentMethodAdminAdd, rows[0].PaymentMethod)
	assert.Equal(t, PaymentProviderAdmin, rows[0].PaymentProvider)
	assert.Equal(t, int64(1900), rows[0].CreateTime)

	var log Log
	require.NoError(t, LOG_DB.First(&log, legacyLog.Id).Error)
	assert.Equal(t, LogTypeTopup, log.Type)
	assert.Equal(t, "管理员充值用户额度", log.Content)
}

func TestBackfilledAdminTopUpsAreSortedByCreateTime(t *testing.T) {
	truncateTables(t)

	insertRankingUser(t, 4207, "legacy-admin-sort-user")
	logs := []Log{
		{
			UserId:    4207,
			Username:  "legacy-admin-sort-user",
			CreatedAt: 1000,
			Type:      LogTypeManage,
			Content:   "管理员增加用户额度",
			Quota:     100,
		},
		{
			UserId:    4207,
			Username:  "legacy-admin-sort-user",
			CreatedAt: 3000,
			Type:      LogTypeManage,
			Content:   "管理员增加用户额度",
			Quota:     300,
		},
		{
			UserId:    4207,
			Username:  "legacy-admin-sort-user",
			CreatedAt: 2000,
			Type:      LogTypeManage,
			Content:   "管理员增加用户额度",
			Quota:     200,
		},
	}
	require.NoError(t, LOG_DB.Create(&logs).Error)

	require.NoError(t, migrateAdminAddedQuotaLogsToRecharge(LOG_DB))
	require.NoError(t, migrateLegacyAdminAddLogsToTopUps(LOG_DB, DB))

	rows, total, err := GetAllTopUpsWithOptions(TopUpQueryOptions{PaymentMethod: PaymentMethodAdminAdd}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, rows, 3)
	assert.Equal(t, int64(3000), rows[0].CreateTime)
	assert.Equal(t, int64(2000), rows[1].CreateTime)
	assert.Equal(t, int64(1000), rows[2].CreateTime)
}

func TestUpdateAdminManagedTopUpEditsBillQuotaAndLog(t *testing.T) {
	truncateTables(t)
	setAdminTopUpMoneySettings(t, 1, 0.6)

	insertRankingUser(t, 4208, "admin-edit-user")
	topUp, err := CreateAdminBalanceTopUp(4208, int(common.QuotaPerUnit*10))
	require.NoError(t, err)
	log := &Log{
		UserId:    4208,
		Username:  "admin-edit-user",
		CreatedAt: topUp.CreateTime,
		Type:      LogTypeTopup,
		Content:   fmt.Sprintf("管理员充值用户额度 %s", logger.LogQuota(int(topUp.Amount))),
		Quota:     int(topUp.Amount),
		Other: common.MapToJsonStr(map[string]interface{}{
			"admin_info": map[string]interface{}{
				"trade_no":         topUp.TradeNo,
				"operation_type":   AdminTopUpOperationRecharge,
				"payment_method":   PaymentMethodAdminAdd,
				"payment_provider": PaymentProviderAdmin,
			},
		}),
	}
	require.NoError(t, LOG_DB.Create(log).Error)

	money := 12.34
	fee := 0.56
	newAmount := int64(common.QuotaPerUnit * 12)
	result, err := UpdateAdminManagedTopUp(AdminTopUpEditParams{
		TradeNo:       topUp.TradeNo,
		OperationType: AdminTopUpOperationRecharge,
		Amount:        newAmount,
		Money:         &money,
		Fee:           &fee,
	})
	require.NoError(t, err)
	require.NoError(t, UpdateAdminTopUpLogForEdit(result, 1, "xauryan"))

	reloaded := GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, reloaded)
	assert.Equal(t, newAmount, reloaded.Amount)
	assert.Equal(t, 12.34, reloaded.Money)
	assert.Equal(t, 0.56, reloaded.Fee)
	assert.Equal(t, int(newAmount), getUserQuotaForPaymentGuardTest(t, 4208))

	var updatedLog Log
	require.NoError(t, LOG_DB.First(&updatedLog, log.Id).Error)
	assert.Equal(t, LogTypeTopup, updatedLog.Type)
	assert.Equal(t, fmt.Sprintf("管理员充值用户额度 %s", logger.LogQuota(int(newAmount))), updatedLog.Content)
	assert.Equal(t, int(newAmount), updatedLog.Quota)
	other, err := common.StrToMap(updatedLog.Other)
	require.NoError(t, err)
	adminInfo := other["admin_info"].(map[string]interface{})
	assert.Equal(t, AdminTopUpOperationRecharge, adminInfo["operation_type"])
	assert.Equal(t, "xauryan", adminInfo["admin_username"])
	assert.Equal(t, 12.34, adminInfo["money"])
	assert.Equal(t, 0.56, adminInfo["fee"])
}

func TestUpdateAdminManagedTopUpConvertsRechargeToGift(t *testing.T) {
	truncateTables(t)
	setAdminTopUpMoneySettings(t, 1, 0.6)

	insertRankingUser(t, 4209, "admin-gift-user")
	topUp, err := CreateAdminBalanceTopUp(4209, int(common.QuotaPerUnit*10))
	require.NoError(t, err)
	log := &Log{
		UserId:    4209,
		Username:  "admin-gift-user",
		CreatedAt: topUp.CreateTime,
		Type:      LogTypeTopup,
		Content:   fmt.Sprintf("管理员充值用户额度 %s", logger.LogQuota(int(topUp.Amount))),
		Quota:     int(topUp.Amount),
		Other: common.MapToJsonStr(map[string]interface{}{
			"admin_info": map[string]interface{}{
				"trade_no":         topUp.TradeNo,
				"operation_type":   AdminTopUpOperationRecharge,
				"payment_method":   PaymentMethodAdminAdd,
				"payment_provider": PaymentProviderAdmin,
			},
		}),
	}
	require.NoError(t, LOG_DB.Create(log).Error)

	result, err := UpdateAdminManagedTopUp(AdminTopUpEditParams{
		TradeNo:         topUp.TradeNo,
		OperationType:   AdminTopUpOperationGift,
		Amount:          topUp.Amount,
		UseDefaultMoney: true,
	})
	require.NoError(t, err)
	require.True(t, result.ConvertedToGift)
	require.NoError(t, UpdateAdminTopUpLogForEdit(result, 1, "xauryan"))

	assert.Nil(t, GetTopUpByTradeNo(topUp.TradeNo))
	assert.Equal(t, int(topUp.Amount), getUserQuotaForPaymentGuardTest(t, 4209))

	var updatedLog Log
	require.NoError(t, LOG_DB.First(&updatedLog, log.Id).Error)
	assert.Equal(t, LogTypeManage, updatedLog.Type)
	assert.Equal(t, fmt.Sprintf("管理员赠送用户额度 %s", logger.LogQuota(int(topUp.Amount))), updatedLog.Content)
	other, err := common.StrToMap(updatedLog.Other)
	require.NoError(t, err)
	adminInfo := other["admin_info"].(map[string]interface{})
	assert.Equal(t, AdminTopUpOperationGift, adminInfo["operation_type"])
	assert.NotContains(t, adminInfo, "trade_no")

	rows, totalQuota, err := GetUserRechargeRankingTotals(0, 0, 10)
	require.NoError(t, err)
	assert.Empty(t, rows)
	assert.Equal(t, int64(0), totalQuota)
}

func TestRefundAdminBalanceTopUpRecordsRefundAndDeductsQuota(t *testing.T) {
	truncateTables(t)
	setAdminTopUpMoneySettings(t, 1, 0.6)

	insertRankingUser(t, 4202, "admin-refund-user")
	quota := int(common.QuotaPerUnit * 10)
	topUp, err := CreateAdminBalanceTopUp(4202, quota)
	require.NoError(t, err)

	refund, err := RefundAdminBalanceTopUp(topUp.TradeNo, int64(common.QuotaPerUnit*3), "manual refund")
	require.NoError(t, err)
	require.NotNil(t, refund)
	assert.Equal(t, TopUpRefundStatusSuccess, refund.Status)
	assert.Equal(t, int64(common.QuotaPerUnit*3), refund.RefundQuota)
	assert.Equal(t, 3.0, refund.RefundAmount)
	assert.Equal(t, "manual refund", refund.Reason)
	assert.Equal(t, int(common.QuotaPerUnit*7), getUserQuotaForPaymentGuardTest(t, 4202))

	reloaded := GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, reloaded)
	assert.Equal(t, common.TopUpStatusPartialRefunded, reloaded.Status)
	assert.Equal(t, int64(common.QuotaPerUnit*3), reloaded.RefundedQuota)
	assert.Equal(t, 3.0, reloaded.RefundedMoney)

	refund, err = RefundAdminBalanceTopUp(topUp.TradeNo, int64(common.QuotaPerUnit*7), "full refund")
	require.NoError(t, err)
	assert.Equal(t, int64(common.QuotaPerUnit*7), refund.RefundQuota)
	assert.Equal(t, 7.0, refund.RefundAmount)
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 4202))

	reloaded = GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, reloaded)
	assert.Equal(t, common.TopUpStatusRefunded, reloaded.Status)
	assert.Equal(t, int64(quota), reloaded.RefundedQuota)
	assert.Equal(t, 10.0, reloaded.RefundedMoney)
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
