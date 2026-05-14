package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/logger"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TopUp struct {
	Id              int     `json:"id"`
	UserId          int     `json:"user_id" gorm:"index"`
	Username        string  `json:"username" gorm:"-"`
	Amount          int64   `json:"amount"`
	Money           float64 `json:"money"`
	RefundedMoney   float64 `json:"refunded_money" gorm:"default:0"`
	RefundedQuota   int64   `json:"refunded_quota" gorm:"default:0"`
	TradeNo         string  `json:"trade_no" gorm:"unique;type:varchar(255);index"`
	PaymentMethod   string  `json:"payment_method" gorm:"type:varchar(50)"`
	PaymentProvider string  `json:"payment_provider" gorm:"type:varchar(50);default:''"`
	CreateTime      int64   `json:"create_time"`
	CompleteTime    int64   `json:"complete_time"`
	Status          string  `json:"status"`
}

const (
	PaymentMethodStripe            = "stripe"
	PaymentMethodCreem             = "creem"
	PaymentMethodWaffo             = "waffo"
	PaymentMethodWaffoPancake      = "waffo_pancake"
	PaymentMethodAlipayOfficial    = "alipay_official"
	PaymentMethodWechatPayOfficial = "wxpay_official"
)

const (
	PaymentProviderEpay              = "epay"
	PaymentProviderStripe            = "stripe"
	PaymentProviderCreem             = "creem"
	PaymentProviderWaffo             = "waffo"
	PaymentProviderWaffoPancake      = "waffo_pancake"
	PaymentProviderAlipayOfficial    = "alipay_official"
	PaymentProviderWechatPayOfficial = "wxpay_official"
)

var (
	ErrPaymentMethodMismatch = errors.New("payment method mismatch")
	ErrTopUpNotFound         = errors.New("topup not found")
	ErrTopUpStatusInvalid    = errors.New("topup status invalid")
)

const (
	TopUpRefundStatusPending = "pending"
	TopUpRefundStatusSuccess = "success"
	TopUpRefundStatusFailed  = "failed"
)

type TopUpRefund struct {
	Id              int     `json:"id"`
	TopUpId         int     `json:"topup_id" gorm:"index"`
	UserId          int     `json:"user_id" gorm:"index"`
	TradeNo         string  `json:"trade_no" gorm:"type:varchar(255);index"`
	OutRequestNo    string  `json:"out_request_no" gorm:"unique;type:varchar(255);index"`
	AlipayTradeNo   string  `json:"alipay_trade_no" gorm:"type:varchar(255);default:''"`
	PaymentMethod   string  `json:"payment_method" gorm:"type:varchar(50)"`
	PaymentProvider string  `json:"payment_provider" gorm:"type:varchar(50);default:''"`
	RefundAmount    float64 `json:"refund_amount"`
	RefundQuota     int64   `json:"refund_quota"`
	Reason          string  `json:"reason" gorm:"type:varchar(255);default:''"`
	Status          string  `json:"status" gorm:"type:varchar(32);index"`
	RawResponse     string  `json:"raw_response" gorm:"type:text"`
	CreateTime      int64   `json:"create_time"`
	CompleteTime    int64   `json:"complete_time"`
}

type OfficialPaymentRefundCreateParams struct {
	TradeNo         string
	PaymentProvider string
	PaymentMethod   string
	RefundAmount    float64
	Reason          string
	OutRequestNo    string
}

func (topUp *TopUp) Insert() error {
	var err error
	err = DB.Create(topUp).Error
	return err
}

func (topUp *TopUp) Update() error {
	var err error
	err = DB.Save(topUp).Error
	return err
}

func GetTopUpById(id int) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("id = ?", id).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

func GetTopUpByTradeNo(tradeNo string) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("trade_no = ?", tradeNo).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

func GetTopUpRefundByOutRequestNo(outRequestNo string) *TopUpRefund {
	var refund *TopUpRefund
	if err := DB.Where("out_request_no = ?", outRequestNo).First(&refund).Error; err != nil {
		return nil
	}
	return refund
}

func GetTopUpRefundsByTradeNo(tradeNo string) ([]*TopUpRefund, error) {
	var refunds []*TopUpRefund
	if err := DB.Where("trade_no = ?", tradeNo).Order("id asc").Find(&refunds).Error; err != nil {
		return nil, err
	}
	return refunds, nil
}

func ListPendingTopUpsBefore(paymentProvider string, createBefore int64, limit int) ([]*TopUp, error) {
	if limit <= 0 {
		limit = 20
	}
	var topUps []*TopUp
	err := DB.
		Where("payment_provider = ? AND status = ? AND create_time <= ?", paymentProvider, common.TopUpStatusPending, createBefore).
		Order("id asc").
		Limit(limit).
		Find(&topUps).Error
	return topUps, err
}

func CreateOfficialPaymentRefund(params OfficialPaymentRefundCreateParams) (*TopUpRefund, error) {
	if strings.TrimSpace(params.TradeNo) == "" {
		return nil, errors.New("未提供订单号")
	}
	if strings.TrimSpace(params.OutRequestNo) == "" {
		return nil, errors.New("未提供退款请求号")
	}
	refundAmount := decimal.NewFromFloat(params.RefundAmount).Round(2)
	if !refundAmount.IsPositive() {
		return nil, errors.New("退款金额必须大于 0")
	}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	var refund *TopUpRefund
	err := DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", params.TradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if params.PaymentProvider != "" && topUp.PaymentProvider != params.PaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if params.PaymentMethod != "" && topUp.PaymentMethod != params.PaymentMethod {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status != common.TopUpStatusSuccess &&
			topUp.Status != common.TopUpStatusPartialRefunded {
			return ErrTopUpStatusInvalid
		}

		orderMoney := decimal.NewFromFloat(topUp.Money).Round(2)
		refundedMoney := decimal.NewFromFloat(topUp.RefundedMoney).Round(2)
		remainingMoney := orderMoney.Sub(refundedMoney)
		if remainingMoney.LessThan(refundAmount) {
			return errors.New("退款金额超过可退金额")
		}

		totalQuota := topUpCreditedQuota(*topUp)
		refundQuota := calculateOfficialRefundQuota(totalQuota, topUp.RefundedQuota, orderMoney, refundAmount, remainingMoney)
		if refundQuota <= 0 {
			return errors.New("退款额度无效")
		}

		refund = &TopUpRefund{
			TopUpId:         topUp.Id,
			UserId:          topUp.UserId,
			TradeNo:         topUp.TradeNo,
			OutRequestNo:    strings.TrimSpace(params.OutRequestNo),
			PaymentMethod:   topUp.PaymentMethod,
			PaymentProvider: topUp.PaymentProvider,
			RefundAmount:    refundAmount.InexactFloat64(),
			RefundQuota:     refundQuota,
			Reason:          strings.TrimSpace(params.Reason),
			Status:          TopUpRefundStatusPending,
			CreateTime:      common.GetTimestamp(),
		}
		if err := tx.Create(refund).Error; err != nil {
			return err
		}

		newRefundedMoney := refundedMoney.Add(refundAmount)
		topUp.RefundedMoney = newRefundedMoney.InexactFloat64()
		topUp.RefundedQuota += refundQuota
		topUp.Status = common.TopUpStatusPartialRefunded
		if !newRefundedMoney.LessThan(orderMoney) {
			topUp.Status = common.TopUpStatusRefunded
		}
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota - ?", refundQuota)).Error; err != nil {
			return err
		}
		return reverseTopUpReferralCommissionForRefundTx(tx, topUp.Id, refundAmount, orderMoney)
	})
	if err != nil {
		return nil, err
	}
	return refund, nil
}

func MarkTopUpRefundSuccess(outRequestNo string, alipayTradeNo string, rawResponse string) error {
	if strings.TrimSpace(outRequestNo) == "" {
		return errors.New("未提供退款请求号")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		refund := &TopUpRefund{}
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("out_request_no = ?", outRequestNo).First(refund).Error; err != nil {
			return err
		}
		if refund.Status == TopUpRefundStatusSuccess {
			return nil
		}
		if refund.Status != TopUpRefundStatusPending {
			return ErrTopUpStatusInvalid
		}
		refund.Status = TopUpRefundStatusSuccess
		refund.AlipayTradeNo = strings.TrimSpace(alipayTradeNo)
		refund.RawResponse = rawResponse
		refund.CompleteTime = common.GetTimestamp()
		return tx.Save(refund).Error
	})
}

func MarkTopUpRefundFailed(outRequestNo string, rawResponse string) error {
	if strings.TrimSpace(outRequestNo) == "" {
		return errors.New("未提供退款请求号")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		refund := &TopUpRefund{}
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("out_request_no = ?", outRequestNo).First(refund).Error; err != nil {
			return err
		}
		if refund.Status != TopUpRefundStatusPending {
			return nil
		}
		topUp := &TopUp{}
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", refund.TopUpId).First(topUp).Error; err != nil {
			return err
		}
		if err := reverseTopUpReferralCommissionForRefundTx(tx, refund.TopUpId, decimal.NewFromFloat(refund.RefundAmount).Neg(), decimal.NewFromFloat(topUp.Money).Round(2)); err != nil {
			return err
		}
		refund.Status = TopUpRefundStatusFailed
		refund.RawResponse = rawResponse
		refund.CompleteTime = common.GetTimestamp()
		if err := tx.Save(refund).Error; err != nil {
			return err
		}

		topUp.RefundedMoney = decimal.NewFromFloat(topUp.RefundedMoney).Sub(decimal.NewFromFloat(refund.RefundAmount)).Round(2).InexactFloat64()
		if topUp.RefundedMoney < 0 {
			topUp.RefundedMoney = 0
		}
		topUp.RefundedQuota -= refund.RefundQuota
		if topUp.RefundedQuota < 0 {
			topUp.RefundedQuota = 0
		}
		if topUp.RefundedQuota == 0 && topUp.RefundedMoney == 0 {
			topUp.Status = common.TopUpStatusSuccess
		} else {
			topUp.Status = common.TopUpStatusPartialRefunded
		}
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}
		return tx.Model(&User{}).Where("id = ?", refund.UserId).Update("quota", gorm.Expr("quota + ?", refund.RefundQuota)).Error
	})
}

func calculateOfficialRefundQuota(totalQuota int64, alreadyRefundedQuota int64, orderMoney decimal.Decimal, refundAmount decimal.Decimal, remainingMoney decimal.Decimal) int64 {
	remainingQuota := totalQuota - alreadyRefundedQuota
	if remainingQuota <= 0 {
		return 0
	}
	if !refundAmount.LessThan(remainingMoney) {
		return remainingQuota
	}
	if orderMoney.IsZero() || orderMoney.IsNegative() {
		return 0
	}
	refundQuota := decimal.NewFromInt(totalQuota).Mul(refundAmount).Div(orderMoney).Round(0).IntPart()
	if refundQuota <= 0 {
		refundQuota = 1
	}
	if refundQuota > remainingQuota {
		return remainingQuota
	}
	return refundQuota
}

func reverseTopUpReferralCommissionForRefundTx(tx *gorm.DB, topUpId int, refundAmount decimal.Decimal, orderMoney decimal.Decimal) error {
	if tx == nil || topUpId <= 0 || refundAmount.IsZero() || !orderMoney.IsPositive() {
		return nil
	}
	var commissions []ReferralCommission
	if err := tx.Where("source_type = ? AND source_id = ?", ReferralCommissionSourceTopUp, topUpId).Find(&commissions).Error; err != nil {
		return err
	}
	for _, commission := range commissions {
		if commission.InviterId <= 0 || commission.CommissionQuota <= 0 {
			continue
		}
		absoluteRefundAmount := refundAmount.Abs()
		reverseQuota := decimal.NewFromInt(int64(commission.CommissionQuota)).
			Mul(absoluteRefundAmount).
			Div(orderMoney).
			Round(0).
			IntPart()
		if reverseQuota <= 0 {
			reverseQuota = 1
		}
		if reverseQuota > int64(commission.CommissionQuota) {
			reverseQuota = int64(commission.CommissionQuota)
		}
		expr := gorm.Expr("aff_quota - ?", reverseQuota)
		historyExpr := gorm.Expr("aff_history - ?", reverseQuota)
		if refundAmount.IsNegative() {
			expr = gorm.Expr("aff_quota + ?", reverseQuota)
			historyExpr = gorm.Expr("aff_history + ?", reverseQuota)
		}
		if err := tx.Model(&User{}).
			Where("id = ?", commission.InviterId).
			Updates(map[string]interface{}{
				"aff_quota":   expr,
				"aff_history": historyExpr,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

func UpdatePendingTopUpStatus(tradeNo string, expectedPaymentProvider string, targetStatus string) error {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if expectedPaymentProvider != "" && topUp.PaymentProvider != expectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}

		topUp.Status = targetStatus
		return tx.Save(topUp).Error
	})
}

func CompleteEpayTopUp(tradeNo string, actualPaymentMethod string) (*TopUp, int, *ReferralCommissionCreditResult, bool, error) {
	if tradeNo == "" {
		return nil, 0, nil, false, errors.New("未提供支付单号")
	}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	var completed bool
	var quotaToAdd int
	var topUp TopUp
	var referralResult *ReferralCommissionCreditResult
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(&topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if topUp.PaymentProvider != PaymentProviderEpay {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}
		if topUp.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}
		if actualPaymentMethod != "" && topUp.PaymentMethod != actualPaymentMethod {
			topUp.PaymentMethod = actualPaymentMethod
		}

		dAmount := decimal.NewFromInt(topUp.Amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		quotaToAdd = int(dAmount.Mul(dQuotaPerUnit).IntPart())
		if quotaToAdd <= 0 {
			return errors.New("无效的充值额度")
		}

		topUp.Status = common.TopUpStatusSuccess
		topUp.CompleteTime = common.GetTimestamp()
		if err := tx.Save(&topUp).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}
		var err error
		referralResult, err = CreditInviteRewardsAfterPaymentTx(tx, topUp.UserId, topUp.Money, topUp.PaymentMethod, ReferralCommissionSourceTopUp, topUp.Id)
		if err != nil {
			return err
		}
		completed = true
		return nil
	})
	if err != nil {
		if topUp.Id != 0 {
			return &topUp, quotaToAdd, referralResult, completed, err
		}
		return nil, 0, nil, false, err
	}
	return &topUp, quotaToAdd, referralResult, completed, nil
}

func Recharge(referenceId string, customerId string, callerIp string) (err error) {
	if referenceId == "" {
		return errors.New("未提供支付单号")
	}

	var quota float64
	topUp := &TopUp{}
	var referralResult *ReferralCommissionCreditResult

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", referenceId).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentProvider != PaymentProviderStripe {
			return ErrPaymentMethodMismatch
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		err = tx.Save(topUp).Error
		if err != nil {
			return err
		}

		quota = topUp.Money * common.QuotaPerUnit
		err = tx.Model(&User{}).Where("id = ?", topUp.UserId).Updates(map[string]interface{}{"stripe_customer": customerId, "quota": gorm.Expr("quota + ?", quota)}).Error
		if err != nil {
			return err
		}

		referralResult, err = CreditInviteRewardsAfterPaymentTx(tx, topUp.UserId, topUp.Money, PaymentMethodStripe, ReferralCommissionSourceTopUp, topUp.Id)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError("topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}

	RecordTopupLog(topUp.UserId, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%d", logger.FormatQuota(int(quota)), topUp.Amount), callerIp, topUp.PaymentMethod, PaymentMethodStripe)
	RecordReferralCommissionLog(referralResult)

	return nil
}

// topUpQueryWindowSeconds 限制充值记录查询的时间窗口（秒）。
const topUpQueryWindowSeconds int64 = 30 * 24 * 60 * 60

// topUpQueryCutoff 返回允许查询的最早 create_time（秒级 Unix 时间戳）。
func topUpQueryCutoff() int64 {
	return common.GetTimestamp() - topUpQueryWindowSeconds
}

func fillTopUpUsernames(topUps []*TopUp) {
	if len(topUps) == 0 {
		return
	}
	userIds := make([]int, 0, len(topUps))
	for _, topUp := range topUps {
		if topUp != nil && topUp.UserId > 0 {
			userIds = append(userIds, topUp.UserId)
		}
	}
	userIds = uniqueIntSlice(userIds)
	if len(userIds) == 0 {
		return
	}
	var users []User
	if err := DB.Select("id", "username").Where("id IN ?", userIds).Find(&users).Error; err != nil {
		return
	}
	usernameById := make(map[int]string, len(users))
	for _, user := range users {
		usernameById[user.Id] = user.Username
	}
	for _, topUp := range topUps {
		if topUp != nil {
			topUp.Username = usernameById[topUp.UserId]
		}
	}
}

func parseTopUpUserIDKeyword(keyword string) (int, bool) {
	value, err := strconv.Atoi(strings.TrimSpace(keyword))
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

func findUserIDsByUsernamePattern(tx *gorm.DB, pattern string) ([]int, error) {
	var users []User
	if err := tx.Select("id").Where("username LIKE ? ESCAPE '!'", pattern).Limit(searchTopUpCountHardLimit).Find(&users).Error; err != nil {
		return nil, err
	}
	userIds := make([]int, 0, len(users))
	for _, user := range users {
		userIds = append(userIds, user.Id)
	}
	return userIds, nil
}

func buildTopUpContainsLikePattern(keyword string) (string, error) {
	pattern, err := sanitizeLikePattern(keyword)
	if err != nil {
		return "", err
	}
	if strings.Count(keyword, "%") == 0 {
		pattern = "%" + pattern + "%"
	}
	return pattern, nil
}

func uniqueIntSlice(values []int) []int {
	if len(values) == 0 {
		return []int{-1}
	}
	seen := make(map[int]struct{}, len(values))
	unique := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	if len(unique) == 0 {
		return []int{-1}
	}
	return unique
}

func GetUserTopUps(userId int, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	// Start transaction
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	cutoff := topUpQueryCutoff()

	// Get total count within transaction
	err = tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, cutoff).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated topups within same transaction
	err = tx.Where("user_id = ? AND create_time >= ?", userId, cutoff).Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	fillTopUpUsernames(topups)

	return topups, total, nil
}

// GetAllTopUps 获取全平台的充值记录（管理员使用，不限制时间窗口）
func GetAllTopUps(pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err = tx.Model(&TopUp{}).Count(&total).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	fillTopUpUsernames(topups)

	return topups, total, nil
}

// searchTopUpCountHardLimit 搜索充值记录时 COUNT 的安全上限，
// 防止对超大表执行无界 COUNT 触发 DoS。
const searchTopUpCountHardLimit = 10000

// SearchUserTopUps 按订单号搜索某用户的充值记录
func SearchUserTopUps(userId int, keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, topUpQueryCutoff())
	if keyword != "" {
		pattern, perr := buildTopUpContainsLikePattern(keyword)
		if perr != nil {
			tx.Rollback()
			return nil, 0, perr
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", pattern)
	}

	if err = query.Limit(searchTopUpCountHardLimit).Count(&total).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	fillTopUpUsernames(topups)
	return topups, total, nil
}

// SearchAllTopUps 按用户 ID、用户名或订单号搜索全平台充值记录（管理员使用，不限制时间窗口）
func SearchAllTopUps(keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&TopUp{})
	if keyword != "" {
		pattern, perr := buildTopUpContainsLikePattern(keyword)
		if perr != nil {
			tx.Rollback()
			return nil, 0, perr
		}
		userIdKeyword, hasUserIdKeyword := parseTopUpUserIDKeyword(keyword)
		matchedUserIds, matchErr := findUserIDsByUsernamePattern(tx, pattern)
		if matchErr != nil {
			tx.Rollback()
			return nil, 0, matchErr
		}
		if hasUserIdKeyword {
			matchedUserIds = append(matchedUserIds, userIdKeyword)
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!' OR user_id IN ?", pattern, uniqueIntSlice(matchedUserIds))
	}

	if err = query.Limit(searchTopUpCountHardLimit).Count(&total).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	fillTopUpUsernames(topups)
	return topups, total, nil
}

// ManualCompleteTopUp 管理员手动完成订单并给用户充值
func ManualCompleteTopUp(tradeNo string, callerIp string) error {
	if tradeNo == "" {
		return errors.New("未提供订单号")
	}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	var userId int
	var quotaToAdd int
	var payMoney float64
	var paymentMethod string
	var topUpId int
	var completed bool
	var referralResult *ReferralCommissionCreditResult

	err := DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		// 行级锁，避免并发补单
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return errors.New("充值订单不存在")
		}

		// 幂等处理：已成功直接返回
		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("订单状态不是待支付，无法补单")
		}

		// 计算应充值额度：
		// - Stripe 订单：Money 代表经分组倍率换算后的美元数量，直接 * QuotaPerUnit
		// - 其他订单（如易支付）：Amount 为美元数量，* QuotaPerUnit
		if topUp.PaymentProvider == PaymentProviderStripe {
			dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
			quotaToAdd = int(decimal.NewFromFloat(topUp.Money).Mul(dQuotaPerUnit).IntPart())
		} else {
			dAmount := decimal.NewFromInt(topUp.Amount)
			dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
			quotaToAdd = int(dAmount.Mul(dQuotaPerUnit).IntPart())
		}
		if quotaToAdd <= 0 {
			return errors.New("无效的充值额度")
		}

		// 标记完成
		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		// 增加用户额度（立即写库，保持一致性）
		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}

		userId = topUp.UserId
		payMoney = topUp.Money
		paymentMethod = topUp.PaymentMethod
		topUpId = topUp.Id
		completed = true

		var referralErr error
		referralResult, referralErr = CreditInviteRewardsAfterPaymentTx(tx, topUp.UserId, topUp.Money, "manual", ReferralCommissionSourceTopUp, topUp.Id)
		if referralErr != nil {
			return referralErr
		}
		return nil
	})

	if err != nil {
		return err
	}

	if completed {
		// 事务外记录日志，避免阻塞
		RecordTopupLog(userId, fmt.Sprintf("管理员补单成功，充值金额: %v，支付金额：%f", logger.FormatQuota(quotaToAdd), payMoney), callerIp, paymentMethod, "admin")
		common.SysLog(fmt.Sprintf("管理员补单成功 topup_id=%d", topUpId))
		RecordReferralCommissionLog(referralResult)
	}
	return nil
}
func RechargeCreem(referenceId string, customerEmail string, customerName string, callerIp string) (err error) {
	if referenceId == "" {
		return errors.New("未提供支付单号")
	}

	var quota int64
	topUp := &TopUp{}
	var referralResult *ReferralCommissionCreditResult

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", referenceId).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentProvider != PaymentProviderCreem {
			return ErrPaymentMethodMismatch
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		err = tx.Save(topUp).Error
		if err != nil {
			return err
		}

		// Creem 直接使用 Amount 作为充值额度（整数）
		quota = topUp.Amount

		// 构建更新字段，优先使用邮箱，如果邮箱为空则使用用户名
		updateFields := map[string]interface{}{
			"quota": gorm.Expr("quota + ?", quota),
		}

		// 如果有客户邮箱，尝试更新用户邮箱（仅当用户邮箱为空时）
		if customerEmail != "" {
			// 先检查用户当前邮箱是否为空
			var user User
			err = tx.Where("id = ?", topUp.UserId).First(&user).Error
			if err != nil {
				return err
			}

			// 如果用户邮箱为空，则更新为支付时使用的邮箱
			if user.Email == "" {
				updateFields["email"] = customerEmail
			}
		}

		err = tx.Model(&User{}).Where("id = ?", topUp.UserId).Updates(updateFields).Error
		if err != nil {
			return err
		}

		referralResult, err = CreditInviteRewardsAfterPaymentTx(tx, topUp.UserId, topUp.Money, PaymentMethodCreem, ReferralCommissionSourceTopUp, topUp.Id)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError("creem topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}

	RecordTopupLog(topUp.UserId, fmt.Sprintf("使用Creem充值成功，充值额度: %v，支付金额：%.2f", quota, topUp.Money), callerIp, topUp.PaymentMethod, PaymentMethodCreem)
	RecordReferralCommissionLog(referralResult)

	return nil
}

func RechargeWaffo(tradeNo string, callerIp string) (err error) {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}
	var referralResult *ReferralCommissionCreditResult

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentProvider != PaymentProviderWaffo {
			return ErrPaymentMethodMismatch
		}

		if topUp.Status == common.TopUpStatusSuccess {
			return nil // 幂等：已成功直接返回
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		dAmount := decimal.NewFromInt(topUp.Amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		quotaToAdd = int(dAmount.Mul(dQuotaPerUnit).IntPart())
		if quotaToAdd <= 0 {
			return errors.New("无效的充值额度")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}

		referralResult, err = CreditInviteRewardsAfterPaymentTx(tx, topUp.UserId, topUp.Money, PaymentMethodWaffo, ReferralCommissionSourceTopUp, topUp.Id)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError("waffo topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}

	if quotaToAdd > 0 {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("Waffo充值成功，充值额度: %v，支付金额: %.2f", logger.FormatQuota(quotaToAdd), topUp.Money), callerIp, topUp.PaymentMethod, PaymentMethodWaffo)
		RecordReferralCommissionLog(referralResult)
	}

	return nil
}

func RechargeWaffoPancake(tradeNo string) (err error) {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}
	var referralResult *ReferralCommissionCreditResult

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.PaymentProvider != PaymentProviderWaffoPancake {
			return ErrPaymentMethodMismatch
		}

		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		quotaToAdd = int(decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
		if quotaToAdd <= 0 {
			return errors.New("无效的充值额度")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}

		referralResult, err = CreditInviteRewardsAfterPaymentTx(tx, topUp.UserId, topUp.Money, PaymentMethodWaffoPancake, ReferralCommissionSourceTopUp, topUp.Id)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError("waffo pancake topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}

	if quotaToAdd > 0 {
		RecordLog(topUp.UserId, LogTypeTopup, fmt.Sprintf("Waffo Pancake充值成功，充值额度: %v，支付金额: %.2f", logger.FormatQuota(quotaToAdd), topUp.Money))
		RecordReferralCommissionLog(referralResult)
	}

	return nil
}

func RechargeOfficialPayment(tradeNo string, expectedPaymentProvider string, actualPaymentMethod string, callerIp string, paidMoney ...float64) (err error) {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}
	var referralResult *ReferralCommissionCreditResult

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if expectedPaymentProvider != "" && topUp.PaymentProvider != expectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}

		if topUp.Status == common.TopUpStatusSuccess ||
			topUp.Status == common.TopUpStatusPartialRefunded ||
			topUp.Status == common.TopUpStatusRefunded {
			return nil
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		if len(paidMoney) > 0 {
			expectedMoney := decimal.NewFromFloat(topUp.Money).Round(2)
			actualMoney := decimal.NewFromFloat(paidMoney[0]).Round(2)
			if !expectedMoney.Equal(actualMoney) {
				return errors.New("支付金额与订单金额不一致")
			}
		}

		quotaToAdd = int(decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
		if quotaToAdd <= 0 {
			return errors.New("无效的充值额度")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if actualPaymentMethod != "" {
			topUp.PaymentMethod = actualPaymentMethod
		}
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}

		commissionPaymentMethod := actualPaymentMethod
		if commissionPaymentMethod == "" {
			commissionPaymentMethod = topUp.PaymentMethod
		}
		referralResult, err = CreditInviteRewardsAfterPaymentTx(tx, topUp.UserId, topUp.Money, commissionPaymentMethod, ReferralCommissionSourceTopUp, topUp.Id)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError("official payment topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}

	if quotaToAdd > 0 {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("官方支付充值成功，充值额度: %v，支付金额: %.2f", logger.FormatQuota(quotaToAdd), topUp.Money), callerIp, topUp.PaymentMethod, expectedPaymentProvider)
		RecordReferralCommissionLog(referralResult)
	}

	return nil
}
