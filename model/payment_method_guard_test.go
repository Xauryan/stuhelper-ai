package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertUserForPaymentGuardTest(t *testing.T, id int, quota int) {
	t.Helper()
	user := &User{
		Id:       id,
		Username: "payment_guard_user",
		Status:   common.UserStatusEnabled,
		Quota:    quota,
	}
	require.NoError(t, DB.Create(user).Error)
}

func insertSubscriptionPlanForPaymentGuardTest(t *testing.T, id int) *SubscriptionPlan {
	t.Helper()
	plan := &SubscriptionPlan{
		Id:            id,
		Title:         "Guard Plan",
		PriceAmount:   9.99,
		Currency:      "USD",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   1000,
	}
	require.NoError(t, DB.Create(plan).Error)
	return plan
}

func insertSubscriptionOrderForPaymentGuardTest(t *testing.T, tradeNo string, userID int, planID int, paymentProvider string) {
	t.Helper()
	order := &SubscriptionOrder{
		UserId:          userID,
		PlanId:          planID,
		Money:           9.99,
		TradeNo:         tradeNo,
		PaymentMethod:   paymentProvider,
		PaymentProvider: paymentProvider,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, order.Insert())
}

func insertTopUpForPaymentGuardTest(t *testing.T, tradeNo string, userID int, paymentProvider string) {
	t.Helper()
	topUp := &TopUp{
		UserId:          userID,
		Amount:          2,
		Money:           9.99,
		TradeNo:         tradeNo,
		PaymentMethod:   paymentProvider,
		PaymentProvider: paymentProvider,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())
}

func getTopUpStatusForPaymentGuardTest(t *testing.T, tradeNo string) string {
	t.Helper()
	topUp := GetTopUpByTradeNo(tradeNo)
	require.NotNil(t, topUp)
	return topUp.Status
}

func countUserSubscriptionsForPaymentGuardTest(t *testing.T, userID int) int64 {
	t.Helper()
	var count int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ?", userID).Count(&count).Error)
	return count
}

func getUserQuotaForPaymentGuardTest(t *testing.T, userID int) int {
	t.Helper()
	var user User
	require.NoError(t, DB.Select("quota").Where("id = ?", userID).First(&user).Error)
	return user.Quota
}

func TestRechargeWaffoPancake_RejectsMismatchedPaymentMethod(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 101, 0)
	insertTopUpForPaymentGuardTest(t, "waffo-pancake-guard", 101, PaymentProviderStripe)

	err := RechargeWaffoPancake("waffo-pancake-guard")
	require.Error(t, err)

	topUp := GetTopUpByTradeNo("waffo-pancake-guard")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusPending, topUp.Status)
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 101))
}

func TestRechargeOfficialPayment_RejectsMismatchedPaymentMethod(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 102, 0)
	insertTopUpForPaymentGuardTest(t, "official-guard", 102, PaymentProviderAlipayOfficial)

	err := RechargeOfficialPayment("official-guard", PaymentProviderWechatPayOfficial, PaymentMethodWechatPayOfficial, "127.0.0.1")
	require.Error(t, err)

	topUp := GetTopUpByTradeNo("official-guard")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusPending, topUp.Status)
	assert.Equal(t, PaymentProviderAlipayOfficial, topUp.PaymentProvider)
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 102))
}

func TestRechargeOfficialPayment_RejectsMismatchedPaidMoney(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 104, 0)
	insertTopUpForPaymentGuardTest(t, "official-money-guard", 104, PaymentProviderWechatPayOfficial)

	err := RechargeOfficialPayment("official-money-guard", PaymentProviderWechatPayOfficial, PaymentMethodWechatPayOfficial, "127.0.0.1", 9.98)
	require.Error(t, err)

	topUp := GetTopUpByTradeNo("official-money-guard")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusPending, topUp.Status)
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 104))
}

func TestRechargeOfficialPayment_CreditsQuotaAndMarksSuccess(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 103, 10)
	insertTopUpForPaymentGuardTest(t, "official-success", 103, PaymentProviderWechatPayOfficial)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 500000

	err := RechargeOfficialPayment("official-success", PaymentProviderWechatPayOfficial, PaymentMethodWechatPayOfficial, "127.0.0.1", 9.99)
	require.NoError(t, err)

	topUp := GetTopUpByTradeNo("official-success")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusSuccess, topUp.Status)
	assert.Equal(t, PaymentMethodWechatPayOfficial, topUp.PaymentMethod)
	assert.Equal(t, 1000010, getUserQuotaForPaymentGuardTest(t, 103))
}

func TestRecordOfficialPaymentRefundLogUsesRefundTypeAndAuditFields(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 185, 0)

	RecordOfficialPaymentRefundLog(
		185,
		"管理员发起支付宝官方退款成功，订单号：ALIPAY_TEST，退款金额：0.50，退回额度：$0.500000",
		"203.0.113.8",
		PaymentMethodAlipayOfficial,
		PaymentProviderAlipayOfficial,
	)

	var log Log
	require.NoError(t, LOG_DB.Where("user_id = ? AND type = ?", 185, LogTypeRefund).First(&log).Error)
	assert.Equal(t, "203.0.113.8", log.Ip)
	assert.Equal(t, "", log.TokenName)
	assert.Equal(t, "", log.ModelName)
	assert.Equal(t, 0, log.ChannelId)
	assert.Equal(t, 0, log.PromptTokens)
	assert.Equal(t, 0, log.CompletionTokens)
	assert.Equal(t, 0, log.Quota)

	var other map[string]interface{}
	require.NoError(t, common.UnmarshalJsonStr(log.Other, &other))
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "203.0.113.8", adminInfo["caller_ip"])
	assert.Equal(t, PaymentMethodAlipayOfficial, adminInfo["payment_method"])
	assert.Equal(t, PaymentProviderAlipayOfficial, adminInfo["callback_payment_method"])
	assert.NotEmpty(t, adminInfo["version"])
}

func TestRechargeOfficialPaymentTreatsRefundedOrdersAsAlreadyCompleted(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 105, 100)
	topUp := &TopUp{
		UserId:          105,
		Amount:          2,
		Money:           9.99,
		TradeNo:         "official-refunded-notify",
		PaymentMethod:   PaymentMethodAlipayOfficial,
		PaymentProvider: PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusPartialRefunded,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())

	err := RechargeOfficialPayment("official-refunded-notify", PaymentProviderAlipayOfficial, PaymentMethodAlipayOfficial, "127.0.0.1", 9.99)
	require.NoError(t, err)
	assert.Equal(t, 100, getUserQuotaForPaymentGuardTest(t, 105))
}

func TestCreateOfficialPaymentRefundRejectsOverRefundAndDeductsPartialQuota(t *testing.T) {
	truncateTables(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 100

	insertUserForPaymentGuardTest(t, 170, 1000)
	topUp := &TopUp{
		UserId:          170,
		Amount:          10,
		Money:           1.00,
		TradeNo:         "official-refund-partial",
		PaymentMethod:   PaymentMethodAlipayOfficial,
		PaymentProvider: PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusSuccess,
		CompleteTime:    time.Now().Unix(),
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())

	refund, err := CreateOfficialPaymentRefund(OfficialPaymentRefundCreateParams{
		TradeNo:         topUp.TradeNo,
		PaymentProvider: PaymentProviderAlipayOfficial,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RefundAmount:    0.40,
		Reason:          "partial refund",
		OutRequestNo:    "official-refund-partial-rf-1",
	})
	require.NoError(t, err)
	require.NotNil(t, refund)
	assert.Equal(t, TopUpRefundStatusPending, refund.Status)
	assert.Equal(t, int64(400), refund.RefundQuota)
	assert.Equal(t, 600, getUserQuotaForPaymentGuardTest(t, 170))

	err = MarkTopUpRefundSuccess("official-refund-partial-rf-1", "202605142200000000", `{"fund_change":"Y"}`)
	require.NoError(t, err)

	updated := GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, updated)
	assert.Equal(t, common.TopUpStatusPartialRefunded, updated.Status)
	assert.Equal(t, 0.40, updated.RefundedMoney)
	assert.Equal(t, int64(400), updated.RefundedQuota)

	_, err = CreateOfficialPaymentRefund(OfficialPaymentRefundCreateParams{
		TradeNo:         topUp.TradeNo,
		PaymentProvider: PaymentProviderAlipayOfficial,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RefundAmount:    0.61,
		Reason:          "too much",
		OutRequestNo:    "official-refund-partial-rf-2",
	})
	require.Error(t, err)
	assert.Equal(t, 600, getUserQuotaForPaymentGuardTest(t, 170))
}

func TestCalculateOfficialPaymentRefundPreviewCapsBalanceRefundByCurrentQuota(t *testing.T) {
	truncateTables(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 1

	insertUserForPaymentGuardTest(t, 270, 150)
	topUp := &TopUp{
		UserId:          270,
		Amount:          100,
		Money:           100.00,
		TradeNo:         "official-refund-preview-full",
		PaymentMethod:   PaymentMethodWechatPayOfficial,
		PaymentProvider: PaymentProviderWechatPayOfficial,
		Status:          common.TopUpStatusSuccess,
		CompleteTime:    time.Now().Unix(),
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())

	preview, err := CalculateOfficialPaymentRefundPreview(topUp.TradeNo)
	require.NoError(t, err)
	require.True(t, preview.Refundable)
	assert.InDelta(t, 100.00, preview.MaxRefundAmount, 0.0001)
	assert.Equal(t, int64(100), preview.MaxRefundQuota)

	require.NoError(t, DB.Model(&User{}).Where("id = ?", 270).Update("quota", 50).Error)
	preview, err = CalculateOfficialPaymentRefundPreview(topUp.TradeNo)
	require.NoError(t, err)
	require.True(t, preview.Refundable)
	assert.InDelta(t, 50.00, preview.MaxRefundAmount, 0.0001)
	assert.Equal(t, int64(50), preview.MaxRefundQuota)
}

func TestCalculateOfficialPaymentRefundPreviewUsesStrictSubscriptionUnusedRatio(t *testing.T) {
	truncateTables(t)

	now := common.GetTimestamp()
	insertUserForPaymentGuardTest(t, 271, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 407)
	order := &SubscriptionOrder{
		UserId:          271,
		PlanId:          plan.Id,
		Money:           100.00,
		TradeNo:         "WXSUB_REFUND_PREVIEW",
		PaymentMethod:   PaymentMethodWechatPayOfficial,
		PaymentProvider: PaymentProviderWechatPayOfficial,
		Status:          common.TopUpStatusSuccess,
		CreateTime:      now - 500,
		CompleteTime:    now - 500,
	}
	require.NoError(t, order.Insert())
	require.NoError(t, DB.Create(&UserSubscription{
		UserId:      271,
		PlanId:      plan.Id,
		AmountTotal: 1000,
		AmountUsed:  200,
		StartTime:   now - 500,
		EndTime:     now + 500,
		Status:      "active",
		Source:      "order",
	}).Error)

	preview, err := CalculateOfficialPaymentRefundPreview(order.TradeNo)
	require.NoError(t, err)
	require.True(t, preview.Refundable)
	assert.True(t, preview.IsSubscription)
	assert.InDelta(t, 50.00, preview.MaxRefundAmount, 0.0001)
	assert.Equal(t, int64(500), preview.MaxRefundQuota)
}

func TestCreateTopUpRefundRequestRequiresReasonAndExposesPendingRequest(t *testing.T) {
	truncateTables(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 1

	insertUserForPaymentGuardTest(t, 273, 50)
	topUp := &TopUp{
		UserId:          273,
		Amount:          100,
		Money:           100.00,
		TradeNo:         "official-refund-request",
		PaymentMethod:   PaymentMethodWechatPayOfficial,
		PaymentProvider: PaymentProviderWechatPayOfficial,
		Status:          common.TopUpStatusSuccess,
		CompleteTime:    time.Now().Unix(),
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())

	_, _, err := CreateTopUpRefundRequest(273, topUp.TradeNo, 10.00, "")
	require.Error(t, err)

	request, preview, err := CreateTopUpRefundRequest(273, topUp.TradeNo, 50.00, "unused quota")
	require.NoError(t, err)
	require.NotNil(t, request)
	require.NotNil(t, preview)
	assert.Equal(t, TopUpRefundRequestStatusPending, request.Status)
	assert.InDelta(t, 50.00, request.RequestedAmount, 0.0001)
	assert.InDelta(t, 50.00, request.MaxRefundAmount, 0.0001)
	assert.Equal(t, int64(50), request.MaxRefundQuota)

	_, _, err = CreateTopUpRefundRequest(273, topUp.TradeNo, 50.01, "too much")
	require.Error(t, err)

	topups, total, err := GetUserTopUps(273, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, topups, 1)
	assert.Equal(t, request.Id, topups[0].RefundRequestId)
	assert.Equal(t, TopUpRefundRequestStatusPending, topups[0].RefundRequestStatus)
	assert.InDelta(t, 50.00, topups[0].RefundRequestAmount, 0.0001)
	assert.Equal(t, "unused quota", topups[0].RefundRequestReason)
}

func TestCreateOfficialPaymentRefundAllowsSubscriptionRefundWithoutDeductingWalletQuota(t *testing.T) {
	truncateTables(t)

	now := common.GetTimestamp()
	insertUserForPaymentGuardTest(t, 272, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 408)
	order := &SubscriptionOrder{
		UserId:          272,
		PlanId:          plan.Id,
		Money:           100.00,
		TradeNo:         "WXSUB_REFUND_EXEC",
		PaymentMethod:   PaymentMethodWechatPayOfficial,
		PaymentProvider: PaymentProviderWechatPayOfficial,
		Status:          common.TopUpStatusSuccess,
		CreateTime:      now - 100,
		CompleteTime:    now - 100,
	}
	require.NoError(t, order.Insert())
	require.NoError(t, DB.Create(&UserSubscription{
		UserId:      272,
		PlanId:      plan.Id,
		AmountTotal: 1000,
		AmountUsed:  0,
		StartTime:   now - 100,
		EndTime:     now + 900,
		Status:      "active",
		Source:      "order",
	}).Error)

	refund, err := CreateOfficialPaymentRefund(OfficialPaymentRefundCreateParams{
		TradeNo:         order.TradeNo,
		PaymentProvider: PaymentProviderWechatPayOfficial,
		PaymentMethod:   PaymentMethodWechatPayOfficial,
		RefundAmount:    10.00,
		Reason:          "subscription partial",
		OutRequestNo:    "WXSUB_REFUND_EXEC_RF_1",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), refund.RefundQuota)
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 272))

	require.NoError(t, MarkTopUpRefundSuccess(refund.OutRequestNo, "WXREFUNDID", "{}"))
	require.NoError(t, RefundSubscriptionOrder(order.TradeNo, PaymentProviderWechatPayOfficial, refund.RefundAmount, false))

	topUp := GetTopUpByTradeNo(order.TradeNo)
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusPartialRefunded, topUp.Status)
	assert.InDelta(t, 10.00, topUp.RefundedMoney, 0.0001)

	reloadedOrder := GetSubscriptionOrderByTradeNo(order.TradeNo)
	require.NotNil(t, reloadedOrder)
	assert.Equal(t, common.TopUpStatusPartialRefunded, reloadedOrder.Status)
}

func TestRefundSubscriptionOrderReversesSubscriptionReferralCommission(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 0)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 1

	now := common.GetTimestamp()
	inviter := &User{Id: 274, Username: "sub-refund-inviter", Status: common.UserStatusEnabled, AffCode: "sub-refund-inviter-aff", AffQuota: 10, AffHistoryQuota: 10}
	invitee := &User{Id: 275, Username: "sub-refund-invitee", Status: common.UserStatusEnabled, AffCode: "sub-refund-invitee-aff", InviterId: 274}
	require.NoError(t, DB.Create(inviter).Error)
	require.NoError(t, DB.Create(invitee).Error)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 409)
	order := &SubscriptionOrder{
		UserId:          275,
		PlanId:          plan.Id,
		Money:           100.00,
		TradeNo:         "WXSUB_REFUND_COMMISSION",
		PaymentMethod:   PaymentMethodWechatPayOfficial,
		PaymentProvider: PaymentProviderWechatPayOfficial,
		Status:          common.TopUpStatusSuccess,
		CreateTime:      now - 100,
		CompleteTime:    now - 100,
	}
	require.NoError(t, order.Insert())
	require.NoError(t, DB.Create(&UserSubscription{
		UserId:      275,
		PlanId:      plan.Id,
		AmountTotal: 1000,
		StartTime:   now - 100,
		EndTime:     now + 900,
		Status:      "active",
		Source:      "order",
	}).Error)
	require.NoError(t, DB.Create(&ReferralCommission{
		InviterId:       274,
		InviteeId:       275,
		SourceType:      ReferralCommissionSourceSubscription,
		SourceId:        order.Id,
		PaymentMethod:   PaymentMethodWechatPayOfficial,
		RechargeAmount:  100.00,
		CommissionQuota: 10,
		CommissionRate:  10,
	}).Error)

	refund, err := CreateOfficialPaymentRefund(OfficialPaymentRefundCreateParams{
		TradeNo:         order.TradeNo,
		PaymentProvider: PaymentProviderWechatPayOfficial,
		PaymentMethod:   PaymentMethodWechatPayOfficial,
		RefundAmount:    40.00,
		Reason:          "subscription partial",
		OutRequestNo:    "WXSUB_REFUND_COMMISSION_RF_1",
	})
	require.NoError(t, err)
	require.NoError(t, MarkTopUpRefundSuccess(refund.OutRequestNo, "WXREFUNDID", "{}"))
	require.NoError(t, RefundSubscriptionOrder(order.TradeNo, PaymentProviderWechatPayOfficial, refund.RefundAmount, false))

	var updatedInviter User
	require.NoError(t, DB.First(&updatedInviter, 274).Error)
	assert.Equal(t, 6, updatedInviter.AffQuota)
	assert.Equal(t, 6, updatedInviter.AffHistoryQuota)

	var updatedCommission ReferralCommission
	require.NoError(t, DB.Where("source_type = ? AND source_id = ?", ReferralCommissionSourceSubscription, order.Id).First(&updatedCommission).Error)
	assert.Equal(t, 4, updatedCommission.RefundedCommissionQuota)
	assert.InDelta(t, 40.00, updatedCommission.RefundedRechargeAmount, 0.0001)
}

func TestCreateOfficialPaymentRefundFullRefundDeductsRemainingQuota(t *testing.T) {
	truncateTables(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 100

	insertUserForPaymentGuardTest(t, 171, 1000)
	topUp := &TopUp{
		UserId:          171,
		Amount:          10,
		Money:           1.00,
		TradeNo:         "official-refund-full",
		PaymentMethod:   PaymentMethodAlipayOfficial,
		PaymentProvider: PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusPartialRefunded,
		RefundedMoney:   0.40,
		RefundedQuota:   400,
		CompleteTime:    time.Now().Unix(),
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())

	refund, err := CreateOfficialPaymentRefund(OfficialPaymentRefundCreateParams{
		TradeNo:         topUp.TradeNo,
		PaymentProvider: PaymentProviderAlipayOfficial,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RefundAmount:    0.60,
		Reason:          "full refund remaining",
		OutRequestNo:    "official-refund-full-rf-1",
	})
	require.NoError(t, err)
	require.NotNil(t, refund)
	assert.Equal(t, int64(600), refund.RefundQuota)
	assert.Equal(t, 400, getUserQuotaForPaymentGuardTest(t, 171))

	err = MarkTopUpRefundSuccess("official-refund-full-rf-1", "202605142200000001", `{"fund_change":"Y"}`)
	require.NoError(t, err)

	updated := GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, updated)
	assert.Equal(t, common.TopUpStatusRefunded, updated.Status)
	assert.Equal(t, 1.00, updated.RefundedMoney)
	assert.Equal(t, int64(1000), updated.RefundedQuota)
}

func TestCreateOfficialPaymentRefundReversesReferralCommission(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 0)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 100

	inviter := &User{Id: 172, Username: "refund-inviter", Status: common.UserStatusEnabled, AffCode: "refund-inviter-aff", AffQuota: 100, AffHistoryQuota: 100}
	invitee := &User{Id: 173, Username: "refund-invitee", Status: common.UserStatusEnabled, AffCode: "refund-invitee-aff", InviterId: 172, Quota: 1000}
	require.NoError(t, DB.Create(inviter).Error)
	require.NoError(t, DB.Create(invitee).Error)
	topUp := &TopUp{
		UserId:          173,
		Amount:          10,
		Money:           1.00,
		TradeNo:         "official-refund-commission",
		PaymentMethod:   PaymentMethodAlipayOfficial,
		PaymentProvider: PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusSuccess,
		CompleteTime:    time.Now().Unix(),
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())
	require.NoError(t, DB.Create(&ReferralCommission{
		InviterId:       172,
		InviteeId:       173,
		SourceType:      ReferralCommissionSourceTopUp,
		SourceId:        topUp.Id,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RechargeAmount:  1.00,
		CommissionQuota: 10,
		CommissionRate:  10,
	}).Error)

	_, err := CreateOfficialPaymentRefund(OfficialPaymentRefundCreateParams{
		TradeNo:         topUp.TradeNo,
		PaymentProvider: PaymentProviderAlipayOfficial,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RefundAmount:    0.40,
		Reason:          "commission refund",
		OutRequestNo:    "official-refund-commission-rf-1",
	})
	require.NoError(t, err)

	var updatedInviter User
	require.NoError(t, DB.First(&updatedInviter, 172).Error)
	assert.Equal(t, 96, updatedInviter.AffQuota)
	assert.Equal(t, 96, updatedInviter.AffHistoryQuota)

	var updatedCommission ReferralCommission
	require.NoError(t, DB.Where("source_type = ? AND source_id = ?", ReferralCommissionSourceTopUp, topUp.Id).First(&updatedCommission).Error)
	assert.Equal(t, 10, updatedCommission.CommissionQuota)
	assert.Equal(t, 4, updatedCommission.RefundedCommissionQuota)
	assert.InDelta(t, 1.00, updatedCommission.RechargeAmount, 0.0001)
	assert.InDelta(t, 0.40, updatedCommission.RefundedRechargeAmount, 0.0001)

	records, total, err := GetAdminReferralRecords(&AdminReferralQuery{PageInfo: &common.PageInfo{Page: 1, PageSize: 20}})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, records, 1)
	assert.Equal(t, 6, records[0].TotalCommissionQuota)
	assert.InDelta(t, 0.60, records[0].TotalRechargeAmount, 0.0001)
}

func TestCreateOfficialPaymentRefundFullyReversesReferralCommissionAfterSplitRefunds(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 0)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 100

	inviter := &User{Id: 182, Username: "split-refund-inviter", Status: common.UserStatusEnabled, AffCode: "split-refund-inviter-aff", AffQuota: 100, AffHistoryQuota: 100}
	invitee := &User{Id: 183, Username: "split-refund-invitee", Status: common.UserStatusEnabled, AffCode: "split-refund-invitee-aff", InviterId: 182, Quota: 1000}
	require.NoError(t, DB.Create(inviter).Error)
	require.NoError(t, DB.Create(invitee).Error)
	topUp := &TopUp{
		UserId:          183,
		Amount:          10,
		Money:           1.00,
		TradeNo:         "official-split-refund-commission",
		PaymentMethod:   PaymentMethodAlipayOfficial,
		PaymentProvider: PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusSuccess,
		CompleteTime:    time.Now().Unix(),
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())
	require.NoError(t, DB.Create(&ReferralCommission{
		InviterId:       182,
		InviteeId:       183,
		SourceType:      ReferralCommissionSourceTopUp,
		SourceId:        topUp.Id,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RechargeAmount:  1.00,
		CommissionQuota: 10,
		CommissionRate:  10,
	}).Error)

	for i, amount := range []float64{0.34, 0.33, 0.33} {
		_, err := CreateOfficialPaymentRefund(OfficialPaymentRefundCreateParams{
			TradeNo:         topUp.TradeNo,
			PaymentProvider: PaymentProviderAlipayOfficial,
			PaymentMethod:   PaymentMethodAlipayOfficial,
			RefundAmount:    amount,
			Reason:          "split commission refund",
			OutRequestNo:    fmt.Sprintf("official-split-refund-commission-rf-%d", i+1),
		})
		require.NoError(t, err)
	}

	var updatedInviter User
	require.NoError(t, DB.First(&updatedInviter, 182).Error)
	assert.Equal(t, 90, updatedInviter.AffQuota)
	assert.Equal(t, 90, updatedInviter.AffHistoryQuota)

	var updatedCommission ReferralCommission
	require.NoError(t, DB.Where("source_type = ? AND source_id = ?", ReferralCommissionSourceTopUp, topUp.Id).First(&updatedCommission).Error)
	assert.Equal(t, 10, updatedCommission.CommissionQuota)
	assert.Equal(t, 10, updatedCommission.RefundedCommissionQuota)
	assert.InDelta(t, 1.00, updatedCommission.RefundedRechargeAmount, 0.0001)
}

func TestMarkTopUpRefundFailedRestoresQuotaAndReferralCommission(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 0)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 100

	inviter := &User{Id: 174, Username: "refund-rollback-inviter", Status: common.UserStatusEnabled, AffCode: "refund-rollback-inviter-aff", AffQuota: 100, AffHistoryQuota: 100}
	invitee := &User{Id: 175, Username: "refund-rollback-invitee", Status: common.UserStatusEnabled, AffCode: "refund-rollback-invitee-aff", InviterId: 174, Quota: 1000}
	require.NoError(t, DB.Create(inviter).Error)
	require.NoError(t, DB.Create(invitee).Error)
	topUp := &TopUp{
		UserId:          175,
		Amount:          10,
		Money:           1.00,
		TradeNo:         "official-refund-rollback",
		PaymentMethod:   PaymentMethodAlipayOfficial,
		PaymentProvider: PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusSuccess,
		CompleteTime:    time.Now().Unix(),
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())
	require.NoError(t, DB.Create(&ReferralCommission{
		InviterId:       174,
		InviteeId:       175,
		SourceType:      ReferralCommissionSourceTopUp,
		SourceId:        topUp.Id,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RechargeAmount:  1.00,
		CommissionQuota: 10,
		CommissionRate:  10,
	}).Error)

	_, err := CreateOfficialPaymentRefund(OfficialPaymentRefundCreateParams{
		TradeNo:         topUp.TradeNo,
		PaymentProvider: PaymentProviderAlipayOfficial,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RefundAmount:    0.40,
		Reason:          "rollback",
		OutRequestNo:    "official-refund-rollback-rf-1",
	})
	require.NoError(t, err)
	require.NoError(t, MarkTopUpRefundFailed("official-refund-rollback-rf-1", "alipay failed"))

	updated := GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, updated)
	assert.Equal(t, common.TopUpStatusSuccess, updated.Status)
	assert.Equal(t, 0.0, updated.RefundedMoney)
	assert.Equal(t, int64(0), updated.RefundedQuota)
	assert.Equal(t, 1000, getUserQuotaForPaymentGuardTest(t, 175))

	var updatedInviter User
	require.NoError(t, DB.First(&updatedInviter, 174).Error)
	assert.Equal(t, 100, updatedInviter.AffQuota)
	assert.Equal(t, 100, updatedInviter.AffHistoryQuota)

	var updatedCommission ReferralCommission
	require.NoError(t, DB.Where("source_type = ? AND source_id = ?", ReferralCommissionSourceTopUp, topUp.Id).First(&updatedCommission).Error)
	assert.Equal(t, 10, updatedCommission.CommissionQuota)
	assert.Equal(t, 0, updatedCommission.RefundedCommissionQuota)
	assert.InDelta(t, 1.00, updatedCommission.RechargeAmount, 0.0001)
	assert.InDelta(t, 0.00, updatedCommission.RefundedRechargeAmount, 0.0001)
}

func TestMarkTopUpRefundFailedCanRollbackSuccessfulRefundAfterDepositbackFailure(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 0)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 100

	inviter := &User{Id: 186, Username: "depositback-rollback-inviter", Status: common.UserStatusEnabled, AffCode: "depositback-rollback-inviter-aff", AffQuota: 100, AffHistoryQuota: 100}
	invitee := &User{Id: 187, Username: "depositback-rollback-invitee", Status: common.UserStatusEnabled, AffCode: "depositback-rollback-invitee-aff", InviterId: 186, Quota: 1000}
	require.NoError(t, DB.Create(inviter).Error)
	require.NoError(t, DB.Create(invitee).Error)
	topUp := &TopUp{
		UserId:          187,
		Amount:          10,
		Money:           1.00,
		TradeNo:         "official-refund-depositback-rollback",
		PaymentMethod:   PaymentMethodAlipayOfficial,
		PaymentProvider: PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusSuccess,
		CompleteTime:    time.Now().Unix(),
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())
	require.NoError(t, DB.Create(&ReferralCommission{
		InviterId:       186,
		InviteeId:       187,
		SourceType:      ReferralCommissionSourceTopUp,
		SourceId:        topUp.Id,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RechargeAmount:  1.00,
		CommissionQuota: 10,
		CommissionRate:  10,
	}).Error)

	_, err := CreateOfficialPaymentRefund(OfficialPaymentRefundCreateParams{
		TradeNo:         topUp.TradeNo,
		PaymentProvider: PaymentProviderAlipayOfficial,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RefundAmount:    0.40,
		Reason:          "depositback rollback",
		OutRequestNo:    "official-refund-depositback-rollback-rf-1",
	})
	require.NoError(t, err)
	require.NoError(t, MarkTopUpRefundSuccess("official-refund-depositback-rollback-rf-1", "202605150000000000", `{"fund_change":"Y"}`))

	require.NoError(t, MarkTopUpRefundFailed("official-refund-depositback-rollback-rf-1", `{"dback_status":"F"}`))

	updated := GetTopUpByTradeNo(topUp.TradeNo)
	require.NotNil(t, updated)
	assert.Equal(t, common.TopUpStatusSuccess, updated.Status)
	assert.Equal(t, 0.0, updated.RefundedMoney)
	assert.Equal(t, int64(0), updated.RefundedQuota)
	assert.Equal(t, 1000, getUserQuotaForPaymentGuardTest(t, 187))

	refund := GetTopUpRefundByOutRequestNo("official-refund-depositback-rollback-rf-1")
	require.NotNil(t, refund)
	assert.Equal(t, TopUpRefundStatusFailed, refund.Status)

	var updatedInviter User
	require.NoError(t, DB.First(&updatedInviter, 186).Error)
	assert.Equal(t, 100, updatedInviter.AffQuota)
	assert.Equal(t, 100, updatedInviter.AffHistoryQuota)
}

func TestFullRefundReversesUnlockedInviterRewardAndPaymentState(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 0)

	originalQuotaPerUnit := common.QuotaPerUnit
	originalInviterRewardAfterPayment := common.InviterRewardAfterPaymentEnabled
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
		common.InviterRewardAfterPaymentEnabled = originalInviterRewardAfterPayment
	})
	common.QuotaPerUnit = 100
	common.InviterRewardAfterPaymentEnabled = true

	inviter := &User{
		Id:              176,
		Username:        "refund-reward-inviter",
		Status:          common.UserStatusEnabled,
		AffCode:         "refund-reward-inviter-aff",
		AffQuota:        130,
		AffHistoryQuota: 130,
	}
	invitee := &User{
		Id:                             177,
		Username:                       "refund-reward-invitee",
		Status:                         common.UserStatusEnabled,
		AffCode:                        "refund-reward-invitee-aff",
		InviterId:                      176,
		Quota:                          1000,
		InviterRewardQuota:             30,
		InviterRewardUnlocked:          true,
		InviterRewardUnlockedByPayment: true,
	}
	require.NoError(t, DB.Create(inviter).Error)
	require.NoError(t, DB.Create(invitee).Error)
	topUp := &TopUp{
		UserId:          177,
		Amount:          10,
		Money:           1.00,
		TradeNo:         "official-refund-reward",
		PaymentMethod:   PaymentMethodAlipayOfficial,
		PaymentProvider: PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusSuccess,
		CompleteTime:    time.Now().Unix(),
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())
	require.NoError(t, DB.Create(&ReferralCommission{
		InviterId:       176,
		InviteeId:       177,
		SourceType:      ReferralCommissionSourceTopUp,
		SourceId:        topUp.Id,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RechargeAmount:  1.00,
		CommissionQuota: 10,
		CommissionRate:  10,
	}).Error)

	_, err := CreateOfficialPaymentRefund(OfficialPaymentRefundCreateParams{
		TradeNo:         topUp.TradeNo,
		PaymentProvider: PaymentProviderAlipayOfficial,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RefundAmount:    1.00,
		Reason:          "full refund reward",
		OutRequestNo:    "official-refund-reward-rf-1",
	})
	require.NoError(t, err)

	var updatedInviter User
	require.NoError(t, DB.First(&updatedInviter, 176).Error)
	assert.Equal(t, 90, updatedInviter.AffQuota)
	assert.Equal(t, 90, updatedInviter.AffHistoryQuota)

	var updatedCommission ReferralCommission
	require.NoError(t, DB.Where("source_type = ? AND source_id = ?", ReferralCommissionSourceTopUp, topUp.Id).First(&updatedCommission).Error)
	assert.Equal(t, 10, updatedCommission.CommissionQuota)
	assert.Equal(t, 10, updatedCommission.RefundedCommissionQuota)
	assert.InDelta(t, 1.00, updatedCommission.RechargeAmount, 0.0001)
	assert.InDelta(t, 1.00, updatedCommission.RefundedRechargeAmount, 0.0001)

	var updatedInvitee User
	require.NoError(t, DB.First(&updatedInvitee, 177).Error)
	assert.False(t, updatedInvitee.InviterRewardUnlocked)

	records, total, err := GetAdminReferralRecords(&AdminReferralQuery{PageInfo: &common.PageInfo{Page: 1, PageSize: 20}})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, records, 1)
	assert.False(t, records[0].InviteeHasPaid)
	assert.Equal(t, int64(0), records[0].FirstPaymentTime)
	assert.Equal(t, 0, records[0].TotalCommissionQuota)
	assert.InDelta(t, 0.0, records[0].TotalRechargeAmount, 0.0001)
}

func TestFullRefundFailureRestoresUnlockedInviterReward(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 0)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 100

	inviter := &User{
		Id:              178,
		Username:        "refund-failure-reward-inviter",
		Status:          common.UserStatusEnabled,
		AffCode:         "refund-failure-reward-inviter-aff",
		AffQuota:        130,
		AffHistoryQuota: 130,
	}
	invitee := &User{
		Id:                             179,
		Username:                       "refund-failure-reward-invitee",
		Status:                         common.UserStatusEnabled,
		AffCode:                        "refund-failure-reward-invitee-aff",
		InviterId:                      178,
		Quota:                          1000,
		InviterRewardQuota:             30,
		InviterRewardUnlocked:          true,
		InviterRewardUnlockedByPayment: true,
	}
	require.NoError(t, DB.Create(inviter).Error)
	require.NoError(t, DB.Create(invitee).Error)
	topUp := &TopUp{
		UserId:          179,
		Amount:          10,
		Money:           1.00,
		TradeNo:         "official-refund-failure-reward",
		PaymentMethod:   PaymentMethodAlipayOfficial,
		PaymentProvider: PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusSuccess,
		CompleteTime:    time.Now().Unix(),
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())
	require.NoError(t, DB.Create(&ReferralCommission{
		InviterId:       178,
		InviteeId:       179,
		SourceType:      ReferralCommissionSourceTopUp,
		SourceId:        topUp.Id,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RechargeAmount:  1.00,
		CommissionQuota: 10,
		CommissionRate:  10,
	}).Error)

	_, err := CreateOfficialPaymentRefund(OfficialPaymentRefundCreateParams{
		TradeNo:         topUp.TradeNo,
		PaymentProvider: PaymentProviderAlipayOfficial,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RefundAmount:    1.00,
		Reason:          "full refund failure reward",
		OutRequestNo:    "official-refund-failure-reward-rf-1",
	})
	require.NoError(t, err)
	require.NoError(t, MarkTopUpRefundFailed("official-refund-failure-reward-rf-1", "alipay failed"))

	var updatedInviter User
	require.NoError(t, DB.First(&updatedInviter, 178).Error)
	assert.Equal(t, 130, updatedInviter.AffQuota)
	assert.Equal(t, 130, updatedInviter.AffHistoryQuota)

	var updatedInvitee User
	require.NoError(t, DB.First(&updatedInvitee, 179).Error)
	assert.True(t, updatedInvitee.InviterRewardUnlocked)
	assert.True(t, updatedInvitee.InviterRewardUnlockedByPayment)
}

func TestFullRefundDoesNotReverseImmediateInviterReward(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 0)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 100

	inviter := &User{
		Id:              180,
		Username:        "refund-immediate-inviter",
		Status:          common.UserStatusEnabled,
		AffCode:         "refund-immediate-inviter-aff",
		AffQuota:        130,
		AffHistoryQuota: 130,
	}
	invitee := &User{
		Id:                    181,
		Username:              "refund-immediate-invitee",
		Status:                common.UserStatusEnabled,
		AffCode:               "refund-immediate-invitee-aff",
		InviterId:             180,
		Quota:                 1000,
		InviterRewardQuota:    30,
		InviterRewardUnlocked: true,
	}
	require.NoError(t, DB.Create(inviter).Error)
	require.NoError(t, DB.Create(invitee).Error)
	topUp := &TopUp{
		UserId:          181,
		Amount:          10,
		Money:           1.00,
		TradeNo:         "official-refund-immediate",
		PaymentMethod:   PaymentMethodAlipayOfficial,
		PaymentProvider: PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusSuccess,
		CompleteTime:    time.Now().Unix(),
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())
	require.NoError(t, DB.Create(&ReferralCommission{
		InviterId:       180,
		InviteeId:       181,
		SourceType:      ReferralCommissionSourceTopUp,
		SourceId:        topUp.Id,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RechargeAmount:  1.00,
		CommissionQuota: 10,
		CommissionRate:  10,
	}).Error)

	_, err := CreateOfficialPaymentRefund(OfficialPaymentRefundCreateParams{
		TradeNo:         topUp.TradeNo,
		PaymentProvider: PaymentProviderAlipayOfficial,
		PaymentMethod:   PaymentMethodAlipayOfficial,
		RefundAmount:    1.00,
		Reason:          "full refund immediate reward",
		OutRequestNo:    "official-refund-immediate-rf-1",
	})
	require.NoError(t, err)

	var updatedInviter User
	require.NoError(t, DB.First(&updatedInviter, 180).Error)
	assert.Equal(t, 120, updatedInviter.AffQuota)
	assert.Equal(t, 120, updatedInviter.AffHistoryQuota)

	var updatedInvitee User
	require.NoError(t, DB.First(&updatedInvitee, 181).Error)
	assert.True(t, updatedInvitee.InviterRewardUnlocked)
	assert.False(t, updatedInvitee.InviterRewardUnlockedByPayment)
}

func TestSearchAllTopUpsMatchesUserIdUsernameAndTradeNo(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{Id: 180, Username: "alice-topup", Status: common.UserStatusEnabled, AffCode: "alice-topup-aff"}).Error)
	require.NoError(t, DB.Create(&User{Id: 181, Username: "bob-topup", Status: common.UserStatusEnabled, AffCode: "bob-topup-aff"}).Error)
	require.NoError(t, DB.Create(&[]TopUp{
		{
			UserId:          180,
			Amount:          1,
			Money:           1,
			TradeNo:         "ALIPAY_SEARCH_A",
			PaymentMethod:   PaymentMethodAlipayOfficial,
			PaymentProvider: PaymentProviderAlipayOfficial,
			Status:          common.TopUpStatusPending,
			CreateTime:      time.Now().Unix(),
		},
		{
			UserId:          181,
			Amount:          1,
			Money:           1,
			TradeNo:         "ALIPAY_SEARCH_B",
			PaymentMethod:   PaymentMethodAlipayOfficial,
			PaymentProvider: PaymentProviderAlipayOfficial,
			Status:          common.TopUpStatusPending,
			CreateTime:      time.Now().Unix(),
		},
	}).Error)

	rows, total, err := SearchAllTopUps("alice", &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "alice-topup", rows[0].Username)

	rows, total, err = SearchAllTopUps("181", &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "bob-topup", rows[0].Username)

	rows, total, err = SearchAllTopUps("SEARCH_A", &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "ALIPAY_SEARCH_A", rows[0].TradeNo)
}

func TestUpdatePendingTopUpStatus_RejectsMismatchedPaymentProvider(t *testing.T) {
	testCases := []struct {
		name                    string
		tradeNo                 string
		storedPaymentProvider   string
		expectedPaymentProvider string
		targetStatus            string
	}{
		{
			name:                    "stripe expire",
			tradeNo:                 "stripe-expire-guard",
			storedPaymentProvider:   PaymentProviderCreem,
			expectedPaymentProvider: PaymentProviderStripe,
			targetStatus:            common.TopUpStatusExpired,
		},
		{
			name:                    "waffo failed",
			tradeNo:                 "waffo-failed-guard",
			storedPaymentProvider:   PaymentProviderStripe,
			expectedPaymentProvider: PaymentProviderWaffo,
			targetStatus:            common.TopUpStatusFailed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			truncateTables(t)
			insertUserForPaymentGuardTest(t, 150, 0)
			insertTopUpForPaymentGuardTest(t, tc.tradeNo, 150, tc.storedPaymentProvider)

			err := UpdatePendingTopUpStatus(tc.tradeNo, tc.expectedPaymentProvider, tc.targetStatus)
			require.ErrorIs(t, err, ErrPaymentMethodMismatch)
			assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, tc.tradeNo))
		})
	}
}

func TestCompleteEpayTopUp_ReturnsOrderOnMismatchedPaymentProvider(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 160, 0)
	insertTopUpForPaymentGuardTest(t, "epay-provider-guard", 160, PaymentProviderStripe)

	topUp, quotaToAdd, referralResult, completed, err := CompleteEpayTopUp("epay-provider-guard", "alipay")
	require.ErrorIs(t, err, ErrPaymentMethodMismatch)
	require.NotNil(t, topUp)
	assert.Equal(t, PaymentProviderStripe, topUp.PaymentProvider)
	assert.Equal(t, 0, quotaToAdd)
	assert.Nil(t, referralResult)
	assert.False(t, completed)
	assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, "epay-provider-guard"))
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 160))
}

func TestCompleteSubscriptionOrder_RejectsMismatchedPaymentProvider(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 202, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 301)
	insertSubscriptionOrderForPaymentGuardTest(t, "sub-guard-order", 202, plan.Id, PaymentProviderStripe)

	err := CompleteSubscriptionOrder("sub-guard-order", `{"provider":"epay"}`, PaymentProviderEpay, "alipay")
	require.ErrorIs(t, err, ErrPaymentMethodMismatch)

	order := GetSubscriptionOrderByTradeNo("sub-guard-order")
	require.NotNil(t, order)
	assert.Equal(t, common.TopUpStatusPending, order.Status)
	assert.Zero(t, countUserSubscriptionsForPaymentGuardTest(t, 202))

	topUp := GetTopUpByTradeNo("sub-guard-order")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusPending, topUp.Status)
	assert.Equal(t, PaymentProviderStripe, topUp.PaymentProvider)
	assert.Equal(t, int64(0), topUp.Amount)
}

func TestSubscriptionOrderInsertCreatesPendingTopUpForBillQuery(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 203, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 302)
	insertSubscriptionOrderForPaymentGuardTest(t, "sub-official-provider", 203, plan.Id, PaymentProviderAlipayOfficial)

	topUp := GetTopUpByTradeNo("sub-official-provider")
	require.NotNil(t, topUp)
	assert.Equal(t, 203, topUp.UserId)
	assert.Equal(t, int64(0), topUp.Amount)
	assert.InDelta(t, 9.99, topUp.Money, 0.0001)
	assert.Equal(t, PaymentProviderAlipayOfficial, topUp.PaymentProvider)
	assert.Equal(t, common.TopUpStatusPending, topUp.Status)
	assert.Zero(t, topUp.CompleteTime)
}

func TestCompleteSubscriptionOrder_PersistsPaymentProviderToTopUp(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 204, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 303)
	insertSubscriptionOrderForPaymentGuardTest(t, "sub-official-provider-success", 204, plan.Id, PaymentProviderAlipayOfficial)

	err := CompleteSubscriptionOrder("sub-official-provider-success", `{"provider":"alipay_official"}`, PaymentProviderAlipayOfficial, PaymentMethodAlipayOfficial)
	require.NoError(t, err)

	topUp := GetTopUpByTradeNo("sub-official-provider-success")
	require.NotNil(t, topUp)
	assert.Equal(t, PaymentMethodAlipayOfficial, topUp.PaymentMethod)
	assert.Equal(t, PaymentProviderAlipayOfficial, topUp.PaymentProvider)
	assert.Equal(t, common.TopUpStatusSuccess, topUp.Status)
	assert.NotZero(t, topUp.CompleteTime)
}

func TestCompleteSubscriptionOrderAllowsEpayActualPaymentMethod(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 206, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 306)
	insertSubscriptionOrderForPaymentGuardTest(t, "sub-epay-actual-method", 206, plan.Id, PaymentProviderEpay)

	err := CompleteSubscriptionOrder("sub-epay-actual-method", `{"type":"alipay"}`, PaymentProviderEpay, "alipay")
	require.NoError(t, err)

	topUp := GetTopUpByTradeNo("sub-epay-actual-method")
	require.NotNil(t, topUp)
	assert.Equal(t, "alipay", topUp.PaymentMethod)
	assert.Equal(t, PaymentProviderEpay, topUp.PaymentProvider)
	assert.Equal(t, common.TopUpStatusSuccess, topUp.Status)
}

func TestExpireSubscriptionOrderSyncsPendingTopUpStatus(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 205, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 304)
	insertSubscriptionOrderForPaymentGuardTest(t, "sub-expire-sync", 205, plan.Id, PaymentProviderAlipayOfficial)

	err := ExpireSubscriptionOrder("sub-expire-sync", PaymentProviderAlipayOfficial)
	require.NoError(t, err)

	order := GetSubscriptionOrderByTradeNo("sub-expire-sync")
	require.NotNil(t, order)
	assert.Equal(t, common.TopUpStatusExpired, order.Status)

	topUp := GetTopUpByTradeNo("sub-expire-sync")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusExpired, topUp.Status)
	assert.Equal(t, PaymentProviderAlipayOfficial, topUp.PaymentProvider)
	assert.NotZero(t, topUp.CompleteTime)
}

func TestExpireSubscriptionOrder_RejectsMismatchedPaymentProvider(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 305, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 401)
	insertSubscriptionOrderForPaymentGuardTest(t, "sub-expire-guard", 305, plan.Id, PaymentProviderStripe)

	err := ExpireSubscriptionOrder("sub-expire-guard", PaymentProviderCreem)
	require.ErrorIs(t, err, ErrPaymentMethodMismatch)

	order := GetSubscriptionOrderByTradeNo("sub-expire-guard")
	require.NotNil(t, order)
	assert.Equal(t, common.TopUpStatusPending, order.Status)

	topUp := GetTopUpByTradeNo("sub-expire-guard")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusPending, topUp.Status)
}
