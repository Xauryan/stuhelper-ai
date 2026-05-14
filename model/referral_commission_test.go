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
	originalInviterRewardAfterPayment := common.InviterRewardAfterPaymentEnabled
	t.Cleanup(func() {
		common.ReferralCommissionEnabled = originalEnabled
		common.ReferralCommissionPercent = originalPercent
		common.ReferralCommissionMaxRecharges = originalMaxRecharges
		common.QuotaPerUnit = originalQuotaPerUnit
		common.InviterRewardAfterPaymentEnabled = originalInviterRewardAfterPayment
	})
	common.ReferralCommissionEnabled = enabled
	common.ReferralCommissionPercent = percent
	common.ReferralCommissionMaxRecharges = maxRecharges
	common.QuotaPerUnit = 1000
	common.InviterRewardAfterPaymentEnabled = false
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

func TestInsertWithReferralCommissionEnabledStillCreditsOneTimeInviteRewards(t *testing.T) {
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
	assert.Equal(t, common.QuotaForInviter, inviter.AffQuota)
	assert.Equal(t, common.QuotaForInviter, inviter.AffHistoryQuota)

	createdInvitee := getReferralUserForTest(t, invitee.Id)
	assert.Equal(t, common.QuotaForNewUser+common.QuotaForInvitee, createdInvitee.Quota)
	assert.Equal(t, common.QuotaForInvitee, createdInvitee.InviteeRewardQuota)
	assert.True(t, createdInvitee.InviterRewardUnlocked)
}

func TestInviterRewardAfterPaymentDelaysAndUnlocksOnce(t *testing.T) {
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
	common.InviterRewardAfterPaymentEnabled = true

	insertReferralUserForTest(t, 61, "delayed-inviter", 0, nil)
	invitee := &User{
		Username:    "delayed-invitee",
		Password:    "password123",
		DisplayName: "delayed-invitee",
		InviterId:   61,
		Role:        common.RoleCommonUser,
	}

	require.NoError(t, invitee.Insert(61))

	createdInvitee := getReferralUserForTest(t, invitee.Id)
	assert.Equal(t, common.QuotaForNewUser+common.QuotaForInvitee, createdInvitee.Quota)
	assert.Equal(t, common.QuotaForInvitee, createdInvitee.InviteeRewardQuota)
	assert.False(t, createdInvitee.InviterRewardUnlocked)
	inviter := getReferralUserForTest(t, 61)
	assert.Equal(t, 1, inviter.AffCount)
	assert.Equal(t, 0, inviter.AffQuota)

	referralResult, err := CreditInviteRewardsAfterPaymentTx(DB, invitee.Id, 10, "stripe", ReferralCommissionSourceTopUp, 901)
	require.NoError(t, err)
	require.NotNil(t, referralResult)
	assert.True(t, referralResult.InviterRewardCredited)
	assert.Equal(t, common.QuotaForInviter, referralResult.InviterRewardQuota)
	assert.True(t, referralResult.CommissionCredited)

	createdInvitee = getReferralUserForTest(t, invitee.Id)
	assert.Equal(t, common.QuotaForNewUser+common.QuotaForInvitee, createdInvitee.Quota)
	assert.True(t, createdInvitee.InviterRewardUnlocked)
	inviter = getReferralUserForTest(t, 61)
	assert.Equal(t, 1, inviter.AffCount)
	assert.Equal(t, common.QuotaForInviter+1000, inviter.AffQuota)
	assert.Equal(t, common.QuotaForInviter+1000, inviter.AffHistoryQuota)

	referralResult, err = CreditInviteRewardsAfterPaymentTx(DB, invitee.Id, 10, "stripe", ReferralCommissionSourceTopUp, 902)
	require.NoError(t, err)
	require.NotNil(t, referralResult)
	assert.False(t, referralResult.InviterRewardCredited)
	assert.True(t, referralResult.CommissionCredited)

	createdInvitee = getReferralUserForTest(t, invitee.Id)
	assert.Equal(t, common.QuotaForNewUser+common.QuotaForInvitee, createdInvitee.Quota)
	inviter = getReferralUserForTest(t, 61)
	assert.Equal(t, 1, inviter.AffCount)
	assert.Equal(t, common.QuotaForInviter+2000, inviter.AffQuota)
	assert.Equal(t, common.QuotaForInviter+2000, inviter.AffHistoryQuota)
}

func TestInviterRewardAfterPaymentUsesRegistrationQuotaSnapshot(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, false, 10, 0)

	originalQuotaForInviter := common.QuotaForInviter
	originalQuotaForInvitee := common.QuotaForInvitee
	originalQuotaForNewUser := common.QuotaForNewUser
	t.Cleanup(func() {
		common.QuotaForInviter = originalQuotaForInviter
		common.QuotaForInvitee = originalQuotaForInvitee
		common.QuotaForNewUser = originalQuotaForNewUser
	})
	common.QuotaForInviter = 300
	common.QuotaForInvitee = 0
	common.QuotaForNewUser = 100
	common.InviterRewardAfterPaymentEnabled = true

	insertReferralUserForTest(t, 71, "snapshot-inviter", 0, nil)
	invitee := &User{
		Username:    "snapshot-invitee",
		Password:    "password123",
		DisplayName: "snapshot-invitee",
		InviterId:   71,
		Role:        common.RoleCommonUser,
	}
	require.NoError(t, invitee.Insert(71))

	createdInvitee := getReferralUserForTest(t, invitee.Id)
	assert.Equal(t, 300, createdInvitee.InviterRewardQuota)
	assert.False(t, createdInvitee.InviterRewardUnlocked)

	common.QuotaForInviter = 900
	referralResult, err := CreditInviteRewardsAfterPaymentTx(DB, invitee.Id, 10, "stripe", ReferralCommissionSourceTopUp, 911)
	require.NoError(t, err)
	require.NotNil(t, referralResult)
	assert.True(t, referralResult.InviterRewardCredited)
	assert.Equal(t, 300, referralResult.InviterRewardQuota)

	inviter := getReferralUserForTest(t, 71)
	assert.Equal(t, 300, inviter.AffQuota)
	assert.Equal(t, 300, inviter.AffHistoryQuota)
}

func TestPendingInviterRewardUnlocksEvenIfSwitchIsDisabledLater(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, false, 10, 0)

	originalQuotaForInviter := common.QuotaForInviter
	originalQuotaForInvitee := common.QuotaForInvitee
	originalQuotaForNewUser := common.QuotaForNewUser
	t.Cleanup(func() {
		common.QuotaForInviter = originalQuotaForInviter
		common.QuotaForInvitee = originalQuotaForInvitee
		common.QuotaForNewUser = originalQuotaForNewUser
	})
	common.QuotaForInviter = 300
	common.QuotaForInvitee = 0
	common.QuotaForNewUser = 100
	common.InviterRewardAfterPaymentEnabled = true

	insertReferralUserForTest(t, 81, "toggle-inviter", 0, nil)
	invitee := &User{
		Username:    "toggle-invitee",
		Password:    "password123",
		DisplayName: "toggle-invitee",
		InviterId:   81,
		Role:        common.RoleCommonUser,
	}
	require.NoError(t, invitee.Insert(81))

	common.InviterRewardAfterPaymentEnabled = false
	referralResult, err := CreditInviteRewardsAfterPaymentTx(DB, invitee.Id, 10, "stripe", ReferralCommissionSourceTopUp, 921)
	require.NoError(t, err)
	require.NotNil(t, referralResult)
	assert.True(t, referralResult.InviterRewardCredited)
	assert.Equal(t, 300, referralResult.InviterRewardQuota)

	inviter := getReferralUserForTest(t, 81)
	assert.Equal(t, 300, inviter.AffQuota)
	assert.Equal(t, 300, inviter.AffHistoryQuota)
}

func TestBackfillInviterRewardMigrationStateSkipsExistingSchemaPendingRewards(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, false, 10, 0)

	insertReferralUserForTest(t, 91, "migration-inviter", 0, nil)
	insertReferralUserForTest(t, 92, "migration-invitee", 91, nil)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 92).Updates(map[string]interface{}{
		"inviter_reward_quota":    300,
		"inviter_reward_unlocked": false,
	}).Error)

	require.NoError(t, backfillInviterRewardMigrationState(true))

	invitee := getReferralUserForTest(t, 92)
	assert.Equal(t, 300, invitee.InviterRewardQuota)
	assert.False(t, invitee.InviterRewardUnlocked)
}

func TestBackfillInviterRewardMigrationStateMarksPreExistingInviteRelations(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, false, 10, 0)

	insertReferralUserForTest(t, 93, "legacy-migration-inviter", 0, nil)
	insertReferralUserForTest(t, 94, "legacy-migration-invitee", 93, nil)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 94).Updates(map[string]interface{}{
		"inviter_reward_quota":    300,
		"inviter_reward_unlocked": false,
	}).Error)

	require.NoError(t, backfillInviterRewardMigrationState(false))

	invitee := getReferralUserForTest(t, 94)
	assert.Equal(t, 0, invitee.InviterRewardQuota)
	assert.True(t, invitee.InviterRewardUnlocked)
}

func TestGetAdminReferralRecordsIncludesRewardAndCommissionState(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, true, 10, 0)

	insertReferralUserForTest(t, 101, "admin-ref-inviter", 0, nil)
	insertReferralUserForTest(t, 102, "admin-ref-invitee", 101, nil)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 102).Updates(map[string]interface{}{
		"invitee_reward_quota":    200,
		"inviter_reward_quota":    300,
		"inviter_reward_unlocked": true,
	}).Error)

	require.NoError(t, DB.Create(&TopUp{
		Id:            701,
		UserId:        102,
		Amount:        1000,
		Money:         10,
		TradeNo:       "admin-ref-topup",
		PaymentMethod: PaymentMethodStripe,
		CreateTime:    1700,
		CompleteTime:  1800,
		Status:        common.TopUpStatusSuccess,
	}).Error)
	require.NoError(t, DB.Create(&SubscriptionOrder{
		Id:            801,
		UserId:        102,
		PlanId:        1,
		Money:         15,
		TradeNo:       "admin-ref-sub",
		PaymentMethod: PaymentProviderStripe,
		Status:        common.TopUpStatusSuccess,
		CreateTime:    1900,
		CompleteTime:  2000,
	}).Error)
	require.NoError(t, DB.Create(&ReferralCommission{
		Id:              901,
		InviterId:       101,
		InviteeId:       102,
		SourceType:      ReferralCommissionSourceTopUp,
		SourceId:        701,
		PaymentMethod:   PaymentMethodStripe,
		RechargeAmount:  10,
		CommissionQuota: 1000,
		CommissionRate:  10,
		CreatedAt:       1800,
	}).Error)
	require.NoError(t, DB.Create(&ReferralCommission{
		Id:              902,
		InviterId:       101,
		InviteeId:       102,
		SourceType:      ReferralCommissionSourceSubscription,
		SourceId:        801,
		PaymentMethod:   PaymentProviderStripe,
		RechargeAmount:  15,
		CommissionQuota: 1500,
		CommissionRate:  10,
		CreatedAt:       2000,
	}).Error)

	records, total, err := GetAdminReferralRecords(&AdminReferralQuery{
		PageInfo: &common.PageInfo{Page: 1, PageSize: 20},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, records, 1)

	record := records[0]
	assert.Equal(t, 101, record.InviterId)
	assert.Equal(t, "admin-ref-inviter", record.InviterUsername)
	assert.Equal(t, 102, record.InviteeId)
	assert.Equal(t, "admin-ref-invitee", record.InviteeUsername)
	assert.Equal(t, 200, record.InviteeRewardQuota)
	assert.Equal(t, 300, record.InviterRewardQuota)
	assert.True(t, record.InviterRewardUnlocked)
	assert.True(t, record.InviteeHasPaid)
	assert.Equal(t, int64(1800), record.FirstPaymentTime)
	assert.Equal(t, 2, record.CommissionCount)
	assert.Equal(t, 2500, record.TotalCommissionQuota)
	assert.InDelta(t, 25.0, record.TotalRechargeAmount, 0.0001)
	assert.Equal(t, int64(2000), record.LastCommissionAt)

	commissions, commissionTotal, err := GetAdminReferralCommissions(102, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.EqualValues(t, 2, commissionTotal)
	require.Len(t, commissions, 2)
	assert.Equal(t, ReferralCommissionSourceSubscription, commissions[0].SourceType)
	assert.Equal(t, ReferralCommissionSourceTopUp, commissions[1].SourceType)
}

func TestGetAdminReferralRecordsFiltersByKeywordAndRewardStatus(t *testing.T) {
	truncateTables(t)
	setReferralCommissionSettingsForTest(t, false, 10, 0)

	insertReferralUserForTest(t, 111, "filter-inviter", 0, nil)
	insertReferralUserForTest(t, 112, "filter-paid", 111, nil)
	insertReferralUserForTest(t, 113, "filter-pending", 111, nil)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 112).Updates(map[string]interface{}{
		"inviter_reward_quota":    300,
		"inviter_reward_unlocked": true,
	}).Error)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 113).Updates(map[string]interface{}{
		"inviter_reward_quota":    300,
		"inviter_reward_unlocked": false,
	}).Error)

	records, total, err := GetAdminReferralRecords(&AdminReferralQuery{
		PageInfo:     &common.PageInfo{Page: 1, PageSize: 20},
		Keyword:      "pending",
		RewardStatus: AdminReferralRewardStatusPending,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, records, 1)
	assert.Equal(t, 113, records[0].InviteeId)
	assert.False(t, records[0].InviterRewardUnlocked)
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
