package model

import (
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/setting"
	"github.com/Xauryan/stuhelper-ai/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setSelfServeTopUpPricingForTest(t *testing.T) {
	t.Helper()
	originalQuotaPerUnit := common.QuotaPerUnit
	originalPrice := operation_setting.Price
	originalSelfServeUnitPrice := setting.SelfServeTopUpUnitPrice
	originalSingleMaxAmount := setting.SelfServeTopUpSingleMaxAmount
	originalDailyMaxAmount := setting.SelfServeTopUpDailyMaxAmount
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
		operation_setting.Price = originalPrice
		setting.SelfServeTopUpUnitPrice = originalSelfServeUnitPrice
		setting.SelfServeTopUpSingleMaxAmount = originalSingleMaxAmount
		setting.SelfServeTopUpDailyMaxAmount = originalDailyMaxAmount
	})
	common.QuotaPerUnit = 1000
	operation_setting.Price = 9.99
	setting.SelfServeTopUpUnitPrice = 1
	setting.SelfServeTopUpSingleMaxAmount = 199.99
	setting.SelfServeTopUpDailyMaxAmount = 499.99
}

func getSelfServeAuditForTest(t *testing.T, tradeNo string) SelfServeTopUpAudit {
	t.Helper()
	var audit SelfServeTopUpAudit
	require.NoError(t, DB.Where("trade_no = ?", tradeNo).First(&audit).Error)
	return audit
}

func getUserForSelfServeTopUpTest(t *testing.T, userID int) User {
	t.Helper()
	var user User
	require.NoError(t, DB.Where("id = ?", userID).First(&user).Error)
	return user
}

func TestCreateSelfServeTopUpCreditsBalanceAndCreatesPendingAudit(t *testing.T) {
	truncateTables(t)
	setSelfServeTopUpPricingForTest(t)

	insertRankingUser(t, 5101, "self-serve-create-user")

	result, err := CreateSelfServeTopUp(SelfServeTopUpCreateParams{
		UserId:        5101,
		PaymentMethod: PaymentMethodAlipaySelfServe,
		DeclaredMoney: 10.12,
		TransactionNo: "SELF_SERVE_TX_001",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.TopUp)
	require.NotNil(t, result.Audit)
	assert.Equal(t, int64(10120), result.QuotaDelta)
	assert.Equal(t, common.TopUpStatusSuccess, result.TopUp.Status)
	assert.Equal(t, PaymentProviderSelfServe, result.TopUp.PaymentProvider)
	assert.Equal(t, PaymentMethodAlipaySelfServe, result.TopUp.PaymentMethod)
	assert.Equal(t, SelfServeTopUpAuditStatusPending, result.Audit.Status)
	assert.Equal(t, "SELF_SERVE_TX_001", result.Audit.TransactionNo)
	assert.Equal(t, int64(10120), result.Audit.CreditedQuota)
	assert.Equal(t, 10120, getUserQuotaForPaymentGuardTest(t, 5101))

	rows, total, err := GetAllTopUpsWithOptions(
		TopUpQueryOptions{
			PaymentMethod: PaymentProviderSelfServe,
			AuditStatus:   SelfServeTopUpAuditStatusPending,
		},
		&common.PageInfo{Page: 1, PageSize: 10},
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, result.TopUp.TradeNo, rows[0].TradeNo)
	assert.Equal(t, SelfServeTopUpAuditStatusPending, rows[0].AuditStatus)
	assert.Equal(t, "SELF_SERVE_TX_001", rows[0].TransactionNo)
	assert.Equal(t, 10.12, rows[0].DeclaredMoney)
	assert.Equal(t, int64(10120), rows[0].CreditedQuota)
}

func TestCreateSelfServeTopUpRejectsUnconfiguredLimits(t *testing.T) {
	truncateTables(t)
	setSelfServeTopUpPricingForTest(t)
	setting.SelfServeTopUpSingleMaxAmount = 0
	setting.SelfServeTopUpDailyMaxAmount = 0

	insertRankingUser(t, 5108, "self-serve-unconfigured-limit-user")

	_, err := CreateSelfServeTopUp(SelfServeTopUpCreateParams{
		UserId:        5108,
		PaymentMethod: PaymentMethodAlipaySelfServe,
		DeclaredMoney: 10,
		TransactionNo: "SELF_SERVE_UNCONFIGURED_LIMIT",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "请先配置自助充值限额")
}

func TestCreateSelfServeTopUpUsesIndependentSelfServeUnitPrice(t *testing.T) {
	truncateTables(t)
	setSelfServeTopUpPricingForTest(t)

	insertRankingUser(t, 5109, "self-serve-independent-price-user")

	result, err := CreateSelfServeTopUp(SelfServeTopUpCreateParams{
		UserId:        5109,
		PaymentMethod: PaymentMethodAlipaySelfServe,
		DeclaredMoney: 10,
		TransactionNo: "SELF_SERVE_INDEPENDENT_PRICE",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(10000), result.QuotaDelta)
	assert.Equal(t, int64(10000), result.Audit.CreditedQuota)

	setting.SelfServeTopUpUnitPrice = 2
	result, err = CreateSelfServeTopUp(SelfServeTopUpCreateParams{
		UserId:        5109,
		PaymentMethod: PaymentMethodWechatSelfServe,
		DeclaredMoney: 10,
		TransactionNo: "SELF_SERVE_UNIT_PRICE_TWO",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(5000), result.QuotaDelta)
	assert.Equal(t, int64(5000), result.Audit.CreditedQuota)
}

func TestCreateSelfServeTopUpRejectsInvalidUnitPrice(t *testing.T) {
	truncateTables(t)
	setSelfServeTopUpPricingForTest(t)
	setting.SelfServeTopUpUnitPrice = 0

	insertRankingUser(t, 5110, "self-serve-invalid-price-user")

	_, err := CreateSelfServeTopUp(SelfServeTopUpCreateParams{
		UserId:        5110,
		PaymentMethod: PaymentMethodAlipaySelfServe,
		DeclaredMoney: 10,
		TransactionNo: "SELF_SERVE_INVALID_PRICE",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "自助充值价格配置错误")
}

func TestCreateSelfServeTopUpEnforcesSingleAndDailyMoneyLimits(t *testing.T) {
	truncateTables(t)
	setSelfServeTopUpPricingForTest(t)

	insertRankingUser(t, 5102, "self-serve-limit-user")

	_, err := CreateSelfServeTopUp(SelfServeTopUpCreateParams{
		UserId:        5102,
		PaymentMethod: PaymentMethodWechatSelfServe,
		DeclaredMoney: 200,
		TransactionNo: "SELF_SERVE_LIMIT_SINGLE",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "单笔自助充值金额不能超过 199.99 元")

	transactionNos := []string{"SELF_SERVE_LIMIT_DAILY_OK_A", "SELF_SERVE_LIMIT_DAILY_OK_B"}
	for index, money := range []float64{199.99, 199.99} {
		_, err = CreateSelfServeTopUp(SelfServeTopUpCreateParams{
			UserId:        5102,
			PaymentMethod: PaymentMethodWechatSelfServe,
			DeclaredMoney: money,
			TransactionNo: transactionNos[index],
		})
		require.NoError(t, err)
	}
	_, err = CreateSelfServeTopUp(SelfServeTopUpCreateParams{
		UserId:        5102,
		PaymentMethod: PaymentMethodWechatSelfServe,
		DeclaredMoney: 100.02,
		TransactionNo: "SELF_SERVE_LIMIT_DAILY_FAIL",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "每日自助充值金额不能超过 499.99 元")
}

func TestCreateSelfServeTopUpRejectsDuplicateTransactionNo(t *testing.T) {
	truncateTables(t)
	setSelfServeTopUpPricingForTest(t)

	insertRankingUser(t, 5103, "self-serve-duplicate-user")
	_, err := CreateSelfServeTopUp(SelfServeTopUpCreateParams{
		UserId:        5103,
		PaymentMethod: PaymentMethodAlipaySelfServe,
		DeclaredMoney: 9.99,
		TransactionNo: "SELF_SERVE_DUPLICATE_TX",
	})
	require.NoError(t, err)

	_, err = CreateSelfServeTopUp(SelfServeTopUpCreateParams{
		UserId:        5103,
		PaymentMethod: PaymentMethodWechatSelfServe,
		DeclaredMoney: 8.88,
		TransactionNo: "SELF_SERVE_DUPLICATE_TX",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "该交易订单号已提交")
}

func TestUpdateSelfServeTopUpAdjustsBalanceAndPendingAudit(t *testing.T) {
	truncateTables(t)
	setSelfServeTopUpPricingForTest(t)

	insertRankingUser(t, 5104, "self-serve-edit-user")
	created, err := CreateSelfServeTopUp(SelfServeTopUpCreateParams{
		UserId:        5104,
		PaymentMethod: PaymentMethodAlipaySelfServe,
		DeclaredMoney: 10,
		TransactionNo: "SELF_SERVE_EDIT_ORIGINAL",
	})
	require.NoError(t, err)

	updated, err := UpdateSelfServeTopUp(SelfServeTopUpEditParams{
		TradeNo:       created.TopUp.TradeNo,
		DeclaredMoney: 12.34,
		TransactionNo: "SELF_SERVE_EDIT_CORRECTED",
		AdminReason:   "corrected amount",
		AuditorId:     7,
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, int64(2340), updated.QuotaDelta)
	assert.Equal(t, int64(12340), updated.TopUp.Amount)
	assert.Equal(t, 12.34, updated.TopUp.Money)
	assert.Equal(t, "SELF_SERVE_EDIT_CORRECTED", updated.Audit.TransactionNo)
	assert.Equal(t, int64(12340), updated.Audit.CreditedQuota)
	assert.Equal(t, 12340, getUserQuotaForPaymentGuardTest(t, 5104))

	audit := getSelfServeAuditForTest(t, created.TopUp.TradeNo)
	assert.Equal(t, SelfServeTopUpAuditStatusPending, audit.Status)
	assert.Equal(t, 7, audit.AuditorId)
	assert.Equal(t, "corrected amount", audit.AdminReason)
}

func TestApproveSelfServeTopUpMarksAuditApproved(t *testing.T) {
	truncateTables(t)
	setSelfServeTopUpPricingForTest(t)

	insertRankingUser(t, 5105, "self-serve-approve-user")
	created, err := CreateSelfServeTopUp(SelfServeTopUpCreateParams{
		UserId:        5105,
		PaymentMethod: PaymentMethodAlipaySelfServe,
		DeclaredMoney: 10,
		TransactionNo: "SELF_SERVE_APPROVE_TX",
	})
	require.NoError(t, err)

	approved, err := ApproveSelfServeTopUp(created.TopUp.TradeNo, 9, "matched")
	require.NoError(t, err)
	require.NotNil(t, approved)
	assert.Equal(t, SelfServeTopUpAuditStatusApproved, approved.Audit.Status)
	assert.Equal(t, 9, approved.Audit.AuditorId)
	assert.NotZero(t, approved.Audit.ReviewedTime)
	assert.Equal(t, 10000, getUserQuotaForPaymentGuardTest(t, 5105))

	approvedAgain, err := ApproveSelfServeTopUp(created.TopUp.TradeNo, 9, "matched")
	require.NoError(t, err)
	assert.Equal(t, SelfServeTopUpAuditStatusApproved, approvedAgain.Audit.Status)
	assert.Equal(t, 10000, getUserQuotaForPaymentGuardTest(t, 5105))
}

func TestRejectSelfServeTopUpDeductsCreditedBalanceAndCanBanUser(t *testing.T) {
	truncateTables(t)
	setSelfServeTopUpPricingForTest(t)

	insertRankingUser(t, 5106, "self-serve-reject-user")
	created, err := CreateSelfServeTopUp(SelfServeTopUpCreateParams{
		UserId:        5106,
		PaymentMethod: PaymentMethodWechatSelfServe,
		DeclaredMoney: 20,
		TransactionNo: "SELF_SERVE_REJECT_TX",
	})
	require.NoError(t, err)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 5106).Update("quota", 1000).Error)

	rejected, err := RejectSelfServeTopUp(created.TopUp.TradeNo, 11, "fake transaction", true)
	require.NoError(t, err)
	require.NotNil(t, rejected)
	assert.Equal(t, SelfServeTopUpAuditStatusRejected, rejected.Audit.Status)
	assert.Equal(t, int64(-20000), rejected.QuotaDelta)
	assert.True(t, rejected.Banned)

	reloadedTopUp := GetTopUpByTradeNo(created.TopUp.TradeNo)
	require.NotNil(t, reloadedTopUp)
	assert.Equal(t, common.TopUpStatusRefunded, reloadedTopUp.Status)
	assert.Equal(t, int64(20000), reloadedTopUp.RefundedQuota)
	assert.Equal(t, 20.0, reloadedTopUp.RefundedMoney)

	user := getUserForSelfServeTopUpTest(t, 5106)
	assert.Equal(t, -19000, user.Quota)
	assert.Equal(t, common.UserStatusDisabled, user.Status)

	var refund TopUpRefund
	require.NoError(t, DB.Where("trade_no = ?", created.TopUp.TradeNo).First(&refund).Error)
	assert.Equal(t, TopUpRefundStatusSuccess, refund.Status)
	assert.Equal(t, int64(20000), refund.RefundQuota)
	assert.Equal(t, PaymentProviderSelfServe, refund.PaymentProvider)
}

func TestRejectSelfServeTopUpCanBanAlreadyRejectedWithoutDoubleDeduct(t *testing.T) {
	truncateTables(t)
	setSelfServeTopUpPricingForTest(t)

	insertRankingUser(t, 5107, "self-serve-reject-idempotent-user")
	created, err := CreateSelfServeTopUp(SelfServeTopUpCreateParams{
		UserId:        5107,
		PaymentMethod: PaymentMethodWechatSelfServe,
		DeclaredMoney: 20,
		TransactionNo: "SELF_SERVE_REJECT_IDEMPOTENT_TX",
	})
	require.NoError(t, err)

	first, err := RejectSelfServeTopUp(created.TopUp.TradeNo, 11, "fake transaction", false)
	require.NoError(t, err)
	assert.False(t, first.Banned)
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 5107))

	second, err := RejectSelfServeTopUp(created.TopUp.TradeNo, 11, "ban after review", true)
	require.NoError(t, err)
	assert.True(t, second.Banned)
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 5107))

	user := getUserForSelfServeTopUpTest(t, 5107)
	assert.Equal(t, common.UserStatusDisabled, user.Status)

	var refundCount int64
	require.NoError(t, DB.Model(&TopUpRefund{}).Where("trade_no = ?", created.TopUp.TradeNo).Count(&refundCount).Error)
	assert.Equal(t, int64(1), refundCount)
}
