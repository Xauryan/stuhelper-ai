package model

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/setting/operation_setting"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const userRankingDefaultLimit = 20

var adminRechargeQuotaContentPattern = regexp.MustCompile(`管理员(?:增加|充值)用户额度\s*[^0-9]*([0-9]+(?:\.[0-9]+)?)`)

type UserRankingTotal struct {
	UserId       int    `json:"user_id"       gorm:"column:user_id"`
	Username     string `json:"username"      gorm:"-"`
	TotalQuota   int64  `json:"total_quota"   gorm:"column:total_quota"`
	TotalTokens  int64  `json:"total_tokens"  gorm:"column:total_tokens"`
	RequestCount int64  `json:"request_count" gorm:"column:request_count"`
}

func GetUserSelfDisplayNameById(userId int) string {
	if userId <= 0 || DB == nil {
		return ""
	}
	var record struct {
		Username    string `gorm:"column:username"`
		DisplayName string `gorm:"column:display_name"`
	}
	if err := DB.Model(&User{}).
		Select("username", "display_name").
		Where("id = ?", userId).
		Take(&record).Error; err != nil {
		return ""
	}
	if record.Username != "" {
		return record.Username
	}
	return strings.TrimSpace(record.DisplayName)
}

func userConsumptionRankingBaseQuery(startTime int64, endTime int64) *gorm.DB {
	query := LOG_DB.Table("logs").
		Select("user_id, "+
			"sum(coalesce(quota, 0)) as total_quota, "+
			"sum(coalesce(prompt_tokens, 0) + coalesce(completion_tokens, 0)) as total_tokens, "+
			"count(*) as request_count").
		Where("type = ? AND quota > 0", LogTypeConsume).
		Group("user_id").
		Having("sum(quota) > 0")
	return applyUnixTimeRange(query, "created_at", startTime, endTime)
}

func GetUserConsumptionRankingTotals(startTime int64, endTime int64, limit int) ([]UserRankingTotal, int64, error) {
	if limit <= 0 {
		limit = userRankingDefaultLimit
	}
	var rows []UserRankingTotal
	if err := userConsumptionRankingBaseQuery(startTime, endTime).
		Order("total_tokens DESC").
		Order("user_id ASC").
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	sortUserConsumptionRankingRows(rows)
	total := sumUserRankingQuota(rows)
	if len(rows) > limit {
		rows = rows[:limit]
	}
	rows = normalizeUserRankingRows(rows)
	return rows, total, nil
}

func GetUserConsumptionRankingTotalForUser(startTime int64, endTime int64, userId int) (*UserRankingTotal, error) {
	if userId <= 0 {
		return nil, nil
	}
	var row UserRankingTotal
	err := userConsumptionRankingBaseQuery(startTime, endTime).
		Where("user_id = ?", userId).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.UserId <= 0 || row.TotalQuota <= 0 {
		return nil, nil
	}
	rows := normalizeUserRankingRows([]UserRankingTotal{row})
	return &rows[0], nil
}

func GetUserConsumptionRankingRank(startTime int64, endTime int64, me UserRankingTotal) (int, error) {
	if me.UserId <= 0 || me.TotalQuota <= 0 {
		return 0, nil
	}
	base := userConsumptionRankingBaseQuery(startTime, endTime)
	var betterCount int64
	err := LOG_DB.Table("(?) AS ranked", base).
		Where("total_tokens > ? OR (total_tokens = ? AND user_id < ?)",
			me.TotalTokens, me.TotalTokens, me.UserId).
		Count(&betterCount).Error
	if err != nil {
		return 0, err
	}
	return int(betterCount) + 1, nil
}

func sortUserConsumptionRankingRows(rows []UserRankingTotal) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TotalTokens == rows[j].TotalTokens {
			return rows[i].UserId < rows[j].UserId
		}
		return rows[i].TotalTokens > rows[j].TotalTokens
	})
}

func GetUserRechargeRankingTotals(startTime int64, endTime int64, limit int) ([]UserRankingTotal, int64, error) {
	aggregates := make(map[int]*UserRankingTotal)
	if err := mergeSuccessfulTopUpRankingRows(aggregates, startTime, endTime); err != nil {
		return nil, 0, err
	}
	if err := mergeUserRankingRows(aggregates, getRedeemedCodeRankingRows(startTime, endTime)); err != nil {
		return nil, 0, err
	}
	if err := mergeAdminAddedQuotaRankingRows(aggregates, startTime, endTime); err != nil {
		return nil, 0, err
	}

	rows := userRankingMapToSortedRows(aggregates)
	total := sumUserRankingQuota(rows)
	if limit <= 0 {
		limit = userRankingDefaultLimit
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, total, nil
}

func mergeSuccessfulTopUpRankingRows(aggregates map[int]*UserRankingTotal, startTime int64, endTime int64) error {
	var topUps []TopUp
	query := DB.Where("status IN ?", []string{common.TopUpStatusSuccess, common.TopUpStatusPartialRefunded})
	if startTime > 0 && endTime > 0 {
		query = query.Where("(complete_time >= ? AND complete_time <= ?) OR (complete_time = 0 AND create_time >= ? AND create_time <= ?)", startTime, endTime, startTime, endTime)
	} else if startTime > 0 {
		query = query.Where("(complete_time >= ?) OR (complete_time = 0 AND create_time >= ?)", startTime, startTime)
	} else if endTime > 0 {
		query = query.Where("(complete_time <= ? AND complete_time > 0) OR (complete_time = 0 AND create_time <= ?)", endTime, endTime)
	}
	if err := query.Find(&topUps).Error; err != nil {
		return err
	}
	for _, topUp := range topUps {
		quota := topUpCreditedQuota(topUp) - topUp.RefundedQuota
		if topUp.UserId <= 0 || quota <= 0 {
			continue
		}
		item := aggregates[topUp.UserId]
		if item == nil {
			aggregates[topUp.UserId] = &UserRankingTotal{
				UserId:     topUp.UserId,
				Username:   usernameForRanking(topUp.UserId),
				TotalQuota: quota,
			}
			continue
		}
		item.TotalQuota += quota
	}
	return nil
}

func getRedeemedCodeRankingRows(startTime int64, endTime int64) *gorm.DB {
	query := DB.Table("redemptions").
		Select("redemptions.used_user_id as user_id, sum(redemptions.quota) as total_quota").
		Where("redemptions.status = ? AND redemptions.used_user_id > 0", common.RedemptionCodeStatusUsed).
		Group("redemptions.used_user_id")
	return applyUnixTimeRange(query, "redemptions.redeemed_time", startTime, endTime)
}

func mergeUserRankingRows(aggregates map[int]*UserRankingTotal, query *gorm.DB) error {
	var rows []UserRankingTotal
	if err := query.Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range normalizeUserRankingRows(rows) {
		if row.UserId <= 0 || row.TotalQuota <= 0 {
			continue
		}
		item := aggregates[row.UserId]
		if item == nil {
			copyRow := row
			aggregates[row.UserId] = &copyRow
			continue
		}
		item.TotalQuota += row.TotalQuota
		if item.Username == "" && row.Username != "" {
			item.Username = row.Username
		}
	}
	return nil
}

func mergeAdminAddedQuotaRankingRows(aggregates map[int]*UserRankingTotal, startTime int64, endTime int64) error {
	var logs []Log
	query := LOG_DB.
		Where("type = ? AND (content LIKE ? OR content LIKE ?)", LogTypeManage, "管理员充值用户额度%", "管理员增加用户额度%")
	query = applyUnixTimeRange(query, "created_at", startTime, endTime)
	if err := query.Find(&logs).Error; err != nil {
		return err
	}
	for _, log := range logs {
		if log.UserId <= 0 {
			continue
		}
		quota := int64(log.Quota)
		if quota <= 0 {
			quota = parseAdminRechargeQuotaFromContent(log.Content)
		}
		if quota <= 0 {
			continue
		}
		item := aggregates[log.UserId]
		if item == nil {
			aggregates[log.UserId] = &UserRankingTotal{
				UserId:     log.UserId,
				Username:   usernameForRanking(log.UserId),
				TotalQuota: quota,
			}
			continue
		}
		item.TotalQuota += quota
	}
	return nil
}

func applyUnixTimeRange(query *gorm.DB, column string, startTime int64, endTime int64) *gorm.DB {
	if startTime > 0 {
		query = query.Where(column+" >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where(column+" <= ?", endTime)
	}
	return query
}

func normalizeUserRankingRows(rows []UserRankingTotal) []UserRankingTotal {
	missing := make([]int, 0, len(rows))
	seen := make(map[int]struct{}, len(rows))
	for i := range rows {
		if rows[i].Username == "" && rows[i].UserId > 0 {
			if _, dup := seen[rows[i].UserId]; !dup {
				seen[rows[i].UserId] = struct{}{}
				missing = append(missing, rows[i].UserId)
			}
		}
	}
	if len(missing) == 0 {
		return rows
	}

	names := batchGetUsernamesByIds(missing)
	for i := range rows {
		if rows[i].Username == "" && rows[i].UserId > 0 {
			if name, ok := names[rows[i].UserId]; ok {
				rows[i].Username = name
			}
		}
	}
	return rows
}

func batchGetUsernamesByIds(ids []int) map[int]string {
	out := make(map[int]string, len(ids))
	if len(ids) == 0 || DB == nil {
		return out
	}
	var records []struct {
		Id       int    `gorm:"column:id"`
		Username string `gorm:"column:username"`
	}
	if err := DB.Model(&User{}).
		Select("id", "username").
		Where("id IN ?", ids).
		Find(&records).Error; err != nil {
		return out
	}
	for _, r := range records {
		out[r.Id] = r.Username
	}
	return out
}

func sumUserRankingQuota(rows []UserRankingTotal) int64 {
	total := int64(0)
	for _, row := range rows {
		if row.TotalQuota > 0 {
			total += row.TotalQuota
		}
	}
	return total
}

func userRankingMapToSortedRows(aggregates map[int]*UserRankingTotal) []UserRankingTotal {
	rows := make([]UserRankingTotal, 0, len(aggregates))
	for _, row := range aggregates {
		if row.TotalQuota > 0 {
			rows = append(rows, *row)
		}
	}
	sortUserRankingRows(rows)
	return rows
}

func sortUserRankingRows(rows []UserRankingTotal) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TotalQuota == rows[j].TotalQuota {
			return rows[i].UserId < rows[j].UserId
		}
		return rows[i].TotalQuota > rows[j].TotalQuota
	})
}

func topUpCreditedQuota(topUp TopUp) int64 {
	if IsAdminTopUpRecord(&topUp) {
		return topUp.Amount
	}
	switch topUp.PaymentProvider {
	case PaymentProviderCreem:
		return topUp.Amount
	case PaymentProviderStripe:
		return decimal.NewFromFloat(topUp.Money).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	default:
		return decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	}
}

func parseAdminRechargeQuotaFromContent(content string) int64 {
	if !strings.HasPrefix(content, "管理员充值用户额度") && !strings.HasPrefix(content, "管理员增加用户额度") {
		return 0
	}
	matches := adminRechargeQuotaContentPattern.FindStringSubmatch(content)
	if len(matches) < 2 {
		return 0
	}
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil || value <= 0 {
		return 0
	}
	if strings.Contains(content, "点额度") {
		return decimal.NewFromFloat(value).IntPart()
	}

	multiplier := common.QuotaPerUnit
	customSymbol := operation_setting.GetGeneralSetting().CustomCurrencySymbol
	switch {
	case strings.Contains(content, "¥"):
		if operation_setting.USDExchangeRate <= 0 {
			return 0
		}
		multiplier = common.QuotaPerUnit / operation_setting.USDExchangeRate
	case strings.Contains(content, "¤"), customSymbol != "" && strings.Contains(content, customSymbol):
		rate := operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate
		if rate <= 0 {
			return 0
		}
		multiplier = common.QuotaPerUnit / rate
	}
	return decimal.NewFromFloat(value).Mul(decimal.NewFromFloat(multiplier)).IntPart()
}

func usernameForRanking(userId int) string {
	username, _ := GetUsernameById(userId, false)
	return username
}
