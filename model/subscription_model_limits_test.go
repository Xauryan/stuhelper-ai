package model

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
