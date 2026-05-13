package model

import (
	"testing"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setReferralCommissionSettingsForTest(t *testing.T, enabled bool, percent float64, maxRecharges int) {
	t.Helper()
	originalEnabled := common.ReferralCommissionEnabled
	originalPercent := common.ReferralCommissionPercent
	originalMaxRecharges := common.ReferralCommissionMaxRecharges
	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.ReferralCommissionEnabled = originalEnabled
		common.ReferralCommissionPercent = originalPercent
		common.ReferralCommissionMaxRecharges = originalMaxRecharges
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.ReferralCommissionEnabled = enabled
	common.ReferralCommissionPercent = percent
	common.ReferralCommissionMaxRecharges = maxRecharges
	common.QuotaPerUnit = 1000
}

func insertReferralUserForTest(t *testing.T, id int, username string, inviterId int, override *float64) {
	t.Helper()
	require.NoError(t, DB.Create(&User{
		Id:                        id,
		Username:                  username,
		Status:                    common.UserStatusEnabled,
		AffCode:                   "ref-" + username,
		InviterId:                 inviterId,
		ReferralCommissionPercent: override,
	}).Error)
}

func getReferralUserForTest(t *testing.T, id int) User {
	t.Helper()
	var user User
	require.NoError(t, DB.Where("id = ?", id).First(&user).Error)
	return user
}

func countReferralCommissionsForTest(t *testing.T, inviterId int) int64 {
	t.Helper()
	var count int64
	require.NoError(t, DB.Model(&ReferralCommission{}).Where("inviter_id = ?", inviterId).Count(&count).Error)
	return count
}

func TestCreditReferralCommissionCreditsInviterAndIsIdempotent(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 0)

	insertReferralUserForTest(t, 1, "inviter", 0, nil)
	insertReferralUserForTest(t, 2, "invitee", 1, nil)

	credited, err := CreditReferralCommission(2, 12.5, "stripe", ReferralCommissionSourceTopUp, 101)
	require.NoError(t, err)
	assert.True(t, credited)

	inviter := getReferralUserForTest(t, 1)
	assert.Equal(t, 1250, inviter.AffQuota)
	assert.Equal(t, 1250, inviter.AffHistoryQuota)
	assert.Equal(t, int64(1), countReferralCommissionsForTest(t, 1))

	credited, err = CreditReferralCommission(2, 12.5, "stripe", ReferralCommissionSourceTopUp, 101)
	require.NoError(t, err)
	assert.False(t, credited)

	inviter = getReferralUserForTest(t, 1)
	assert.Equal(t, 1250, inviter.AffQuota)
	assert.Equal(t, 1250, inviter.AffHistoryQuota)
	assert.Equal(t, int64(1), countReferralCommissionsForTest(t, 1))
}

func TestCreditReferralCommissionUsesInviterOverrideAndMaxRechargeCap(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 1)

	override := 25.0
	insertReferralUserForTest(t, 11, "override-inviter", 0, &override)
	insertReferralUserForTest(t, 12, "capped-invitee", 11, nil)

	credited, err := CreditReferralCommission(12, 10, "epay", ReferralCommissionSourceTopUp, 201)
	require.NoError(t, err)
	assert.True(t, credited)

	credited, err = CreditReferralCommission(12, 10, "epay", ReferralCommissionSourceTopUp, 202)
	require.NoError(t, err)
	assert.False(t, credited)

	inviter := getReferralUserForTest(t, 11)
	assert.Equal(t, 2500, inviter.AffQuota)
	assert.Equal(t, 2500, inviter.AffHistoryQuota)
	assert.Equal(t, int64(1), countReferralCommissionsForTest(t, 11))
}

func TestUpdateOptionRejectsInvalidReferralCommissionSettings(t *testing.T) {
	originalPercent := common.ReferralCommissionPercent
	originalMaxRecharges := common.ReferralCommissionMaxRecharges
	t.Cleanup(func() {
		common.ReferralCommissionPercent = originalPercent
		common.ReferralCommissionMaxRecharges = originalMaxRecharges
	})

	common.ReferralCommissionPercent = 12.5
	common.ReferralCommissionMaxRecharges = 3

	require.Error(t, UpdateOption("ReferralCommissionPercent", "invalid"))
	assert.Equal(t, 12.5, common.ReferralCommissionPercent)

	require.Error(t, UpdateOption("ReferralCommissionPercent", "101"))
	assert.Equal(t, 12.5, common.ReferralCommissionPercent)

	require.Error(t, UpdateOption("ReferralCommissionMaxRecharges", "-1"))
	assert.Equal(t, 3, common.ReferralCommissionMaxRecharges)
}

func TestInsertWithReferralCommissionEnabledSkipsLegacyInviteBonuses(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 0)

	originalQuotaForInviter := common.QuotaForInviter
	originalQuotaForInvitee := common.QuotaForInvitee
	originalQuotaForNewUser := common.QuotaForNewUser
	t.Cleanup(func() {
		common.QuotaForInviter = originalQuotaForInviter
		common.QuotaForInvitee = originalQuotaForInvitee
		common.QuotaForNewUser = originalQuotaForNewUser
	})
	common.QuotaForInviter = 300
	common.QuotaForInvitee = 200
	common.QuotaForNewUser = 100

	insertReferralUserForTest(t, 51, "registration-inviter", 0, nil)
	invitee := &User{
		Username:    "registration-invitee",
		Password:    "password123",
		DisplayName: "registration-invitee",
		InviterId:   51,
		Role:        common.RoleCommonUser,
	}

	require.NoError(t, invitee.Insert(51))

	inviter := getReferralUserForTest(t, 51)
	assert.Equal(t, 1, inviter.AffCount)
	assert.Equal(t, 0, inviter.AffQuota)
	assert.Equal(t, 0, inviter.AffHistoryQuota)

	createdInvitee := getReferralUserForTest(t, invitee.Id)
	assert.Equal(t, common.QuotaForNewUser, createdInvitee.Quota)
}

func TestCompleteSubscriptionOrderCreditsEachOrderOnce(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 0)

	insertReferralUserForTest(t, 21, "sub-inviter", 0, nil)
	insertReferralUserForTest(t, 22, "sub-invitee", 21, nil)
	plan := &SubscriptionPlan{
		Id:            501,
		Title:         "Referral Plan",
		PriceAmount:   10,
		Currency:      "USD",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   1000,
	}
	require.NoError(t, DB.Create(plan).Error)

	orders := []SubscriptionOrder{
		{
			UserId:          22,
			PlanId:          plan.Id,
			Money:           10,
			TradeNo:         "sub-ref-1",
			PaymentMethod:   PaymentProviderStripe,
			PaymentProvider: PaymentProviderStripe,
			Status:          common.TopUpStatusPending,
			CreateTime:      time.Now().Unix(),
		},
		{
			UserId:          22,
			PlanId:          plan.Id,
			Money:           15,
			TradeNo:         "sub-ref-2",
			PaymentMethod:   PaymentProviderStripe,
			PaymentProvider: PaymentProviderStripe,
			Status:          common.TopUpStatusPending,
			CreateTime:      time.Now().Unix(),
		},
	}
	require.NoError(t, DB.Create(&orders).Error)

	require.NoError(t, CompleteSubscriptionOrder("sub-ref-1", "", PaymentProviderStripe, ""))
	require.NoError(t, CompleteSubscriptionOrder("sub-ref-2", "", PaymentProviderStripe, ""))
	require.NoError(t, CompleteSubscriptionOrder("sub-ref-2", "", PaymentProviderStripe, ""))

	inviter := getReferralUserForTest(t, 21)
	assert.Equal(t, 2500, inviter.AffQuota)
	assert.Equal(t, 2500, inviter.AffHistoryQuota)
	assert.Equal(t, int64(2), countReferralCommissionsForTest(t, 21))
}

func TestCompleteSubscriptionOrderUsesActualPaymentMethodForTopUpAndCommission(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 0)

	insertReferralUserForTest(t, 41, "actual-method-inviter", 0, nil)
	insertReferralUserForTest(t, 42, "actual-method-invitee", 41, nil)
	plan := &SubscriptionPlan{
		Id:            601,
		Title:         "Actual Payment Method Plan",
		PriceAmount:   12,
		Currency:      "USD",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   1200,
	}
	require.NoError(t, DB.Create(plan).Error)

	order := &SubscriptionOrder{
		UserId:          42,
		PlanId:          plan.Id,
		Money:           12,
		TradeNo:         "sub-ref-actual-method",
		PaymentMethod:   PaymentProviderEpay,
		PaymentProvider: PaymentProviderEpay,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, DB.Create(order).Error)

	require.NoError(t, CompleteSubscriptionOrder("sub-ref-actual-method", "", PaymentProviderEpay, "alipay"))

	reloadedOrder := GetSubscriptionOrderByTradeNo("sub-ref-actual-method")
	require.NotNil(t, reloadedOrder)
	assert.Equal(t, "alipay", reloadedOrder.PaymentMethod)

	topUp := GetTopUpByTradeNo("sub-ref-actual-method")
	require.NotNil(t, topUp)
	assert.Equal(t, "alipay", topUp.PaymentMethod)

	var commission ReferralCommission
	require.NoError(t, DB.Where("invitee_id = ?", 42).First(&commission).Error)
	assert.Equal(t, "alipay", commission.PaymentMethod)
}

func TestGetUserReferralCommissionsSanitizesInvalidPagination(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 0)

	insertReferralUserForTest(t, 31, "page-inviter", 0, nil)
	insertReferralUserForTest(t, 32, "page-invitee", 31, nil)

	for i := 0; i < common.ItemsPerPage+2; i++ {
		credited, err := CreditReferralCommission(32, 10, "stripe", ReferralCommissionSourceTopUp, 700+i)
		require.NoError(t, err)
		require.True(t, credited)
	}

	commissions, total, err := GetUserReferralCommissions(31, &common.PageInfo{
		Page:     1,
		PageSize: -1,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(common.ItemsPerPage+2), total)
	assert.Len(t, commissions, common.ItemsPerPage)
}
