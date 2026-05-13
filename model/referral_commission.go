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

	if common.QuotaForInvitee > 0 {
		if err := IncreaseUserQuota(userId, common.QuotaForInvitee, true); err == nil {
			RecordLog(userId, LogTypeSystem, fmt.Sprintf("使用邀请码赠送 %s", logger.LogQuota(common.QuotaForInvitee)))
		}
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
