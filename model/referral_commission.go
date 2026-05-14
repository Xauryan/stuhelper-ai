package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/logger"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ReferralCommissionSourceTopUp        = "topup"
	ReferralCommissionSourceSubscription = "subscription"
)

type ReferralCommission struct {
	Id              int     `json:"id" gorm:"primaryKey;index:idx_referral_commissions_inviter_id,priority:2"`
	InviterId       int     `json:"inviter_id" gorm:"index;index:idx_referral_commissions_inviter_id,priority:1"`
	InviteeId       int     `json:"invitee_id" gorm:"index;uniqueIndex:idx_referral_commission_source,priority:1"`
	SourceType      string  `json:"source_type" gorm:"type:varchar(32);uniqueIndex:idx_referral_commission_source,priority:2"`
	SourceId        int     `json:"source_id" gorm:"uniqueIndex:idx_referral_commission_source,priority:3"`
	PaymentMethod   string  `json:"payment_method" gorm:"type:varchar(50);uniqueIndex:idx_referral_commission_source,priority:4"`
	RechargeAmount  float64 `json:"recharge_amount"`
	CommissionQuota int     `json:"commission_quota"`
	CommissionRate  float64 `json:"commission_rate"`
	CreatedAt       int64   `json:"created_at" gorm:"autoCreateTime"`
}

type ReferralCommissionWithUser struct {
	ReferralCommission
	InviteeUsername string `json:"invitee_username"`
}

const (
	AdminReferralRewardStatusAll      = ""
	AdminReferralRewardStatusUnlocked = "unlocked"
	AdminReferralRewardStatusPending  = "pending"
)

type AdminReferralQuery struct {
	PageInfo     *common.PageInfo
	Keyword      string
	RewardStatus string
}

type AdminReferralRecord struct {
	InviterId             int     `json:"inviter_id"`
	InviterUsername       string  `json:"inviter_username"`
	InviterDisplayName    string  `json:"inviter_display_name"`
	InviteeId             int     `json:"invitee_id"`
	InviteeUsername       string  `json:"invitee_username"`
	InviteeDisplayName    string  `json:"invitee_display_name"`
	InviteeEmail          string  `json:"invitee_email"`
	InviteeCreatedAt      int64   `json:"invitee_created_at"`
	InviteeRewardQuota    int     `json:"invitee_reward_quota"`
	InviterRewardQuota    int     `json:"inviter_reward_quota"`
	InviterRewardUnlocked bool    `json:"inviter_reward_unlocked"`
	InviteeHasPaid        bool    `json:"invitee_has_paid"`
	FirstPaymentTime      int64   `json:"first_payment_time"`
	CommissionCount       int     `json:"commission_count"`
	TotalCommissionQuota  int     `json:"total_commission_quota"`
	TotalRechargeAmount   float64 `json:"total_recharge_amount"`
	LastCommissionAt      int64   `json:"last_commission_at"`
}

type ReferralCommissionCreditResult struct {
	Credited              bool
	CommissionCredited    bool
	InviterRewardCredited bool
	InviterRewardQuota    int
	InviterId             int
	InviteeId             int
	CommissionQuota       int
	CommissionRate        float64
	RechargeAmount        float64
}

func FinalizeUserInvitation(userId int, inviterId int) {
	if inviterId <= 0 {
		return
	}

	inviteeRewardQuota := 0
	if common.QuotaForInvitee > 0 {
		if err := IncreaseUserQuota(userId, common.QuotaForInvitee, true); err == nil {
			inviteeRewardQuota = common.QuotaForInvitee
			RecordLog(userId, LogTypeSystem, fmt.Sprintf("使用邀请码赠送 %s", logger.LogQuota(common.QuotaForInvitee)))
		}
	}
	if err := setInviteeRewardQuota(userId, inviteeRewardQuota); err != nil {
		common.SysLog("failed to snapshot invitee reward quota: " + err.Error())
	}

	inviterQuota := 0
	if common.InviterRewardAfterPaymentEnabled {
		_ = DB.Transaction(func(tx *gorm.DB) error {
			if err := recordUserInvitationTx(tx, inviterId, 0); err != nil {
				return err
			}
			if err := setInviterRewardQuotaTx(tx, userId, common.QuotaForInviter); err != nil {
				return err
			}
			if common.QuotaForInviter <= 0 {
				return markInviterRewardUnlockedTx(tx, userId)
			}
			return nil
		})
		return
	}

	inviterQuota = common.QuotaForInviter
	if err := DB.Transaction(func(tx *gorm.DB) error {
		if err := recordUserInvitationTx(tx, inviterId, inviterQuota); err != nil {
			return err
		}
		if err := setInviterRewardQuotaTx(tx, userId, inviterQuota); err != nil {
			return err
		}
		return markInviterRewardUnlockedTx(tx, userId)
	}); err == nil && inviterQuota > 0 {
		RecordLog(inviterId, LogTypeSystem, fmt.Sprintf("邀请用户赠送 %s", logger.LogQuota(inviterQuota)))
	}
}

func normalizeReferralCommissionSource(sourceType string) string {
	switch strings.TrimSpace(sourceType) {
	case ReferralCommissionSourceTopUp:
		return ReferralCommissionSourceTopUp
	case ReferralCommissionSourceSubscription:
		return ReferralCommissionSourceSubscription
	default:
		return ""
	}
}

func effectiveReferralCommissionRate(inviter *User) float64 {
	if inviter != nil && inviter.ReferralCommissionPercent != nil {
		rate := *inviter.ReferralCommissionPercent
		if rate >= 0 && rate <= 100 {
			return rate
		}
	}
	return common.ReferralCommissionPercent
}

func calculateReferralCommissionQuota(rechargeAmount float64, rate float64) int {
	if rechargeAmount <= 0 || rate <= 0 || rate > 100 || common.QuotaPerUnit <= 0 {
		return 0
	}
	return int(decimal.NewFromFloat(rechargeAmount).
		Mul(decimal.NewFromFloat(common.QuotaPerUnit)).
		Mul(decimal.NewFromFloat(rate)).
		Div(decimal.NewFromInt(100)).
		IntPart())
}

func CreditReferralCommission(userId int, rechargeAmount float64, paymentMethod string, sourceType string, sourceId int) (bool, error) {
	var result *ReferralCommissionCreditResult
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = CreditReferralCommissionTx(tx, userId, rechargeAmount, paymentMethod, sourceType, sourceId)
		return err
	})
	if err != nil {
		return false, err
	}
	RecordReferralCommissionLog(result)
	return result != nil && result.Credited, nil
}

func CreditInviteRewardsAfterPaymentTx(tx *gorm.DB, userId int, rechargeAmount float64, paymentMethod string, sourceType string, sourceId int) (*ReferralCommissionCreditResult, error) {
	result := &ReferralCommissionCreditResult{}
	inviterRewardCredited, rewardInviterId, rewardInviteeId, rewardQuota, err := creditInviterRewardAfterPaymentTx(tx, userId)
	if err != nil {
		return nil, err
	}
	result.InviterRewardCredited = inviterRewardCredited
	result.InviterRewardQuota = rewardQuota
	result.InviterId = rewardInviterId
	result.InviteeId = rewardInviteeId

	commissionResult, err := CreditReferralCommissionTx(tx, userId, rechargeAmount, paymentMethod, sourceType, sourceId)
	if err != nil {
		return nil, err
	}
	if commissionResult != nil {
		result.CommissionCredited = commissionResult.CommissionCredited
		if commissionResult.InviterId > 0 {
			result.InviterId = commissionResult.InviterId
		}
		if commissionResult.InviteeId > 0 {
			result.InviteeId = commissionResult.InviteeId
		}
		result.CommissionQuota = commissionResult.CommissionQuota
		result.CommissionRate = commissionResult.CommissionRate
		result.RechargeAmount = commissionResult.RechargeAmount
	}
	result.Credited = result.InviterRewardCredited || result.CommissionCredited
	return result, nil
}

func CreditReferralCommissionTx(tx *gorm.DB, userId int, rechargeAmount float64, paymentMethod string, sourceType string, sourceId int) (*ReferralCommissionCreditResult, error) {
	if tx == nil {
		return nil, errors.New("tx is nil")
	}
	sourceType = normalizeReferralCommissionSource(sourceType)
	paymentMethod = strings.TrimSpace(paymentMethod)
	if !common.ReferralCommissionEnabled || rechargeAmount <= 0 || sourceType == "" || sourceId <= 0 || paymentMethod == "" {
		return &ReferralCommissionCreditResult{}, nil
	}

	var invitee User
	if err := tx.Set("gorm:query_option", "FOR UPDATE").
		Select("id", "inviter_id").
		Where("id = ?", userId).
		First(&invitee).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &ReferralCommissionCreditResult{}, nil
		}
		return nil, err
	}
	if invitee.InviterId <= 0 || invitee.InviterId == invitee.Id {
		return &ReferralCommissionCreditResult{}, nil
	}

	var inviter User
	if err := tx.Set("gorm:query_option", "FOR UPDATE").
		Where("id = ?", invitee.InviterId).
		First(&inviter).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &ReferralCommissionCreditResult{}, nil
		}
		return nil, err
	}

	rate := effectiveReferralCommissionRate(&inviter)
	commissionQuota := calculateReferralCommissionQuota(rechargeAmount, rate)
	if commissionQuota <= 0 {
		return &ReferralCommissionCreditResult{}, nil
	}

	if common.ReferralCommissionMaxRecharges > 0 {
		var count int64
		if err := tx.Model(&ReferralCommission{}).
			Where("invitee_id = ?", invitee.Id).
			Count(&count).Error; err != nil {
			return nil, err
		}
		if count >= int64(common.ReferralCommissionMaxRecharges) {
			return &ReferralCommissionCreditResult{}, nil
		}
	}

	commission := &ReferralCommission{
		InviterId:       invitee.InviterId,
		InviteeId:       invitee.Id,
		SourceType:      sourceType,
		SourceId:        sourceId,
		PaymentMethod:   paymentMethod,
		RechargeAmount:  rechargeAmount,
		CommissionQuota: commissionQuota,
		CommissionRate:  rate,
	}
	insertResult := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(commission)
	if insertResult.Error != nil {
		return nil, insertResult.Error
	}
	if insertResult.RowsAffected == 0 {
		return &ReferralCommissionCreditResult{}, nil
	}

	if err := tx.Model(&User{}).
		Where("id = ?", invitee.InviterId).
		Updates(map[string]interface{}{
			"aff_quota":   gorm.Expr("aff_quota + ?", commissionQuota),
			"aff_history": gorm.Expr("aff_history + ?", commissionQuota),
		}).Error; err != nil {
		return nil, err
	}

	return &ReferralCommissionCreditResult{
		Credited:           true,
		CommissionCredited: true,
		InviterId:          invitee.InviterId,
		InviteeId:          invitee.Id,
		CommissionQuota:    commissionQuota,
		CommissionRate:     rate,
		RechargeAmount:     rechargeAmount,
	}, nil
}

func RecordReferralCommissionLog(result *ReferralCommissionCreditResult) {
	if result == nil {
		return
	}
	if result.InviterRewardCredited {
		RecordLog(
			result.InviterId,
			LogTypeSystem,
			fmt.Sprintf("邀请用户赠送 %s", logger.LogQuota(result.InviterRewardQuota)),
		)
	}
	if !result.CommissionCredited {
		return
	}
	RecordLog(
		result.InviterId,
		LogTypeSystem,
		fmt.Sprintf("邀请用户充值返佣 %s (%.2f%% of %.2f)", logger.LogQuota(result.CommissionQuota), result.CommissionRate, result.RechargeAmount),
	)
}

func creditInviterRewardAfterPaymentTx(tx *gorm.DB, userId int) (bool, int, int, int, error) {
	return creditInviterRewardTx(tx, userId)
}

func creditInviterRewardTx(tx *gorm.DB, userId int) (bool, int, int, int, error) {
	if tx == nil {
		return false, 0, 0, 0, errors.New("tx is nil")
	}
	if userId <= 0 {
		return false, 0, 0, 0, nil
	}

	var invitee User
	if err := tx.Set("gorm:query_option", "FOR UPDATE").
		Select("id", "inviter_id", "inviter_reward_quota", "inviter_reward_unlocked").
		Where("id = ?", userId).
		First(&invitee).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, 0, 0, 0, nil
		}
		return false, 0, 0, 0, err
	}
	if invitee.InviterId <= 0 || invitee.InviterRewardUnlocked {
		return false, invitee.InviterId, invitee.Id, invitee.InviterRewardQuota, nil
	}
	if invitee.InviterId == invitee.Id {
		return false, invitee.InviterId, invitee.Id, invitee.InviterRewardQuota, nil
	}
	if invitee.InviterRewardQuota <= 0 {
		if err := markInviterRewardUnlockedTx(tx, invitee.Id); err != nil {
			return false, 0, 0, 0, err
		}
		return false, invitee.InviterId, invitee.Id, invitee.InviterRewardQuota, nil
	}

	result := tx.Model(&User{}).
		Where("id = ? AND inviter_reward_unlocked = ?", invitee.Id, false).
		Update("inviter_reward_unlocked", true)
	if result.Error != nil {
		return false, 0, 0, 0, result.Error
	}
	if result.RowsAffected == 0 {
		return false, invitee.InviterId, invitee.Id, invitee.InviterRewardQuota, nil
	}

	if err := tx.Model(&User{}).
		Where("id = ?", invitee.InviterId).
		Updates(map[string]interface{}{
			"aff_quota":   gorm.Expr("aff_quota + ?", invitee.InviterRewardQuota),
			"aff_history": gorm.Expr("aff_history + ?", invitee.InviterRewardQuota),
		}).Error; err != nil {
		return false, 0, 0, 0, err
	}
	return true, invitee.InviterId, invitee.Id, invitee.InviterRewardQuota, nil
}

func setInviterRewardQuotaTx(tx *gorm.DB, userId int, quota int) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if userId <= 0 {
		return nil
	}
	return tx.Model(&User{}).
		Where("id = ?", userId).
		Update("inviter_reward_quota", quota).Error
}

func setInviteeRewardQuota(userId int, quota int) error {
	if userId <= 0 {
		return nil
	}
	return DB.Model(&User{}).
		Where("id = ?", userId).
		Update("invitee_reward_quota", quota).Error
}

func markInviterRewardUnlocked(userId int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		return markInviterRewardUnlockedTx(tx, userId)
	})
}

func markInviterRewardUnlockedTx(tx *gorm.DB, userId int) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if userId <= 0 {
		return nil
	}
	return tx.Model(&User{}).
		Where("id = ?", userId).
		Update("inviter_reward_unlocked", true).Error
}

func GetUserReferralCommissions(inviterId int, pageInfo *common.PageInfo) ([]*ReferralCommissionWithUser, int64, error) {
	var total int64
	var commissions []*ReferralCommissionWithUser
	if pageInfo == nil {
		pageInfo = &common.PageInfo{}
	}
	if pageInfo.Page < 1 {
		pageInfo.Page = 1
	}
	if pageInfo.PageSize <= 0 || pageInfo.PageSize > 100 {
		pageInfo.PageSize = common.ItemsPerPage
	}

	query := DB.Table("referral_commissions").
		Select("referral_commissions.*, users.username as invitee_username").
		Joins("LEFT JOIN users ON users.id = referral_commissions.invitee_id").
		Where("referral_commissions.inviter_id = ?", inviterId)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("referral_commissions.id desc").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Find(&commissions).Error
	return commissions, total, err
}

func GetAdminReferralRecords(query *AdminReferralQuery) ([]*AdminReferralRecord, int64, error) {
	if query == nil {
		query = &AdminReferralQuery{}
	}
	pageInfo := normalizeReferralPageInfo(query.PageInfo)
	keyword := strings.TrimSpace(query.Keyword)
	rewardStatus := strings.TrimSpace(query.RewardStatus)

	baseQuery := DB.Table("users AS invitees").
		Joins("JOIN users AS inviters ON inviters.id = invitees.inviter_id").
		Where("invitees.inviter_id > 0")

	if keyword != "" {
		like := "%" + keyword + "%"
		if keywordId, err := decimal.NewFromString(keyword); err == nil && keywordId.IsInteger() {
			baseQuery = baseQuery.Where(
				"(invitees.id = ? OR inviters.id = ? OR invitees.username LIKE ? OR inviters.username LIKE ? OR invitees.email LIKE ? OR invitees.display_name LIKE ? OR inviters.display_name LIKE ?)",
				int(keywordId.IntPart()), int(keywordId.IntPart()), like, like, like, like, like,
			)
		} else {
			baseQuery = baseQuery.Where(
				"(invitees.username LIKE ? OR inviters.username LIKE ? OR invitees.email LIKE ? OR invitees.display_name LIKE ? OR inviters.display_name LIKE ?)",
				like, like, like, like, like,
			)
		}
	}

	switch rewardStatus {
	case AdminReferralRewardStatusUnlocked:
		baseQuery = baseQuery.Where("invitees.inviter_reward_unlocked = ?", true)
	case AdminReferralRewardStatusPending:
		baseQuery = baseQuery.Where("invitees.inviter_reward_unlocked = ? AND invitees.inviter_reward_quota > ?", false, 0)
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	selectColumns := strings.Join([]string{
		"invitees.inviter_id",
		"inviters.username AS inviter_username",
		"inviters.display_name AS inviter_display_name",
		"invitees.id AS invitee_id",
		"invitees.username AS invitee_username",
		"invitees.display_name AS invitee_display_name",
		"invitees.email AS invitee_email",
		"invitees.created_at AS invitee_created_at",
		"invitees.invitee_reward_quota",
		"invitees.inviter_reward_quota",
		"invitees.inviter_reward_unlocked",
	}, ", ")

	records := make([]*AdminReferralRecord, 0)
	if err := baseQuery.Select(selectColumns).
		Order("invitees.id desc").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Find(&records).Error; err != nil {
		return nil, 0, err
	}
	if len(records) == 0 {
		return records, total, nil
	}

	inviteeIds := make([]int, 0, len(records))
	for _, record := range records {
		inviteeIds = append(inviteeIds, record.InviteeId)
	}
	if err := fillAdminReferralPaymentState(records, inviteeIds); err != nil {
		return nil, 0, err
	}
	if err := fillAdminReferralCommissionSummary(records, inviteeIds); err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func normalizeReferralPageInfo(pageInfo *common.PageInfo) *common.PageInfo {
	if pageInfo == nil {
		pageInfo = &common.PageInfo{}
	}
	if pageInfo.Page < 1 {
		pageInfo.Page = 1
	}
	if pageInfo.PageSize <= 0 || pageInfo.PageSize > 100 {
		pageInfo.PageSize = common.ItemsPerPage
	}
	return pageInfo
}

type referralPaymentState struct {
	UserId           int
	FirstPaymentTime int64
}

func fillAdminReferralPaymentState(records []*AdminReferralRecord, inviteeIds []int) error {
	stateByInviteeId := make(map[int]*referralPaymentState, len(inviteeIds))
	var topUpStates []*referralPaymentState
	if err := DB.Table("top_ups").
		Select("user_id, MIN(complete_time) AS first_payment_time").
		Where("user_id IN ? AND status IN ? AND complete_time > ?", inviteeIds, []string{
			common.TopUpStatusSuccess,
			common.TopUpStatusPartialRefunded,
			common.TopUpStatusRefunded,
		}, 0).
		Group("user_id").
		Find(&topUpStates).Error; err != nil {
		return err
	}
	for _, state := range topUpStates {
		if state == nil {
			continue
		}
		stateByInviteeId[state.UserId] = state
	}

	var subscriptionStates []*referralPaymentState
	if err := DB.Table("subscription_orders").
		Select("user_id, MIN(complete_time) AS first_payment_time").
		Where("user_id IN ? AND status = ? AND complete_time > ?", inviteeIds, common.TopUpStatusSuccess, 0).
		Group("user_id").
		Find(&subscriptionStates).Error; err != nil {
		return err
	}
	for _, state := range subscriptionStates {
		if state == nil {
			continue
		}
		existing := stateByInviteeId[state.UserId]
		if existing == nil || state.FirstPaymentTime < existing.FirstPaymentTime {
			stateByInviteeId[state.UserId] = state
		}
	}

	for _, record := range records {
		state := stateByInviteeId[record.InviteeId]
		if state == nil {
			continue
		}
		record.InviteeHasPaid = true
		record.FirstPaymentTime = state.FirstPaymentTime
	}
	return nil
}

type adminReferralCommissionSummary struct {
	InviteeId            int
	CommissionCount      int
	TotalCommissionQuota int
	TotalRechargeAmount  float64
	LastCommissionAt     int64
}

func fillAdminReferralCommissionSummary(records []*AdminReferralRecord, inviteeIds []int) error {
	summaryByInviteeId := make(map[int]*adminReferralCommissionSummary, len(records))
	var summaries []*adminReferralCommissionSummary
	if err := DB.Table("referral_commissions").
		Select("invitee_id, COUNT(*) AS commission_count, SUM(commission_quota) AS total_commission_quota, SUM(recharge_amount) AS total_recharge_amount, MAX(created_at) AS last_commission_at").
		Where("referral_commissions.invitee_id IN ?", inviteeIds).
		Group("invitee_id").
		Find(&summaries).Error; err != nil {
		return err
	}
	for _, summary := range summaries {
		if summary == nil {
			continue
		}
		summaryByInviteeId[summary.InviteeId] = summary
	}
	for _, record := range records {
		summary := summaryByInviteeId[record.InviteeId]
		if summary == nil {
			continue
		}
		record.CommissionCount = summary.CommissionCount
		record.TotalCommissionQuota = summary.TotalCommissionQuota
		record.TotalRechargeAmount = summary.TotalRechargeAmount
		record.LastCommissionAt = summary.LastCommissionAt
	}
	return nil
}

func GetAdminReferralCommissions(inviteeId int, pageInfo *common.PageInfo) ([]*ReferralCommissionWithUser, int64, error) {
	var total int64
	commissions := make([]*ReferralCommissionWithUser, 0)
	if inviteeId <= 0 {
		return commissions, 0, nil
	}
	pageInfo = normalizeReferralPageInfo(pageInfo)
	query := DB.Table("referral_commissions").
		Select("referral_commissions.*, users.username AS invitee_username").
		Joins("LEFT JOIN users ON users.id = referral_commissions.invitee_id").
		Where("referral_commissions.invitee_id = ?", inviteeId)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("referral_commissions.id desc").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Find(&commissions).Error; err != nil {
		return nil, 0, err
	}
	return commissions, total, nil
}
