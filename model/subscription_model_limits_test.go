package model

import (
	"strings"
	"testing"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func insertSubscriptionPlanForModelLimitTest(t *testing.T, id int, totalAmount int64, modelLimitsEnabled bool, modelLimits string) *SubscriptionPlan {
	t.Helper()
	plan := &SubscriptionPlan{
		Id:                 id,
		Title:              "Model Limit Plan",
		PriceAmount:        9.99,
		Currency:           "USD",
		DurationUnit:       SubscriptionDurationMonth,
		DurationValue:      1,
		Enabled:            true,
		TotalAmount:        totalAmount,
		ModelLimitsEnabled: modelLimitsEnabled,
		ModelLimits:        modelLimits,
	}
	require.NoError(t, DB.Create(plan).Error)
	return plan
}

func insertUserSubscriptionForModelLimitTest(t *testing.T, id int, userId int, planId int, amountTotal int64, amountUsed int64, endTime int64) {
	t.Helper()
	sub := &UserSubscription{
		Id:          id,
		UserId:      userId,
		PlanId:      planId,
		AmountTotal: amountTotal,
		AmountUsed:  amountUsed,
		StartTime:   time.Now().Unix() - 60,
		EndTime:     endTime,
		Status:      "active",
		Source:      "admin",
	}
	require.NoError(t, DB.Create(sub).Error)
}

func TestSubscriptionPlanModelLimitsNormalizeAndDeduplicate(t *testing.T) {
	plan := &SubscriptionPlan{
		ModelLimitsEnabled: true,
		ModelLimits:        " gpt-4o,claude-3-5, gpt-4o ,, claude-3-5 ",
	}

	assert.Equal(t, []string{"gpt-4o", "claude-3-5"}, plan.GetModelLimits())
	assert.True(t, plan.IsModelAllowed("gpt-4o"))
	assert.False(t, plan.IsModelAllowed("gemini-1.5-pro"))
}

func TestSubscriptionPlanModelLimitsDisabledIgnoresStoredCSV(t *testing.T) {
	plan := &SubscriptionPlan{
		ModelLimitsEnabled: false,
		ModelLimits:        "gpt-4o",
	}

	assert.Empty(t, plan.GetModelLimits())
	assert.True(t, plan.IsModelAllowed("gemini-1.5-pro"))
}

func TestSubscriptionPlanModelLimitsEnabledWithEmptyListAllowsAll(t *testing.T) {
	plan := &SubscriptionPlan{
		ModelLimitsEnabled: true,
		ModelLimits:        " , , ",
	}

	assert.Empty(t, plan.GetModelLimits())
	assert.True(t, plan.IsModelAllowed("gemini-1.5-pro"))
}

func TestPreConsumeUserSubscriptionSkipsDisallowedPlanAndUsesAllowedPlan(t *testing.T) {
	truncateTables(t)

	now := time.Now().Unix()
	insertSubscriptionPlanForModelLimitTest(t, 1001, 100, true, "gpt-4o")
	insertSubscriptionPlanForModelLimitTest(t, 1002, 100, true, "claude-3-5-sonnet")
	insertUserSubscriptionForModelLimitTest(t, 2001, 3001, 1001, 100, 0, now+3600)
	insertUserSubscriptionForModelLimitTest(t, 2002, 3001, 1002, 100, 0, now+7200)

	result, err := PreConsumeUserSubscription("model-limit-allowed", 3001, "claude-3-5-sonnet", 0, 10)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2002, result.UserSubscriptionId)
	assert.EqualValues(t, 10, result.AmountUsedAfter)
}

func TestPreConsumeUserSubscriptionReturnsModelLimitErrorBeforeQuotaError(t *testing.T) {
	truncateTables(t)

	now := time.Now().Unix()
	insertSubscriptionPlanForModelLimitTest(t, 1003, 100, true, "gpt-4o")
	insertUserSubscriptionForModelLimitTest(t, 2003, 3002, 1003, 100, 0, now+3600)

	result, err := PreConsumeUserSubscription("model-limit-denied", 3002, "gemini-1.5-pro", 0, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, strings.Contains(err.Error(), "no subscription allows model gemini-1.5-pro"), err.Error())
}

func TestRefundSubscriptionPreConsumeRefundsQuotaAndMarksRecordOnce(t *testing.T) {
	truncateTables(t)

	now := time.Now().Unix()
	insertSubscriptionPlanForModelLimitTest(t, 1004, 100, false, "")
	insertUserSubscriptionForModelLimitTest(t, 2004, 3004, 1004, 100, 30, now+3600)
	require.NoError(t, DB.Create(&SubscriptionPreConsumeRecord{
		RequestId:          "refund-same-tx",
		UserId:             3004,
		UserSubscriptionId: 2004,
		PreConsumed:        30,
		Status:             "consumed",
	}).Error)

	err := RefundSubscriptionPreConsume("refund-same-tx")

	require.NoError(t, err)

	var sub UserSubscription
	require.NoError(t, DB.First(&sub, 2004).Error)
	assert.EqualValues(t, 0, sub.AmountUsed)

	var record SubscriptionPreConsumeRecord
	require.NoError(t, DB.Where("request_id = ?", "refund-same-tx").First(&record).Error)
	assert.Equal(t, "refunded", record.Status)

	require.NoError(t, RefundSubscriptionPreConsume("refund-same-tx"))
	require.NoError(t, DB.First(&sub, 2004).Error)
	assert.EqualValues(t, 0, sub.AmountUsed)
}

func TestCreateUserSubscriptionFromPlanTxLocksUserBeforeMaxPurchaseCheck(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{Id: 3005, Username: "max-purchase-lock-user", Status: common.UserStatusEnabled}).Error)
	plan := insertSubscriptionPlanForModelLimitTest(t, 1005, 100, false, "")
	plan.MaxPurchasePerUser = 1

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		_, err := CreateUserSubscriptionFromPlanTx(tx, 3005, plan, "order")
		return err
	}))

	err := DB.Transaction(func(tx *gorm.DB) error {
		_, err := CreateUserSubscriptionFromPlanTx(tx, 3005, plan, "order")
		return err
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "已达到该套餐购买上限")
}

func TestSubscriptionOrderInsertCountsPendingOrdersAgainstMaxPurchase(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{Id: 3006, Username: "max-purchase-pending-user", Status: common.UserStatusEnabled}).Error)
	plan := insertSubscriptionPlanForModelLimitTest(t, 1006, 100, false, "")
	plan.MaxPurchasePerUser = 1
	require.NoError(t, DB.Save(plan).Error)

	first := &SubscriptionOrder{
		UserId:          3006,
		PlanId:          plan.Id,
		Money:           9.99,
		TradeNo:         "pending-max-purchase-1",
		PaymentMethod:   PaymentMethodAlipayOfficial,
		PaymentProvider: PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
	}
	require.NoError(t, first.Insert())

	second := &SubscriptionOrder{
		UserId:          3006,
		PlanId:          plan.Id,
		Money:           9.99,
		TradeNo:         "pending-max-purchase-2",
		PaymentMethod:   PaymentMethodWechatPayOfficial,
		PaymentProvider: PaymentProviderWechatPayOfficial,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
	}
	err := second.Insert()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "已达到该套餐购买上限")
}

func TestSubscriptionOrderInsertAllowsExpiredPendingOrderReplacement(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{Id: 3007, Username: "max-purchase-expired-user", Status: common.UserStatusEnabled}).Error)
	plan := insertSubscriptionPlanForModelLimitTest(t, 1007, 100, false, "")
	plan.MaxPurchasePerUser = 1
	require.NoError(t, DB.Save(plan).Error)
	require.NoError(t, DB.Create(&SubscriptionOrder{
		UserId:          3007,
		PlanId:          plan.Id,
		Money:           9.99,
		TradeNo:         "expired-max-purchase",
		PaymentMethod:   PaymentMethodAlipayOfficial,
		PaymentProvider: PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusExpired,
		CreateTime:      common.GetTimestamp(),
	}).Error)

	order := &SubscriptionOrder{
		UserId:          3007,
		PlanId:          plan.Id,
		Money:           9.99,
		TradeNo:         "replacement-max-purchase",
		PaymentMethod:   PaymentMethodWechatPayOfficial,
		PaymentProvider: PaymentProviderWechatPayOfficial,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
	}

	require.NoError(t, order.Insert())
}

func TestPurchaseSubscriptionWithBalanceDeductsQuotaAndCreatesSubscriptionOrder(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 0)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 1000
	require.NoError(t, DB.Create(&User{
		Id:        3008,
		Username:  "balance-subscription-user",
		Status:    common.UserStatusEnabled,
		Quota:     2500,
		Group:     "default",
		InviterId: 99,
		AffCode:   "balance-subscription-user",
	}).Error)
	require.NoError(t, DB.Create(&User{
		Id:       99,
		Username: "balance-subscription-inviter",
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  "balance-subscription-inviter",
	}).Error)
	plan := insertSubscriptionPlanForModelLimitTest(t, 1008, 100, false, "")
	plan.PriceAmount = 1.25
	plan.UpgradeGroup = "vip"
	require.NoError(t, DB.Save(plan).Error)

	err := PurchaseSubscriptionWithBalance(3008, plan.Id)

	require.NoError(t, err)
	var user User
	require.NoError(t, DB.Where("id = ?", 3008).First(&user).Error)
	assert.Equal(t, 1250, user.Quota)
	assert.Equal(t, "vip", user.Group)

	var sub UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", 3008, plan.Id).First(&sub).Error)
	assert.Equal(t, "active", sub.Status)
	assert.Equal(t, "order", sub.Source)
	assert.Equal(t, "vip", sub.UpgradeGroup)
	assert.Equal(t, "default", sub.PrevUserGroup)

	var order SubscriptionOrder
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", 3008, plan.Id).First(&order).Error)
	assert.Equal(t, PaymentMethodBalance, order.PaymentMethod)
	assert.Equal(t, PaymentProviderBalance, order.PaymentProvider)
	assert.Equal(t, common.TopUpStatusSuccess, order.Status)
	assert.Contains(t, order.ProviderPayload, "charged_quota=1250")

	var topUp TopUp
	require.NoError(t, DB.Where("trade_no = ?", order.TradeNo).First(&topUp).Error)
	assert.Equal(t, PaymentMethodBalance, topUp.PaymentMethod)
	assert.Equal(t, PaymentProviderBalance, topUp.PaymentProvider)
	assert.Equal(t, common.TopUpStatusSuccess, topUp.Status)
	assert.Equal(t, order.Money, topUp.Money)
	assert.Regexp(t, `^BALANCE__3008_[A-Za-z0-9]+$`, topUp.TradeNo)

	var commissionCount int64
	require.NoError(t, DB.Model(&ReferralCommission{}).Count(&commissionCount).Error)
	assert.EqualValues(t, 0, commissionCount)
}

func TestTopUpListIncludesSubscriptionPlanTitle(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       3018,
		Username: "subscription-title-billing-user",
		Status:   common.UserStatusEnabled,
		Quota:    1000,
	}).Error)
	plan := insertSubscriptionPlanForModelLimitTest(t, 1018, 1000, false, "")
	plan.Title = "GPT10元/日周卡 · 订阅 #12"
	plan.PriceAmount = 10
	require.NoError(t, DB.Save(plan).Error)
	InvalidateSubscriptionPlanCache(plan.Id)
	now := common.GetTimestamp()
	order := &SubscriptionOrder{
		UserId:          3018,
		PlanId:          plan.Id,
		Money:           10,
		TradeNo:         "WXSUB_TITLE_TEST",
		PaymentMethod:   PaymentMethodWechatPayOfficial,
		PaymentProvider: PaymentProviderWechatPayOfficial,
		Status:          common.TopUpStatusSuccess,
		CreateTime:      now,
		CompleteTime:    now,
	}
	require.NoError(t, order.Insert())

	result, err := GetUserTopUpsResultWithOptions(3018, TopUpQueryOptions{}, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, plan.Id, result.Items[0].SubscriptionPlanId)
	assert.Equal(t, plan.Title, result.Items[0].SubscriptionPlanTitle)
}

func TestPurchaseSubscriptionWithBalanceExtendsSamePlanAndDefersFutureUse(t *testing.T) {
	truncateTables(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 1000
	require.NoError(t, DB.Create(&User{
		Id:       3014,
		Username: "balance-subscription-renew-user",
		Status:   common.UserStatusEnabled,
		Quota:    3000,
		Group:    "default",
		AffCode:  "balance-subscription-renew-user",
	}).Error)
	plan := insertSubscriptionPlanForModelLimitTest(t, 1014, 1000, false, "")
	plan.PriceAmount = 1
	plan.DurationUnit = SubscriptionDurationCustom
	plan.CustomSeconds = int64(time.Hour / time.Second)
	require.NoError(t, DB.Save(plan).Error)
	InvalidateSubscriptionPlanCache(plan.Id)

	require.NoError(t, PurchaseSubscriptionWithBalance(3014, plan.Id))
	require.NoError(t, PurchaseSubscriptionWithBalance(3014, plan.Id))

	var subs []UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", 3014, plan.Id).
		Order("start_time asc, id asc").
		Find(&subs).Error)
	require.Len(t, subs, 2)
	assert.Equal(t, subs[0].EndTime, subs[1].StartTime)
	assert.Equal(t, subs[1].StartTime+int64(time.Hour/time.Second), subs[1].EndTime)

	activeSubs, err := GetAllActiveUserSubscriptions(3014)
	require.NoError(t, err)
	require.Len(t, activeSubs, 1)
	assert.Equal(t, subs[0].Id, activeSubs[0].Subscription.Id)

	hasActive, err := HasActiveUserSubscription(3014)
	require.NoError(t, err)
	assert.True(t, hasActive)

	result, err := PreConsumeUserSubscription("balance-renew-current-only", 3014, "", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, subs[0].Id, result.UserSubscriptionId)

	var futureSub UserSubscription
	require.NoError(t, DB.Where("id = ?", subs[1].Id).First(&futureSub).Error)
	assert.EqualValues(t, 0, futureSub.AmountUsed)
}

func TestBalanceSubscriptionRefundTargetsOrderSubscriptionFromPayload(t *testing.T) {
	truncateTables(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 1000
	require.NoError(t, DB.Create(&User{
		Id:       3015,
		Username: "balance-subscription-refund-target-user",
		Status:   common.UserStatusEnabled,
		Quota:    3000,
		Group:    "default",
		AffCode:  "balance-subscription-refund-target-user",
	}).Error)
	plan := insertSubscriptionPlanForModelLimitTest(t, 1015, 1000, false, "")
	plan.PriceAmount = 1
	plan.DurationUnit = SubscriptionDurationCustom
	plan.CustomSeconds = int64(time.Hour / time.Second)
	require.NoError(t, DB.Save(plan).Error)
	InvalidateSubscriptionPlanCache(plan.Id)

	require.NoError(t, PurchaseSubscriptionWithBalance(3015, plan.Id))
	require.NoError(t, PurchaseSubscriptionWithBalance(3015, plan.Id))

	var orders []SubscriptionOrder
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", 3015, plan.Id).
		Order("id asc").
		Find(&orders).Error)
	require.Len(t, orders, 2)
	var subs []UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", 3015, plan.Id).
		Order("id asc").
		Find(&subs).Error)
	require.Len(t, subs, 2)

	firstPayloadSubId, ok := providerPayloadSubscriptionID(orders[0].ProviderPayload)
	require.True(t, ok)
	secondPayloadSubId, ok := providerPayloadSubscriptionID(orders[1].ProviderPayload)
	require.True(t, ok)
	assert.Equal(t, subs[0].Id, firstPayloadSubId)
	assert.Equal(t, subs[1].Id, secondPayloadSubId)

	refund, err := CreateBalanceSubscriptionRefund(orders[1].TradeNo, 1, "refund renewed segment", true)
	require.NoError(t, err)
	require.NotNil(t, refund)
	assert.EqualValues(t, 1000, refund.RefundQuota)
	assert.Equal(t, 2000, getUserQuotaForPaymentGuardTest(t, 3015))

	var firstSub UserSubscription
	require.NoError(t, DB.Where("id = ?", subs[0].Id).First(&firstSub).Error)
	assert.Equal(t, "active", firstSub.Status)
	var secondSub UserSubscription
	require.NoError(t, DB.Where("id = ?", subs[1].Id).First(&secondSub).Error)
	assert.Equal(t, "cancelled", secondSub.Status)
}

func TestRenewedSubscriptionUpgradeGroupAppliesWhenStarted(t *testing.T) {
	truncateTables(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 1000
	require.NoError(t, DB.Create(&User{
		Id:       3017,
		Username: "balance-subscription-renew-group-user",
		Status:   common.UserStatusEnabled,
		Quota:    3000,
		Group:    "default",
		AffCode:  "balance-subscription-renew-group-user",
	}).Error)
	plan := insertSubscriptionPlanForModelLimitTest(t, 1017, 1000, false, "")
	plan.PriceAmount = 1
	plan.UpgradeGroup = "vip"
	plan.DurationUnit = SubscriptionDurationCustom
	plan.CustomSeconds = int64(time.Hour / time.Second)
	require.NoError(t, DB.Save(plan).Error)
	InvalidateSubscriptionPlanCache(plan.Id)

	require.NoError(t, PurchaseSubscriptionWithBalance(3017, plan.Id))
	require.NoError(t, PurchaseSubscriptionWithBalance(3017, plan.Id))

	var orders []SubscriptionOrder
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", 3017, plan.Id).
		Order("id asc").
		Find(&orders).Error)
	require.Len(t, orders, 2)
	var subs []UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", 3017, plan.Id).
		Order("start_time asc, id asc").
		Find(&subs).Error)
	require.Len(t, subs, 2)
	assert.Equal(t, "default", subs[0].PrevUserGroup)
	assert.Equal(t, "default", subs[1].PrevUserGroup)
	assert.Equal(t, "vip", getUserGroupForPaymentGuardTest(t, 3017))

	_, err := CreateBalanceSubscriptionRefund(orders[0].TradeNo, 1, "refund current segment", true)
	require.NoError(t, err)
	assert.Equal(t, "default", getUserGroupForPaymentGuardTest(t, 3017))

	now := common.GetTimestamp()
	require.NoError(t, DB.Model(&UserSubscription{}).
		Where("id = ?", subs[1].Id).
		Updates(map[string]interface{}{
			"start_time": now - 10,
			"end_time":   now + int64(time.Hour/time.Second),
		}).Error)
	applied, err := ApplyStartedSubscriptionUpgradeGroups(20)
	require.NoError(t, err)
	assert.Equal(t, 1, applied)
	assert.Equal(t, "vip", getUserGroupForPaymentGuardTest(t, 3017))
}

func TestBalanceSubscriptionRefundRequestCanBeCreated(t *testing.T) {
	truncateTables(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 1000
	require.NoError(t, DB.Create(&User{
		Id:       3016,
		Username: "balance-subscription-request-user",
		Status:   common.UserStatusEnabled,
		Quota:    2000,
		Group:    "default",
		AffCode:  "balance-subscription-request-user",
	}).Error)
	plan := insertSubscriptionPlanForModelLimitTest(t, 1016, 1000, false, "")
	plan.PriceAmount = 1
	require.NoError(t, DB.Save(plan).Error)
	InvalidateSubscriptionPlanCache(plan.Id)

	require.NoError(t, PurchaseSubscriptionWithBalance(3016, plan.Id))
	var order SubscriptionOrder
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", 3016, plan.Id).First(&order).Error)

	request, preview, err := CreateTopUpRefundRequest(3016, order.TradeNo, 0.5, "balance subscription refund request")
	require.NoError(t, err)
	require.NotNil(t, request)
	require.NotNil(t, preview)
	assert.True(t, preview.Refundable)
	assert.True(t, preview.IsSubscription)
	assert.Equal(t, TopUpRefundRequestStatusPending, request.Status)
	assert.Equal(t, PaymentProviderBalance, request.PaymentProvider)
}

func TestBalanceSubscriptionRefundPreviewAndPartialRefund(t *testing.T) {
	truncateTables(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 1000
	require.NoError(t, DB.Create(&User{
		Id:       3011,
		Username: "balance-subscription-refund-user",
		Status:   common.UserStatusEnabled,
		Quota:    3000,
		Group:    "default",
		AffCode:  "balance-subscription-refund-user",
	}).Error)
	plan := insertSubscriptionPlanForModelLimitTest(t, 1011, 1000, false, "")
	plan.PriceAmount = 2
	require.NoError(t, DB.Save(plan).Error)

	require.NoError(t, PurchaseSubscriptionWithBalance(3011, plan.Id))
	assert.Equal(t, 1000, getUserQuotaForPaymentGuardTest(t, 3011))

	var order SubscriptionOrder
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", 3011, plan.Id).First(&order).Error)
	require.NoError(t, DB.Model(&UserSubscription{}).
		Where("user_id = ? AND plan_id = ?", 3011, plan.Id).
		Update("amount_used", 250).Error)

	preview, err := CalculateOfficialPaymentRefundPreview(order.TradeNo)
	require.NoError(t, err)
	require.NotNil(t, preview)
	assert.True(t, preview.Refundable)
	assert.True(t, preview.IsSubscription)
	assert.Equal(t, order.Id, preview.SubscriptionOrderId)
	assert.InDelta(t, 1.50, preview.MaxRefundAmount, 0.0001)
	assert.EqualValues(t, 1500, preview.MaxRefundQuota)

	refund, err := CreateBalanceSubscriptionRefund(order.TradeNo, 1, "partial balance subscription refund", false)
	require.NoError(t, err)
	require.NotNil(t, refund)
	assert.Equal(t, TopUpRefundStatusSuccess, refund.Status)
	assert.Equal(t, PaymentProviderBalance, refund.PaymentProvider)
	assert.EqualValues(t, 1000, refund.RefundQuota)
	assert.Equal(t, 2000, getUserQuotaForPaymentGuardTest(t, 3011))

	topUp := GetTopUpByTradeNo(order.TradeNo)
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusPartialRefunded, topUp.Status)
	assert.InDelta(t, 1.0, topUp.RefundedMoney, 0.0001)
	assert.EqualValues(t, 1000, topUp.RefundedQuota)

	reloadedOrder := GetSubscriptionOrderByTradeNo(order.TradeNo)
	require.NotNil(t, reloadedOrder)
	assert.Equal(t, common.TopUpStatusPartialRefunded, reloadedOrder.Status)

	var sub UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", 3011, plan.Id).First(&sub).Error)
	assert.Equal(t, "cancelled", sub.Status)
}

func TestBalanceSubscriptionFullRefundReturnsRemainingQuotaAndCancelsSubscription(t *testing.T) {
	truncateTables(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 1000
	require.NoError(t, DB.Create(&User{
		Id:       3012,
		Username: "balance-subscription-full-refund-user",
		Status:   common.UserStatusEnabled,
		Quota:    3000,
		Group:    "default",
		AffCode:  "balance-subscription-full-refund-user",
	}).Error)
	plan := insertSubscriptionPlanForModelLimitTest(t, 1012, 1000, false, "")
	plan.PriceAmount = 2
	plan.UpgradeGroup = "vip"
	require.NoError(t, DB.Save(plan).Error)

	require.NoError(t, PurchaseSubscriptionWithBalance(3012, plan.Id))
	assert.Equal(t, 1000, getUserQuotaForPaymentGuardTest(t, 3012))

	var order SubscriptionOrder
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", 3012, plan.Id).First(&order).Error)

	refund, err := CreateBalanceSubscriptionRefund(order.TradeNo, 2, "full balance subscription refund", true)
	require.NoError(t, err)
	require.NotNil(t, refund)
	assert.EqualValues(t, 2000, refund.RefundQuota)
	assert.Equal(t, 3000, getUserQuotaForPaymentGuardTest(t, 3012))

	topUp := GetTopUpByTradeNo(order.TradeNo)
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusRefunded, topUp.Status)
	assert.InDelta(t, 2.0, topUp.RefundedMoney, 0.0001)
	assert.EqualValues(t, 2000, topUp.RefundedQuota)

	reloadedOrder := GetSubscriptionOrderByTradeNo(order.TradeNo)
	require.NotNil(t, reloadedOrder)
	assert.Equal(t, common.TopUpStatusRefunded, reloadedOrder.Status)

	var sub UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", 3012, plan.Id).First(&sub).Error)
	assert.Equal(t, "cancelled", sub.Status)
}

func TestBalancePlainTopUpRefundRequestIsRejected(t *testing.T) {
	truncateTables(t)

	require.NoError(t, DB.Create(&User{
		Id:       3013,
		Username: "balance-plain-refund-user",
		Status:   common.UserStatusEnabled,
		Quota:    1000,
	}).Error)
	require.NoError(t, DB.Create(&TopUp{
		UserId:          3013,
		Amount:          0,
		Money:           1,
		TradeNo:         "BALANCE_PLAIN_TOPUP",
		PaymentMethod:   PaymentMethodBalance,
		PaymentProvider: PaymentProviderBalance,
		Status:          common.TopUpStatusSuccess,
		CreateTime:      common.GetTimestamp(),
		CompleteTime:    common.GetTimestamp(),
	}).Error)

	_, _, err := CreateTopUpRefundRequest(3013, "BALANCE_PLAIN_TOPUP", 1, "not subscription")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "仅官方支付、自助支付或余额订阅订单支持退款申请")
}

func TestPurchaseSubscriptionWithBalanceRejectsDisabledBalancePayPlan(t *testing.T) {
	truncateTables(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 1000
	require.NoError(t, DB.Create(&User{
		Id:       3010,
		Username: "balance-subscription-disabled-user",
		Status:   common.UserStatusEnabled,
		Quota:    2500,
		Group:    "default",
		AffCode:  "balance-subscription-disabled-user",
	}).Error)

	plan := insertSubscriptionPlanForModelLimitTest(t, 1010, 100, false, "")
	plan.PriceAmount = 1.25
	plan.AllowBalancePay = common.GetPointer(false)
	require.NoError(t, DB.Save(plan).Error)
	InvalidateSubscriptionPlanCache(plan.Id)

	err := PurchaseSubscriptionWithBalance(3010, plan.Id)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "不允许使用余额兑换")
	assert.Equal(t, 2500, getUserQuotaForPaymentGuardTest(t, 3010))

	var subCount int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ?", 3010).Count(&subCount).Error)
	assert.EqualValues(t, 0, subCount)

	var orderCount int64
	require.NoError(t, DB.Model(&SubscriptionOrder{}).Where("user_id = ?", 3010).Count(&orderCount).Error)
	assert.EqualValues(t, 0, orderCount)

	var topUpCount int64
	require.NoError(t, DB.Model(&TopUp{}).Where("user_id = ?", 3010).Count(&topUpCount).Error)
	assert.EqualValues(t, 0, topUpCount)
}

func TestPurchaseSubscriptionWithBalanceInsufficientQuotaDoesNotCreateSubscription(t *testing.T) {
	truncateTables(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 1000
	require.NoError(t, DB.Create(&User{
		Id:       3009,
		Username: "balance-subscription-low-quota-user",
		Status:   common.UserStatusEnabled,
		Quota:    999,
		Group:    "default",
		AffCode:  "balance-subscription-low-quota-user",
	}).Error)
	plan := insertSubscriptionPlanForModelLimitTest(t, 1009, 100, false, "")
	plan.PriceAmount = 1
	require.NoError(t, DB.Save(plan).Error)

	err := PurchaseSubscriptionWithBalance(3009, plan.Id)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "余额不足")
	assert.Equal(t, 999, getUserQuotaForPaymentGuardTest(t, 3009))

	var subCount int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ?", 3009).Count(&subCount).Error)
	assert.EqualValues(t, 0, subCount)

	var orderCount int64
	require.NoError(t, DB.Model(&SubscriptionOrder{}).Where("user_id = ?", 3009).Count(&orderCount).Error)
	assert.EqualValues(t, 0, orderCount)

	var topUpCount int64
	require.NoError(t, DB.Model(&TopUp{}).Where("user_id = ?", 3009).Count(&topUpCount).Error)
	assert.EqualValues(t, 0, topUpCount)
}
