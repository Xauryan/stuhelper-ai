package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/Xauryan/stuhelper-ai/model"
)

const (
	userRankingLimit    = 20
	userRankingCacheTTL = 60 * time.Second
)

type UserRankingsResponse struct {
	Period      string       `json:"period"`
	Consumption []RankedUser `json:"consumption"`
	Recharge    []RankedUser `json:"recharge"`
	Me          *RankedUser  `json:"me,omitempty"`
	UpdatedAt   int64        `json:"updated_at"`
}

type RankedUser struct {
	Rank         int    `json:"rank"`
	Display      string `json:"display"`
	TotalQuota   int64  `json:"total_quota"`
	TotalTokens  int64  `json:"total_tokens"`
	RequestCount int64  `json:"request_count"`
	IsMe         bool   `json:"is_me,omitempty"`
}

type userRankingPeriodConfig struct {
	id       string
	duration time.Duration
}

type userRankingCacheItem struct {
	expiresAt       time.Time
	data            *UserRankingsResponse
	consumptionRows []model.UserRankingTotal
	startTime       int64
	endTime         int64
}

var (
	userRankingCacheMu sync.Mutex
	userRankingCache   = map[string]userRankingCacheItem{}
)

func ClearUserRankingsCache() {
	userRankingCacheMu.Lock()
	userRankingCache = map[string]userRankingCacheItem{}
	userRankingCacheMu.Unlock()
}

func GetUserRankingsSnapshot(period string, viewerUserId int, consumptionMetricParam ...string) (*UserRankingsResponse, error) {
	config, err := userRankingConfig(period)
	if err != nil {
		return nil, err
	}
	consumptionMetric, err := userRankingConsumptionMetric(consumptionMetricParam...)
	if err != nil {
		return nil, err
	}

	item, err := getCachedUserRankingsPublicSnapshot(config, consumptionMetric)
	if err != nil {
		return nil, err
	}

	out := cloneUserRankingsResponse(item.data)
	if viewerUserId > 0 {
		me, err := buildUserRankingMeRow(item, viewerUserId, consumptionMetric)
		if err != nil {
			return nil, err
		}
		out.Me = me
		for i, src := range item.consumptionRows {
			if src.UserId == viewerUserId && i < len(out.Consumption) {
				out.Consumption[i].IsMe = true
			}
		}
	}
	return out, nil
}

func getCachedUserRankingsPublicSnapshot(config userRankingPeriodConfig, consumptionMetric string) (userRankingCacheItem, error) {
	now := time.Now()
	cacheKey := userRankingCacheKey(config, consumptionMetric)

	userRankingCacheMu.Lock()
	cached, ok := userRankingCache[cacheKey]
	userRankingCacheMu.Unlock()
	if ok && cached.expiresAt.After(now) {
		return cached, nil
	}

	startTime, endTime := userRankingTimeRange(config, now)

	consumptionRows, _, err := model.GetUserConsumptionRankingTotalsByMetric(startTime, endTime, userRankingLimit, consumptionMetric)
	if err != nil {
		return userRankingCacheItem{}, err
	}
	rechargeRows, _, err := model.GetUserRechargeRankingTotals(startTime, endTime, userRankingLimit)
	if err != nil {
		return userRankingCacheItem{}, err
	}

	item := userRankingCacheItem{
		expiresAt: now.Add(userRankingCacheTTL),
		data: &UserRankingsResponse{
			Period:      config.id,
			Consumption: buildRankedUsers(consumptionRows),
			Recharge:    buildRankedUsers(rechargeRows),
			UpdatedAt:   now.Unix(),
		},
		consumptionRows: consumptionRows,
		startTime:       startTime,
		endTime:         endTime,
	}

	userRankingCacheMu.Lock()
	userRankingCache[cacheKey] = item
	userRankingCacheMu.Unlock()

	return item, nil
}

func userRankingCacheKey(config userRankingPeriodConfig, consumptionMetric string) string {
	return config.id + ":" + consumptionMetric
}

func cloneUserRankingsResponse(src *UserRankingsResponse) *UserRankingsResponse {
	dst := *src
	dst.Consumption = append([]RankedUser(nil), src.Consumption...)
	dst.Recharge = append([]RankedUser(nil), src.Recharge...)
	dst.Me = nil
	return &dst
}

func buildUserRankingMeRow(item userRankingCacheItem, viewerUserId int, consumptionMetric string) (*RankedUser, error) {
	for idx, row := range item.consumptionRows {
		if row.UserId == viewerUserId {
			return &RankedUser{
				Rank:         idx + 1,
				Display:      displayNameForRankingSelf(row),
				TotalQuota:   row.TotalQuota,
				TotalTokens:  row.TotalTokens,
				RequestCount: row.RequestCount,
			}, nil
		}
	}

	row, err := model.GetUserConsumptionRankingTotalForUser(item.startTime, item.endTime, viewerUserId)
	if err != nil || row == nil {
		return nil, err
	}
	rank, err := model.GetUserConsumptionRankingRankByMetric(item.startTime, item.endTime, *row, consumptionMetric)
	if err != nil {
		return nil, err
	}
	return &RankedUser{
		Rank:         rank,
		Display:      displayNameForRankingSelf(*row),
		TotalQuota:   row.TotalQuota,
		TotalTokens:  row.TotalTokens,
		RequestCount: row.RequestCount,
	}, nil
}

func displayNameForRankingSelf(row model.UserRankingTotal) string {
	if name := model.GetUserSelfDisplayNameById(row.UserId); name != "" {
		return name
	}
	if row.Username != "" {
		return row.Username
	}
	return fmt.Sprintf("User #%d", row.UserId)
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

func userRankingConsumptionMetric(metricParam ...string) (string, error) {
	metric := model.UserConsumptionRankingMetricTokens
	if len(metricParam) > 0 {
		metric = metricParam[0]
	}
	switch metric {
	case "", model.UserConsumptionRankingMetricTokens:
		return model.UserConsumptionRankingMetricTokens, nil
	case model.UserConsumptionRankingMetricQuota, model.UserConsumptionRankingMetricCalls:
		return metric, nil
	default:
		return "", fmt.Errorf("invalid ranking metric: %s", metric)
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
			Rank:         idx + 1,
			Display:      maskRankingUsername(row.Username, row.UserId),
			TotalQuota:   row.TotalQuota,
			TotalTokens:  row.TotalTokens,
			RequestCount: row.RequestCount,
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
