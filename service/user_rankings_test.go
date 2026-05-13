package service

import (
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRankingsResponseDoesNotExposeAggregateTotalsOrShare(t *testing.T) {
	rows := []model.UserRankingTotal{
		{UserId: 1, Username: "alice", TotalQuota: 900},
		{UserId: 2, Username: "bob", TotalQuota: 100},
	}

	ranked := buildRankedUsers(rows)

	require.Len(t, ranked, 2)
	assert.Equal(t, 1, ranked[0].Rank)
	assert.Equal(t, 1, ranked[0].UserId)
	assert.Equal(t, "al***ce", ranked[0].Display)

	data, err := common.Marshal(UserRankingsResponse{
		Period:      "week",
		Consumption: ranked,
		Recharge:    ranked,
		UpdatedAt:   1700000000,
	})
	require.NoError(t, err)
	assert.NotContains(t, string(data), "share")
	assert.NotContains(t, string(data), "consumption_total")
	assert.NotContains(t, string(data), "recharge_total")
}
