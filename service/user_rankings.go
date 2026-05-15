package service

import (
	"fmt"
	"time"

	"github.com/Xauryan/stuhelper-ai/model"
)

const userRankingLimit = 20

type UserRankingsResponse struct {
	Period      string       `json:"period"`
	Consumption []RankedUser `json:"consumption"`
	Recharge    []RankedUser `json:"recharge"`
	UpdatedAt   int64        `json:"updated_at"`
}

type RankedUser struct {
	Rank       int    `json:"rank"`
	Display    string `json:"display"`
	TotalQuota int64  `json:"total_quota"`
}

type userRankingPeriodConfig struct {
	id       string
	duration time.Duration
}

func GetUserRankingsSnapshot(period string) (*UserRankingsResponse, error) {
	config, err := userRankingConfig(period)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	startTime, endTime := userRankingTimeRange(config, now)

	consumptionRows, _, err := model.GetUserConsumptionRankingTotals(startTime, endTime, userRankingLimit)
	if err != nil {
		return nil, err
	}
	rechargeRows, _, err := model.GetUserRechargeRankingTotals(startTime, endTime, userRankingLimit)
	if err != nil {
		return nil, err
	}

	return &UserRankingsResponse{
		Period:      config.id,
		Consumption: buildRankedUsers(consumptionRows),
		Recharge:    buildRankedUsers(rechargeRows),
		UpdatedAt:   now.Unix(),
	}, nil
}

func userRankingConfig(period string) (userRankingPeriodConfig, error) {
	switch period {
	case "", "week":
		return userRankingPeriodConfig{id: "week", duration: 7 * 24 * time.Hour}, nil
	case "today", "day":
		return userRankingPeriodConfig{id: "day", duration: 24 * time.Hour}, nil
	case "month":
		return userRankingPeriodConfig{id: "month", duration: 30 * 24 * time.Hour}, nil
	case "all":
		return userRankingPeriodConfig{id: "all"}, nil
	default:
		return userRankingPeriodConfig{}, fmt.Errorf("invalid ranking period: %s", period)
	}
}

func userRankingTimeRange(config userRankingPeriodConfig, now time.Time) (int64, int64) {
	endTime := now.Unix()
	if config.duration <= 0 {
		return 0, endTime
	}
	return now.Add(-config.duration).Unix(), endTime
}

func buildRankedUsers(rows []model.UserRankingTotal) []RankedUser {
	result := make([]RankedUser, 0, len(rows))
	for idx, row := range rows {
		result = append(result, RankedUser{
			Rank:       idx + 1,
			Display:    maskRankingUsername(row.Username, row.UserId),
			TotalQuota: row.TotalQuota,
		})
	}
	return result
}

func maskRankingUsername(username string, userId int) string {
	if username == "" {
		return fmt.Sprintf("User #%d", userId)
	}
	runes := []rune(username)
	if len(runes) <= 2 {
		return string(runes[:1]) + "*"
	}
	if len(runes) <= 4 {
		return string(runes[:1]) + "**" + string(runes[len(runes)-1:])
	}
	return string(runes[:2]) + "***" + string(runes[len(runes)-2:])
}
