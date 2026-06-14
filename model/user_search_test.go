package model

import (
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/stretchr/testify/require"
)

func TestSearchUsersStatusDeletedFilter(t *testing.T) {
	truncateTables(t)

	active := &User{Id: 9101, Username: "search-status-active", Email: "active@example.com", Status: common.UserStatusEnabled, AffCode: "search-status-active-aff"}
	disabled := &User{Id: 9102, Username: "search-status-disabled", Email: "disabled@example.com", Status: common.UserStatusDisabled, AffCode: "search-status-disabled-aff"}
	deleted := &User{Id: 9103, Username: "search-status-deleted", Email: "deleted@example.com", Status: common.UserStatusEnabled, AffCode: "search-status-deleted-aff"}
	require.NoError(t, DB.Create(active).Error)
	require.NoError(t, DB.Create(disabled).Error)
	require.NoError(t, DB.Create(deleted).Error)
	require.NoError(t, DB.Delete(deleted).Error)

	deletedStatus := -1
	users, total, err := SearchUsers("search-status", "", nil, &deletedStatus, 0, 20)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, users, 1)
	require.Equal(t, deleted.Id, users[0].Id)

	enabledStatus := common.UserStatusEnabled
	users, total, err = SearchUsers("search-status", "", nil, &enabledStatus, 0, 20)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, users, 1)
	require.Equal(t, active.Id, users[0].Id)

	disabledStatus := common.UserStatusDisabled
	users, total, err = SearchUsers("search-status", "", nil, &disabledStatus, 0, 20)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, users, 1)
	require.Equal(t, disabled.Id, users[0].Id)
}
