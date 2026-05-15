package model

import (
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserInsertPersistsInviterId(t *testing.T) {
	truncateTables(t)
	originalQuotaForInviter := common.QuotaForInviter
	originalQuotaForInvitee := common.QuotaForInvitee
	originalQuotaForNewUser := common.QuotaForNewUser
	originalInviterRewardAfterPayment := common.InviterRewardAfterPaymentEnabled
	t.Cleanup(func() {
		common.QuotaForInviter = originalQuotaForInviter
		common.QuotaForInvitee = originalQuotaForInvitee
		common.QuotaForNewUser = originalQuotaForNewUser
		common.InviterRewardAfterPaymentEnabled = originalInviterRewardAfterPayment
	})
	common.QuotaForInviter = 0
	common.QuotaForInvitee = 0
	common.QuotaForNewUser = 0
	common.InviterRewardAfterPaymentEnabled = false

	inviter := &User{
		Username: "insert-inviter",
		Status:   common.UserStatusEnabled,
		AffCode:  "insert-inviter-aff",
	}
	require.NoError(t, DB.Create(inviter).Error)

	invitee := &User{
		Username: "insert-invitee",
		Password: "password123",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, invitee.Insert(inviter.Id))

	var saved User
	require.NoError(t, DB.Select("id", "inviter_id").Where("username = ?", "insert-invitee").First(&saved).Error)
	require.Equal(t, inviter.Id, saved.InviterId)
}

func TestUserInsertWithTxPersistsInviterId(t *testing.T) {
	truncateTables(t)
	originalQuotaForInviter := common.QuotaForInviter
	originalQuotaForInvitee := common.QuotaForInvitee
	originalQuotaForNewUser := common.QuotaForNewUser
	originalInviterRewardAfterPayment := common.InviterRewardAfterPaymentEnabled
	t.Cleanup(func() {
		common.QuotaForInviter = originalQuotaForInviter
		common.QuotaForInvitee = originalQuotaForInvitee
		common.QuotaForNewUser = originalQuotaForNewUser
		common.InviterRewardAfterPaymentEnabled = originalInviterRewardAfterPayment
	})
	common.QuotaForInviter = 0
	common.QuotaForInvitee = 0
	common.QuotaForNewUser = 0
	common.InviterRewardAfterPaymentEnabled = false

	inviter := &User{
		Username: "tx-inviter",
		Status:   common.UserStatusEnabled,
		AffCode:  "tx-inviter-aff",
	}
	require.NoError(t, DB.Create(inviter).Error)

	invitee := &User{
		Username: "tx-invitee",
		Password: "password123",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return invitee.InsertWithTx(tx, inviter.Id)
	}))

	var saved User
	require.NoError(t, DB.Select("id", "inviter_id").Where("username = ?", "tx-invitee").First(&saved).Error)
	require.Equal(t, inviter.Id, saved.InviterId)
}
