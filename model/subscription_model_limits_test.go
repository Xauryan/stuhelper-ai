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

	var commissionCount int64
	require.NoError(t, DB.Model(&ReferralCommission{}).Count(&commissionCount).Error)
	assert.EqualValues(t, 0, commissionCount)
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
