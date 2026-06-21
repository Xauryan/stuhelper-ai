package model

import (
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/stretchr/testify/require"
)

func TestHardDeleteUserDeletesOAuthBindings(t *testing.T) {
	truncateTables(t)

	user := &User{
		Username: "hard-delete-oauth",
		Status:   common.UserStatusEnabled,
		AffCode:  "hard-delete-oauth-aff",
	}
	require.NoError(t, DB.Create(user).Error)
	require.NoError(t, CreateUserOAuthBinding(&UserOAuthBinding{
		UserId:         user.Id,
		ProviderId:     1001,
		ProviderUserId: "provider-user-1001",
	}))

	require.NoError(t, user.HardDelete())

	var bindingCount int64
	require.NoError(t, DB.Model(&UserOAuthBinding{}).Where("user_id = ?", user.Id).Count(&bindingCount).Error)
	require.Equal(t, int64(0), bindingCount)

	var userCount int64
	require.NoError(t, DB.Unscoped().Model(&User{}).Where("id = ?", user.Id).Count(&userCount).Error)
	require.Equal(t, int64(0), userCount)
}

func TestHardDeleteUserByIdDeletesOAuthBindings(t *testing.T) {
	truncateTables(t)

	user := &User{
		Username: "hard-delete-by-id-oauth",
		Status:   common.UserStatusEnabled,
		AffCode:  "hard-delete-by-id-oauth-aff",
	}
	require.NoError(t, DB.Create(user).Error)
	require.NoError(t, CreateUserOAuthBinding(&UserOAuthBinding{
		UserId:         user.Id,
		ProviderId:     1002,
		ProviderUserId: "provider-user-1002",
	}))

	require.NoError(t, HardDeleteUserById(user.Id))

	var bindingCount int64
	require.NoError(t, DB.Model(&UserOAuthBinding{}).Where("user_id = ?", user.Id).Count(&bindingCount).Error)
	require.Equal(t, int64(0), bindingCount)

	var userCount int64
	require.NoError(t, DB.Unscoped().Model(&User{}).Where("id = ?", user.Id).Count(&userCount).Error)
	require.Equal(t, int64(0), userCount)
}
