package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/logger"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TopUp struct {
	Id                  int     `json:"id"`
	UserId              int     `json:"user_id" gorm:"index;index:idx_topup_user_create,priority:1"`
	Username            string  `json:"username" gorm:"-"`
	Amount              int64   `json:"amount"`
	Money               float64 `json:"money"`
	Fee                 float64 `json:"fee" gorm:"default:0"`
	RefundedMoney       float64 `json:"refunded_money" gorm:"default:0"`
	RefundedQuota       int64   `json:"refunded_quota" gorm:"default:0"`
	RefundRequestId     int     `json:"refund_request_id" gorm:"-"`
	RefundRequestStatus string  `json:"refund_request_status" gorm:"-"`
	RefundRequestAmount float64 `json:"refund_request_amount" gorm:"-"`
	RefundRequestReason string  `json:"refund_request_reason" gorm:"-"`
	TradeNo             string  `json:"trade_no" gorm:"unique;type:varchar(255);index"`
	PaymentMethod       string  `json:"payment_method" gorm:"type:varchar(50)"`
	PaymentProvider     string  `json:"payment_provider" gorm:"type:varchar(50);default:''"`
	CreateTime          int64   `json:"create_time" gorm:"index:idx_topup_user_create,priority:2"`
	CompleteTime        int64   `json:"complete_time"`
	Status              string  `json:"status"`
}

func (topUp TopUp) PaidMoney() float64 {
	money := decimal.NewFromFloat(topUp.Money).Round(2)
	fee := decimal.NewFromFloat(topUp.Fee).Round(2)
	if fee.IsNegative() {
		fee = decimal.Zero
	}
	return money.Add(fee).Round(2).InexactFloat64()
}

const (
	PaymentMethodStripe            = "stripe"
	PaymentMethodCreem             = "creem"
	PaymentMethodWaffo             = "waffo"
	PaymentMethodWaffoPancake      = "waffo_pancake"
	PaymentMethodAlipayOfficial    = "alipay_official"
	PaymentMethodWechatPayOfficial = "wxpay_official"
	PaymentMethodAdminAdd          = "admin_add"
)

const (
	PaymentProviderEpay              = "epay"
	PaymentProviderStripe            = "stripe"
	PaymentProviderCreem             = "creem"
	PaymentProviderWaffo             = "waffo"
	PaymentProviderWaffoPancake      = "waffo_pancake"
	PaymentProviderAlipayOfficial    = "alipay_official"
	PaymentProviderWechatPayOfficial = "wxpay_official"
	PaymentProviderAdmin             = "admin"
)

func IsOfficialPaymentProvider(paymentProvider string) bool {
	return paymentProvider == PaymentProviderAlipayOfficial ||
		paymentProvider == PaymentProviderWechatPayOfficial
}

func IsAdminTopUpRecord(topUp *TopUp) bool {
	return topUp != nil &&
		(topUp.PaymentProvider == PaymentProviderAdmin ||
			topUp.PaymentMethod == PaymentMethodAdminAdd)
}

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

const (
	TopUpRefundRequestStatusPending  = "pending"
	TopUpRefundRequestStatusApproved = "approved"
	TopUpRefundRequestStatusRejected = "rejected"
	TopUpRefundRequestStatusFailed   = "failed"
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

type TopUpRefundRequest struct {
	Id                    int     `json:"id"`
	TopUpId               int     `json:"topup_id" gorm:"index"`
	UserId                int     `json:"user_id" gorm:"index"`
	TradeNo               string  `json:"trade_no" gorm:"type:varchar(255);index"`
	PaymentMethod         string  `json:"payment_method" gorm:"type:varchar(50)"`
	PaymentProvider       string  `json:"payment_provider" gorm:"type:varchar(50);default:''"`
	RequestedAmount       float64 `json:"requested_amount"`
	MaxRefundAmount       float64 `json:"max_refund_amount"`
	MaxRefundQuota        int64   `json:"max_refund_quota"`
	Reason                string  `json:"reason" gorm:"type:varchar(255);default:''"`
	AdminReason           string  `json:"admin_reason" gorm:"type:varchar(255);default:''"`
	Status                string  `json:"status" gorm:"type:varchar(32);index"`
	OutRequestNo          string  `json:"out_request_no" gorm:"type:varchar(255);index;default:''"`
	ApproverId            int     `json:"approver_id" gorm:"index;default:0"`
	IsSubscription        bool    `json:"is_subscription" gorm:"default:false"`
	SubscriptionOrderId   int     `json:"subscription_order_id" gorm:"index;default:0"`
	UserSubscriptionId    int     `json:"user_subscription_id" gorm:"index;default:0"`
	SubscriptionUsedRatio float64 `json:"subscription_used_ratio" gorm:"default:0"`
	CreateTime            int64   `json:"create_time"`
	UpdateTime            int64   `json:"update_time"`
}

type OfficialPaymentRefundPreview struct {
	TradeNo                string  `json:"trade_no"`
	PaymentMethod          string  `json:"payment_method"`
	PaymentProvider        string  `json:"payment_provider"`
	Refundable             bool    `json:"refundable"`
	IsSubscription         bool    `json:"is_subscription"`
	RemainingRefundAmount  float64 `json:"remaining_refund_amount"`
	RemainingRefundQuota   int64   `json:"remaining_refund_quota"`
	MaxRefundAmount        float64 `json:"max_refund_amount"`
	MaxRefundQuota         int64   `json:"max_refund_quota"`
	SubscriptionUsedRatio  float64 `json:"subscription_used_ratio,omitempty"`
	SubscriptionTimeRatio  float64 `json:"subscription_time_ratio,omitempty"`
	SubscriptionQuotaRatio float64 `json:"subscription_quota_ratio,omitempty"`
	UserSubscriptionId     int     `json:"user_subscription_id,omitempty"`
	SubscriptionOrderId    int     `json:"subscription_order_id,omitempty"`
	Reason                 string  `json:"reason,omitempty"`
	ExistingPendingRequest int     `json:"existing_pending_request,omitempty"`
}

type OfficialPaymentRefundCreateParams struct {
	TradeNo         string
	PaymentProvider string
	PaymentMethod   string
	RefundAmount    float64
	Reason          string
	OutRequestNo    string
	AllowFullRefund bool
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

func GetTopUpRefundRequestById(id int) *TopUpRefundRequest {
	if id <= 0 {
		return nil
	}
	var request TopUpRefundRequest
	if err := DB.Where("id = ?", id).First(&request).Error; err != nil {
		return nil
	}
	return &request
}

func GetPendingTopUpRefundRequestByTradeNo(tradeNo string) *TopUpRefundRequest {
	tradeNo = strings.TrimSpace(tradeNo)
	if tradeNo == "" {
		return nil
	}
	var request TopUpRefundRequest
	if err := DB.Where("trade_no = ? AND status = ?", tradeNo, TopUpRefundRequestStatusPending).
		Order("id desc").
		First(&request).Error; err != nil {
		return nil
	}
	return &request
}

func CalculateOfficialPaymentRefundPreview(tradeNo string) (*OfficialPaymentRefundPreview, error) {
	tradeNo = strings.TrimSpace(tradeNo)
	if tradeNo == "" {
		return nil, errors.New("未提供订单号")
	}
	var topUp TopUp
	if err := DB.Where("trade_no = ?", tradeNo).First(&topUp).Error; err != nil {
		return nil, ErrTopUpNotFound
	}
	return calculateOfficialPaymentRefundPreviewFromTopUp(DB, topUp)
}

func calculateOfficialPaymentRefundPreviewFromTopUp(tx *gorm.DB, topUp TopUp) (*OfficialPaymentRefundPreview, error) {
	if tx == nil {
		tx = DB
	}
	preview := &OfficialPaymentRefundPreview{
		TradeNo:         topUp.TradeNo,
		PaymentMethod:   topUp.PaymentMethod,
		PaymentProvider: topUp.PaymentProvider,
	}
	if topUp.Status != common.TopUpStatusSuccess && topUp.Status != common.TopUpStatusPartialRefunded {
		preview.Reason = "订单状态不可退款"
		return preview, nil
	}
	if topUp.PaymentProvider != PaymentProviderAlipayOfficial && topUp.PaymentProvider != PaymentProviderWechatPayOfficial {
		preview.Reason = "仅官方支付宝或微信支付订单支持在线退款"
		return preview, nil
	}
	orderMoney := decimal.NewFromFloat(topUp.Money).Round(2)
	refundedMoney := decimal.NewFromFloat(topUp.RefundedMoney).Round(2)
	remainingMoney := orderMoney.Sub(refundedMoney)
	if !remainingMoney.IsPositive() {
		preview.Reason = "订单已无可退金额"
		return preview, nil
	}
	totalQuota := topUpCreditedQuota(topUp)
	remainingQuota := totalQuota - topUp.RefundedQuota
	if remainingQuota < 0 {
		remainingQuota = 0
	}
	preview.RemainingRefundAmount = remainingMoney.InexactFloat64()
	preview.RemainingRefundQuota = remainingQuota

	var order SubscriptionOrder
	orderQuery := tx.Where("trade_no = ?", topUp.TradeNo).First(&order)
	if orderQuery.Error == nil {
		return calculateSubscriptionPaymentRefundPreview(tx, topUp, order, preview, orderMoney, remainingMoney)
	}
	if orderQuery.Error != nil && !errors.Is(orderQuery.Error, gorm.ErrRecordNotFound) {
		return nil, orderQuery.Error
	}
	return calculateBalanceTopUpRefundPreview(tx, topUp, preview, orderMoney, remainingMoney, remainingQuota)
}

func calculateBalanceTopUpRefundPreview(tx *gorm.DB, topUp TopUp, preview *OfficialPaymentRefundPreview, orderMoney decimal.Decimal, remainingMoney decimal.Decimal, remainingQuota int64) (*OfficialPaymentRefundPreview, error) {
	var user User
	if err := tx.Select("id", "quota").Where("id = ?", topUp.UserId).First(&user).Error; err != nil {
		return nil, err
	}
	availableQuota := int64(user.Quota)
	if availableQuota < 0 {
		availableQuota = 0
	}
	maxQuota := remainingQuota
	if maxQuota > availableQuota {
		maxQuota = availableQuota
	}
	preview.MaxRefundQuota = maxQuota
	preview.MaxRefundAmount = refundAmountForQuota(maxQuota, topUpCreditedQuota(topUp), orderMoney, remainingMoney)
	preview.Refundable = preview.MaxRefundQuota > 0 && decimal.NewFromFloat(preview.MaxRefundAmount).Round(2).IsPositive()
	if !preview.Refundable {
		preview.Reason = "用户当前余额不足以扣回该订单未使用部分"
	}
	return preview, nil
}

func calculateSubscriptionPaymentRefundPreview(tx *gorm.DB, topUp TopUp, order SubscriptionOrder, preview *OfficialPaymentRefundPreview, orderMoney decimal.Decimal, remainingMoney decimal.Decimal) (*OfficialPaymentRefundPreview, error) {
	preview.IsSubscription = true
	preview.SubscriptionOrderId = order.Id
	sub, err := findRefundableUserSubscriptionForOrderTx(tx, order)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		preview.Reason = "未找到可退款的订阅实例"
		return preview, nil
	}
	plan, err := getSubscriptionPlanByIdTx(tx, order.PlanId)
	if err != nil {
		return nil, err
	}
	now := common.GetTimestamp()
	timeRatio := subscriptionRefundTimeRatio(*sub, plan, now)
	quotaRatio := subscriptionQuotaUsedRatio(*sub)
	usedRatio := decimal.Max(timeRatio, quotaRatio)
	if usedRatio.LessThan(decimal.Zero) {
		usedRatio = decimal.Zero
	}
	if usedRatio.GreaterThan(decimal.NewFromInt(1)) {
		usedRatio = decimal.NewFromInt(1)
	}
	unusedRatio := decimal.NewFromInt(1).Sub(usedRatio)
	maxMoney := orderMoney.Mul(unusedRatio).RoundFloor(2)
	if maxMoney.GreaterThan(remainingMoney) {
		maxMoney = remainingMoney
	}
	if maxMoney.IsNegative() {
		maxMoney = decimal.Zero
	}
	totalQuota := sub.AmountTotal
	remainingQuota := totalQuota - sub.AmountUsed
	if remainingQuota < 0 {
		remainingQuota = 0
	}
	maxQuota := decimal.NewFromInt(totalQuota).Mul(unusedRatio).RoundFloor(0).IntPart()
	if maxQuota > remainingQuota {
		maxQuota = remainingQuota
	}
	preview.UserSubscriptionId = sub.Id
	preview.SubscriptionUsedRatio = usedRatio.InexactFloat64()
	preview.SubscriptionTimeRatio = timeRatio.InexactFloat64()
	preview.SubscriptionQuotaRatio = quotaRatio.InexactFloat64()
	preview.RemainingRefundQuota = remainingQuota
	preview.MaxRefundAmount = maxMoney.InexactFloat64()
	preview.MaxRefundQuota = maxQuota
	preview.Refundable = maxMoney.IsPositive()
	if !preview.Refundable {
		preview.Reason = "订阅权益已使用完或已无可退金额"
	}
	return preview, nil
}

func findRefundableUserSubscriptionForOrder(order SubscriptionOrder) (*UserSubscription, error) {
	return findRefundableUserSubscriptionForOrderTx(DB, order)
}

func findRefundableUserSubscriptionForOrderTx(tx *gorm.DB, order SubscriptionOrder) (*UserSubscription, error) {
	if tx == nil {
		tx = DB
	}
	var sub UserSubscription
	query := tx.Where("user_id = ? AND plan_id = ? AND source = ? AND created_at >= ?",
		order.UserId,
		order.PlanId,
		"order",
		order.CompleteTime,
	).
		Order("id asc").
		First(&sub)
	if query.Error == nil {
		return &sub, nil
	}
	if !errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return nil, query.Error
	}
	query = tx.Where("user_id = ? AND plan_id = ? AND source = ?", order.UserId, order.PlanId, "order").
		Order("id desc").
		First(&sub)
	if query.Error != nil {
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, query.Error
	}
	return &sub, nil
}

func subscriptionElapsedRatio(sub UserSubscription, now int64) decimal.Decimal {
	if sub.StartTime <= 0 || sub.EndTime <= sub.StartTime {
		return decimal.Zero
	}
	if now <= sub.StartTime {
		return decimal.Zero
	}
	if now >= sub.EndTime {
		return decimal.NewFromInt(1)
	}
	return decimal.NewFromInt(now - sub.StartTime).Div(decimal.NewFromInt(sub.EndTime - sub.StartTime))
}

func subscriptionRefundTimeRatio(sub UserSubscription, plan *SubscriptionPlan, now int64) decimal.Decimal {
	if plan == nil {
		return subscriptionElapsedRatio(sub, now)
	}
	switch NormalizeResetPeriod(plan.QuotaResetPeriod) {
	case SubscriptionResetDaily:
		return subscriptionFixedCycleElapsedRatio(sub, now, int64((24*time.Hour)/time.Second))
	case SubscriptionResetWeekly:
		return subscriptionFixedCycleElapsedRatio(sub, now, int64((7*24*time.Hour)/time.Second))
	case SubscriptionResetCustom:
		return subscriptionFixedCycleElapsedRatio(sub, now, plan.QuotaResetCustomSeconds)
	case SubscriptionResetMonthly:
		return subscriptionMonthlyCycleElapsedRatio(sub, now)
	default:
		return subscriptionElapsedRatio(sub, now)
	}
}

func subscriptionFixedCycleElapsedRatio(sub UserSubscription, now int64, cycleSeconds int64) decimal.Decimal {
	if sub.StartTime <= 0 || sub.EndTime <= sub.StartTime || cycleSeconds <= 0 {
		return subscriptionElapsedRatio(sub, now)
	}
	if now <= sub.StartTime {
		return decimal.Zero
	}
	if now >= sub.EndTime {
		return decimal.NewFromInt(1)
	}
	durationSeconds := sub.EndTime - sub.StartTime
	totalCycles := ceilDivideInt64(durationSeconds, cycleSeconds)
	if totalCycles <= 0 {
		return subscriptionElapsedRatio(sub, now)
	}
	usedCycles := ceilDivideInt64(now-sub.StartTime, cycleSeconds)
	if usedCycles < 0 {
		usedCycles = 0
	}
	if usedCycles > totalCycles {
		usedCycles = totalCycles
	}
	return decimal.NewFromInt(usedCycles).Div(decimal.NewFromInt(totalCycles))
}

func subscriptionMonthlyCycleElapsedRatio(sub UserSubscription, now int64) decimal.Decimal {
	if sub.StartTime <= 0 || sub.EndTime <= sub.StartTime {
		return decimal.Zero
	}
	if now <= sub.StartTime {
		return decimal.Zero
	}
	if now >= sub.EndTime {
		return decimal.NewFromInt(1)
	}
	start := time.Unix(sub.StartTime, 0)
	end := time.Unix(sub.EndTime, 0)
	current := time.Unix(now, 0)
	totalCycles := countSubscriptionCalendarCycles(start, end, func(t time.Time) time.Time {
		return t.AddDate(0, 1, 0)
	})
	usedCycles := countSubscriptionCalendarCycles(start, current, func(t time.Time) time.Time {
		return t.AddDate(0, 1, 0)
	})
	if totalCycles <= 0 {
		return subscriptionElapsedRatio(sub, now)
	}
	if usedCycles > totalCycles {
		usedCycles = totalCycles
	}
	return decimal.NewFromInt(usedCycles).Div(decimal.NewFromInt(totalCycles))
}

func countSubscriptionCalendarCycles(start time.Time, end time.Time, advance func(time.Time) time.Time) int64 {
	if !end.After(start) {
		return 0
	}
	var count int64
	cursor := start
	for cursor.Before(end) {
		count++
		next := advance(cursor)
		if !next.After(cursor) {
			break
		}
		cursor = next
	}
	return count
}

func ceilDivideInt64(value int64, divisor int64) int64 {
	if value <= 0 || divisor <= 0 {
		return 0
	}
	return (value + divisor - 1) / divisor
}

func subscriptionQuotaUsedRatio(sub UserSubscription) decimal.Decimal {
	if sub.AmountTotal <= 0 {
		return decimal.Zero
	}
	if sub.AmountUsed <= 0 {
		return decimal.Zero
	}
	if sub.AmountUsed >= sub.AmountTotal {
		return decimal.NewFromInt(1)
	}
	return decimal.NewFromInt(sub.AmountUsed).Div(decimal.NewFromInt(sub.AmountTotal))
}

func refundAmountForQuota(refundQuota int64, totalQuota int64, orderMoney decimal.Decimal, remainingMoney decimal.Decimal) float64 {
	if refundQuota <= 0 || totalQuota <= 0 || !orderMoney.IsPositive() {
		return 0
	}
	amount := orderMoney.Mul(decimal.NewFromInt(refundQuota)).Div(decimal.NewFromInt(totalQuota)).RoundFloor(2)
	if amount.GreaterThan(remainingMoney) {
		amount = remainingMoney
	}
	if amount.IsNegative() {
		return 0
	}
	return amount.InexactFloat64()
}

func CreateTopUpRefundRequest(userId int, tradeNo string, requestedAmount float64, reason string) (*TopUpRefundRequest, *OfficialPaymentRefundPreview, error) {
	if userId <= 0 {
		return nil, nil, errors.New("无效用户")
	}
	tradeNo = strings.TrimSpace(tradeNo)
	if tradeNo == "" {
		return nil, nil, errors.New("未提供订单号")
	}
	requestAmount := decimal.NewFromFloat(requestedAmount).Round(2)
	if !requestAmount.IsPositive() {
		return nil, nil, errors.New("退款金额必须大于 0")
	}
	reason = normalizeRefundRequestReason(reason)
	if reason == "" {
		return nil, nil, errors.New("请填写退款原因")
	}
	var request *TopUpRefundRequest
	var preview *OfficialPaymentRefundPreview
	err := DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		if err := withRowLock(tx).Where("trade_no = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if topUp.UserId != userId {
			return ErrPaymentMethodMismatch
		}
		var pendingCount int64
		if err := tx.Model(&TopUpRefundRequest{}).Where("trade_no = ? AND status = ?", tradeNo, TopUpRefundRequestStatusPending).Count(&pendingCount).Error; err != nil {
			return err
		}
		if pendingCount > 0 {
			return errors.New("该订单已有待处理退款申请")
		}
		calculated, err := calculateOfficialPaymentRefundPreviewFromTopUp(tx, *topUp)
		if err != nil {
			return err
		}
		preview = calculated
		if !preview.Refundable {
			if preview.Reason != "" {
				return errors.New(preview.Reason)
			}
			return errors.New("该订单当前不可退款")
		}
		maxAmount := decimal.NewFromFloat(preview.MaxRefundAmount).Round(2)
		if requestAmount.GreaterThan(maxAmount) {
			return errors.New("退款金额超过当前可退金额")
		}
		now := common.GetTimestamp()
		request = &TopUpRefundRequest{
			TopUpId:               topUp.Id,
			UserId:                topUp.UserId,
			TradeNo:               topUp.TradeNo,
			PaymentMethod:         topUp.PaymentMethod,
			PaymentProvider:       topUp.PaymentProvider,
			RequestedAmount:       requestAmount.InexactFloat64(),
			MaxRefundAmount:       preview.MaxRefundAmount,
			MaxRefundQuota:        preview.MaxRefundQuota,
			Reason:                reason,
			Status:                TopUpRefundRequestStatusPending,
			IsSubscription:        preview.IsSubscription,
			SubscriptionOrderId:   preview.SubscriptionOrderId,
			UserSubscriptionId:    preview.UserSubscriptionId,
			SubscriptionUsedRatio: preview.SubscriptionUsedRatio,
			CreateTime:            now,
			UpdateTime:            now,
		}
		return tx.Create(request).Error
	})
	if err != nil {
		return nil, preview, err
	}
	return request, preview, nil
}

func normalizeRefundRequestReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ""
	}
	runes := []rune(reason)
	if len(runes) <= 255 {
		return reason
	}
	return string(runes[:255])
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

func ExpireOfficialPaymentPendingTopUpsBefore(ctx context.Context, paymentProvider string, createBefore int64, completeTime int64, userId int) (int64, error) {
	if !IsOfficialPaymentProvider(paymentProvider) {
		return 0, ErrPaymentMethodMismatch
	}
	if createBefore <= 0 {
		return 0, nil
	}
	if completeTime <= 0 {
		completeTime = common.GetTimestamp()
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var expiredTopUps int64
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		subscriptionQuery := tx.Model(&SubscriptionOrder{}).
			Where("payment_provider = ? AND status = ? AND create_time <= ?", paymentProvider, common.TopUpStatusPending, createBefore)
		if userId > 0 {
			subscriptionQuery = subscriptionQuery.Where("user_id = ?", userId)
		}
		if err := subscriptionQuery.Updates(map[string]interface{}{
			"status":        common.TopUpStatusExpired,
			"complete_time": completeTime,
		}).Error; err != nil {
			return err
		}
		query := tx.Model(&TopUp{}).
			Where("payment_provider = ? AND status = ? AND create_time <= ?", paymentProvider, common.TopUpStatusPending, createBefore)
		if userId > 0 {
			query = query.Where("user_id = ?", userId)
		}
		result := query.Updates(map[string]interface{}{
			"status":        common.TopUpStatusExpired,
			"complete_time": completeTime,
		})
		if result.Error != nil {
			return result.Error
		}
		expiredTopUps = result.RowsAffected
		return nil
	})
	return expiredTopUps, err
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
		if err := withRowLock(tx).Where(refCol+" = ?", params.TradeNo).First(topUp).Error; err != nil {
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
		var subscriptionOrder *SubscriptionOrder
		var lockedSubscriptionOrder SubscriptionOrder
		subscriptionQuery := tx.Where("trade_no = ?", topUp.TradeNo).First(&lockedSubscriptionOrder)
		if subscriptionQuery.Error == nil {
			subscriptionOrder = &lockedSubscriptionOrder
		} else if !errors.Is(subscriptionQuery.Error, gorm.ErrRecordNotFound) {
			return subscriptionQuery.Error
		}
		if !params.AllowFullRefund {
			calculatedPreview, err := calculateOfficialPaymentRefundPreviewFromTopUp(tx, *topUp)
			if err != nil {
				return err
			}
			preview := calculatedPreview
			if !preview.Refundable {
				if preview.Reason != "" {
					return errors.New(preview.Reason)
				}
				return errors.New("该订单当前不可退款")
			}
			if refundAmount.GreaterThan(decimal.NewFromFloat(preview.MaxRefundAmount).Round(2)) {
				return errors.New("退款金额超过当前可退金额")
			}
		}

		orderMoney := decimal.NewFromFloat(topUp.Money).Round(2)
		refundedMoney := decimal.NewFromFloat(topUp.RefundedMoney).Round(2)
		remainingMoney := orderMoney.Sub(refundedMoney)
		if remainingMoney.LessThan(refundAmount) {
			return errors.New("退款金额超过可退金额")
		}

		totalQuota := topUpCreditedQuota(*topUp)
		refundQuota := calculateOfficialRefundQuota(totalQuota, topUp.RefundedQuota, orderMoney, refundAmount, remainingMoney)
		if subscriptionOrder != nil {
			refundQuota = 0
		}
		if refundQuota <= 0 && subscriptionOrder == nil {
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
		if subscriptionOrder == nil {
			if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota - ?", refundQuota)).Error; err != nil {
				return err
			}
			if err := reverseTopUpReferralCommissionForRefundTx(tx, topUp.Id, refundAmount, orderMoney); err != nil {
				return err
			}
		}
		if topUp.Status == common.TopUpStatusRefunded && subscriptionOrder == nil {
			return reverseInviterRewardForFullRefundTx(tx, topUp.UserId)
		}
		return nil
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
		if err := withRowLock(tx).Where("out_request_no = ?", outRequestNo).First(refund).Error; err != nil {
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
		if err := withRowLock(tx).Where("out_request_no = ?", outRequestNo).First(refund).Error; err != nil {
			return err
		}
		if refund.Status == TopUpRefundStatusFailed {
			return nil
		}
		if refund.Status != TopUpRefundStatusPending && refund.Status != TopUpRefundStatusSuccess {
			return ErrTopUpStatusInvalid
		}
		topUp := &TopUp{}
		if err := withRowLock(tx).Where("id = ?", refund.TopUpId).First(topUp).Error; err != nil {
			return err
		}
		var subscriptionOrder *SubscriptionOrder
		var lockedSubscriptionOrder SubscriptionOrder
		subscriptionQuery := tx.Where("trade_no = ?", topUp.TradeNo).First(&lockedSubscriptionOrder)
		if subscriptionQuery.Error == nil {
			subscriptionOrder = &lockedSubscriptionOrder
		} else if !errors.Is(subscriptionQuery.Error, gorm.ErrRecordNotFound) {
			return subscriptionQuery.Error
		}
		if subscriptionOrder == nil {
			if err := reverseTopUpReferralCommissionForRefundTx(tx, refund.TopUpId, decimal.NewFromFloat(refund.RefundAmount).Neg(), decimal.NewFromFloat(topUp.Money).Round(2)); err != nil {
				return err
			}
		}
		wasFullyRefunded := topUp.Status == common.TopUpStatusRefunded
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
		if wasFullyRefunded && topUp.Status != common.TopUpStatusRefunded {
			if err := restoreInviterRewardForRefundFailureTx(tx, topUp.UserId); err != nil {
				return err
			}
		}
		if subscriptionOrder != nil {
			return nil
		}
		return tx.Model(&User{}).Where("id = ?", refund.UserId).Update("quota", gorm.Expr("quota + ?", refund.RefundQuota)).Error
	})
}

func MarkTopUpRefundRequestApproved(requestId int, approverId int, outRequestNo string, adminReason string) error {
	if requestId <= 0 {
		return errors.New("无效退款申请")
	}
	result := DB.Model(&TopUpRefundRequest{}).
		Where("id = ? AND status = ?", requestId, TopUpRefundRequestStatusPending).
		Updates(map[string]interface{}{
			"status":         TopUpRefundRequestStatusApproved,
			"approver_id":    approverId,
			"out_request_no": strings.TrimSpace(outRequestNo),
			"admin_reason":   strings.TrimSpace(adminReason),
			"update_time":    common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTopUpStatusInvalid
	}
	return nil
}

func MarkTopUpRefundRequestRejected(requestId int, approverId int, adminReason string) error {
	if requestId <= 0 {
		return errors.New("无效退款申请")
	}
	result := DB.Model(&TopUpRefundRequest{}).
		Where("id = ? AND status = ?", requestId, TopUpRefundRequestStatusPending).
		Updates(map[string]interface{}{
			"status":       TopUpRefundRequestStatusRejected,
			"approver_id":  approverId,
			"admin_reason": strings.TrimSpace(adminReason),
			"update_time":  common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTopUpStatusInvalid
	}
	return nil
}

func MarkTopUpRefundRequestFailed(requestId int, approverId int, adminReason string) error {
	if requestId <= 0 {
		return errors.New("无效退款申请")
	}
	result := DB.Model(&TopUpRefundRequest{}).
		Where("id = ? AND status = ?", requestId, TopUpRefundRequestStatusPending).
		Updates(map[string]interface{}{
			"status":       TopUpRefundRequestStatusFailed,
			"approver_id":  approverId,
			"admin_reason": strings.TrimSpace(adminReason),
			"update_time":  common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTopUpStatusInvalid
	}
	return nil
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
	return reverseReferralCommissionForRefundTx(tx, ReferralCommissionSourceTopUp, topUpId, refundAmount, orderMoney)
}

func reverseReferralCommissionForRefundTx(tx *gorm.DB, sourceType string, sourceId int, refundAmount decimal.Decimal, orderMoney decimal.Decimal) error {
	if tx == nil || sourceId <= 0 || refundAmount.IsZero() || !orderMoney.IsPositive() {
		return nil
	}
	var commissions []ReferralCommission
	if err := tx.Where("source_type = ? AND source_id = ?", normalizeReferralCommissionSource(sourceType), sourceId).Find(&commissions).Error; err != nil {
		return err
	}
	for _, commission := range commissions {
		absoluteRefundAmount := refundAmount.Abs()
		currentRefundedAmount := decimal.NewFromFloat(commission.RefundedRechargeAmount).Round(2)
		if currentRefundedAmount.IsNegative() {
			currentRefundedAmount = decimal.Zero
		}

		nextRefundedAmount := currentRefundedAmount.Add(absoluteRefundAmount)
		if refundAmount.IsNegative() {
			nextRefundedAmount = currentRefundedAmount.Sub(absoluteRefundAmount)
		}
		if nextRefundedAmount.IsNegative() {
			nextRefundedAmount = decimal.Zero
		}
		if nextRefundedAmount.GreaterThan(orderMoney) {
			nextRefundedAmount = orderMoney
		}

		if err := applyReferralCommissionRefundTargetTx(tx, commission, nextRefundedAmount, orderMoney); err != nil {
			return err
		}
	}
	return nil
}

func setReferralCommissionRefundTargetTx(tx *gorm.DB, sourceType string, sourceId int, refundedAmount decimal.Decimal, orderMoney decimal.Decimal) error {
	if tx == nil || sourceId <= 0 || !orderMoney.IsPositive() {
		return nil
	}
	if refundedAmount.IsNegative() {
		refundedAmount = decimal.Zero
	}
	if refundedAmount.GreaterThan(orderMoney) {
		refundedAmount = orderMoney
	}
	var commissions []ReferralCommission
	if err := tx.Where("source_type = ? AND source_id = ?", normalizeReferralCommissionSource(sourceType), sourceId).Find(&commissions).Error; err != nil {
		return err
	}
	for _, commission := range commissions {
		if err := applyReferralCommissionRefundTargetTx(tx, commission, refundedAmount, orderMoney); err != nil {
			return err
		}
	}
	return nil
}

func applyReferralCommissionRefundTargetTx(tx *gorm.DB, commission ReferralCommission, targetRefundedAmount decimal.Decimal, orderMoney decimal.Decimal) error {
	if tx == nil || !orderMoney.IsPositive() {
		return nil
	}
	if targetRefundedAmount.IsNegative() {
		targetRefundedAmount = decimal.Zero
	}
	if targetRefundedAmount.GreaterThan(orderMoney) {
		targetRefundedAmount = orderMoney
	}
	currentRefundedQuota := commission.RefundedCommissionQuota
	if currentRefundedQuota < 0 {
		currentRefundedQuota = 0
	}
	if currentRefundedQuota > commission.CommissionQuota {
		currentRefundedQuota = commission.CommissionQuota
	}
	targetRefundedQuota := referralCommissionRefundQuotaTarget(commission.CommissionQuota, targetRefundedAmount, orderMoney)
	quotaDelta := int64(targetRefundedQuota - currentRefundedQuota)

	commissionAmount := decimal.NewFromFloat(commission.RechargeAmount).Round(2)
	recordedRefundedAmount := targetRefundedAmount
	if commissionAmount.IsPositive() && recordedRefundedAmount.GreaterThan(commissionAmount) {
		recordedRefundedAmount = commissionAmount
	}

	if quotaDelta != 0 && commission.InviterId > 0 {
		affExpr := gorm.Expr("aff_quota - ?", quotaDelta)
		historyExpr := gorm.Expr("aff_history - ?", quotaDelta)
		if quotaDelta < 0 {
			restoredQuota := -quotaDelta
			affExpr = gorm.Expr("aff_quota + ?", restoredQuota)
			historyExpr = gorm.Expr("aff_history + ?", restoredQuota)
		}
		if err := tx.Model(&User{}).
			Where("id = ?", commission.InviterId).
			Updates(map[string]interface{}{
				"aff_quota":   affExpr,
				"aff_history": historyExpr,
			}).Error; err != nil {
			return err
		}
	}
	return tx.Model(&ReferralCommission{}).
		Where("id = ?", commission.Id).
		Updates(map[string]interface{}{
			"refunded_commission_quota": targetRefundedQuota,
			"refunded_recharge_amount":  recordedRefundedAmount.InexactFloat64(),
		}).Error
}

func referralCommissionRefundQuotaTarget(commissionQuota int, refundedAmount decimal.Decimal, orderMoney decimal.Decimal) int {
	if commissionQuota <= 0 || !refundedAmount.IsPositive() || !orderMoney.IsPositive() {
		return 0
	}
	if !refundedAmount.LessThan(orderMoney) {
		return commissionQuota
	}
	target := decimal.NewFromInt(int64(commissionQuota)).
		Mul(refundedAmount).
		Div(orderMoney).
		Round(0).
		IntPart()
	if target < 0 {
		return 0
	}
	if target > int64(commissionQuota) {
		return commissionQuota
	}
	return int(target)
}

func reverseInviterRewardForFullRefundTx(tx *gorm.DB, userId int) error {
	if tx == nil || userId <= 0 {
		return nil
	}
	var invitee User
	if err := withRowLock(tx).
		Select("id", "inviter_id", "inviter_reward_quota", "inviter_reward_unlocked", "inviter_reward_unlocked_by_payment").
		Where("id = ?", userId).
		First(&invitee).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if invitee.InviterId <= 0 || invitee.InviterId == invitee.Id ||
		!invitee.InviterRewardUnlocked || !invitee.InviterRewardUnlockedByPayment ||
		invitee.InviterRewardQuota <= 0 {
		return nil
	}
	hasPayment, err := inviteeHasEffectivePaymentTx(tx, userId)
	if err != nil {
		return err
	}
	if hasPayment {
		return nil
	}
	if err := tx.Model(&User{}).
		Where("id = ?", invitee.InviterId).
		Updates(map[string]interface{}{
			"aff_quota":   gorm.Expr("aff_quota - ?", invitee.InviterRewardQuota),
			"aff_history": gorm.Expr("aff_history - ?", invitee.InviterRewardQuota),
		}).Error; err != nil {
		return err
	}
	return tx.Model(&User{}).
		Where("id = ? AND inviter_reward_unlocked = ?", invitee.Id, true).
		Updates(map[string]interface{}{
			"inviter_reward_unlocked":            false,
			"inviter_reward_unlocked_by_payment": false,
		}).Error
}

func restoreInviterRewardForRefundFailureTx(tx *gorm.DB, userId int) error {
	if tx == nil || userId <= 0 {
		return nil
	}
	hasPayment, err := inviteeHasEffectivePaymentTx(tx, userId)
	if err != nil {
		return err
	}
	if !hasPayment {
		return nil
	}
	_, _, _, _, err = creditInviterRewardTx(tx, userId)
	return err
}

func inviteeHasEffectivePaymentTx(tx *gorm.DB, userId int) (bool, error) {
	if tx == nil || userId <= 0 {
		return false, nil
	}
	var topUpCount int64
	if err := tx.Model(&TopUp{}).
		Where("user_id = ? AND status IN ? AND complete_time > ?", userId, []string{
			common.TopUpStatusSuccess,
			common.TopUpStatusPartialRefunded,
		}, 0).
		Count(&topUpCount).Error; err != nil {
		return false, err
	}
	var subscriptionCount int64
	if err := tx.Model(&SubscriptionOrder{}).
		Where("user_id = ? AND status = ? AND complete_time > ?", userId, common.TopUpStatusSuccess, 0).
		Count(&subscriptionCount).Error; err != nil {
		return false, err
	}
	return topUpCount+subscriptionCount > 0, nil
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
		if err := withRowLock(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
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
		if err := withRowLock(tx).Where(refCol+" = ?", tradeNo).First(&topUp).Error; err != nil {
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
		err := withRowLock(tx).Where(refCol+" = ?", referenceId).First(topUp).Error
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

func fillTopUpPendingRefundRequests(topUps []*TopUp) {
	if len(topUps) == 0 {
		return
	}
	tradeNos := make([]string, 0, len(topUps))
	for _, topUp := range topUps {
		if topUp != nil && strings.TrimSpace(topUp.TradeNo) != "" {
			tradeNos = append(tradeNos, topUp.TradeNo)
		}
	}
	if len(tradeNos) == 0 {
		return
	}
	var requests []TopUpRefundRequest
	if err := DB.Select("id", "trade_no", "status", "requested_amount", "reason").
		Where("trade_no IN ? AND status = ?", tradeNos, TopUpRefundRequestStatusPending).
		Order("id desc").
		Find(&requests).Error; err != nil {
		return
	}
	requestByTradeNo := make(map[string]TopUpRefundRequest, len(requests))
	for _, request := range requests {
		if _, exists := requestByTradeNo[request.TradeNo]; !exists {
			requestByTradeNo[request.TradeNo] = request
		}
	}
	for _, topUp := range topUps {
		if topUp == nil {
			continue
		}
		if request, ok := requestByTradeNo[topUp.TradeNo]; ok {
			topUp.RefundRequestId = request.Id
			topUp.RefundRequestStatus = request.Status
			topUp.RefundRequestAmount = request.RequestedAmount
			topUp.RefundRequestReason = request.Reason
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

func findUserIDsByUsernamePattern(db *gorm.DB, pattern string) ([]int, error) {
	var users []User
	if err := db.Select("id").Where("username LIKE ? ESCAPE '!'", pattern).Limit(searchTopUpCountHardLimit).Find(&users).Error; err != nil {
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

type TopUpQueryOptions struct {
	Keyword       string
	UserID        int
	Username      string
	PaymentMethod string
	TradeNo       string
	StartTime     int64
	EndTime       int64
	PendingRefund bool
}

type TopUpQueryResult struct {
	Items      []*TopUp `json:"items"`
	Total      int64    `json:"total"`
	TotalMoney float64  `json:"total_money"`
}

func applyPendingTopUpRefundFilter(db *gorm.DB, query *gorm.DB) *gorm.DB {
	pendingRefundTradeNos := db.Model(&TopUpRefundRequest{}).
		Select("trade_no").
		Where("status = ?", TopUpRefundRequestStatusPending)
	return query.Where("trade_no IN (?)", pendingRefundTradeNos)
}

func applyTopUpExactFilters(query *gorm.DB, options TopUpQueryOptions) (*gorm.DB, error) {
	if options.TradeNo != "" {
		pattern, err := buildTopUpContainsLikePattern(options.TradeNo)
		if err != nil {
			return nil, err
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", pattern)
	}
	if options.PaymentMethod != "" {
		query = query.Where("(payment_method = ? OR payment_provider = ?)", options.PaymentMethod, options.PaymentMethod)
	}
	if options.StartTime > 0 {
		query = query.Where("create_time >= ?", options.StartTime)
	}
	if options.EndTime > 0 {
		query = query.Where("create_time <= ?", options.EndTime)
	}
	return query, nil
}

func applyUserTopUpQueryOptions(db *gorm.DB, query *gorm.DB, userId int, options TopUpQueryOptions) (*gorm.DB, error) {
	query = query.Where("user_id = ? AND create_time >= ?", userId, topUpQueryCutoff())
	if options.Keyword != "" {
		pattern, err := buildTopUpContainsLikePattern(options.Keyword)
		if err != nil {
			return nil, err
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", pattern)
	}
	var err error
	query, err = applyTopUpExactFilters(query, options)
	if err != nil {
		return nil, err
	}
	if options.PendingRefund {
		query = applyPendingTopUpRefundFilter(db, query)
	}
	return query, nil
}

func applyAllTopUpQueryOptions(db *gorm.DB, query *gorm.DB, options TopUpQueryOptions) (*gorm.DB, error) {
	if options.UserID > 0 {
		query = query.Where("user_id = ?", options.UserID)
	}
	if options.Username != "" {
		pattern, err := buildTopUpContainsLikePattern(options.Username)
		if err != nil {
			return nil, err
		}
		matchedUserIds, matchErr := findUserIDsByUsernamePattern(db, pattern)
		if matchErr != nil {
			return nil, matchErr
		}
		query = query.Where("user_id IN ?", uniqueIntSlice(matchedUserIds))
	}
	if options.Keyword != "" {
		pattern, err := buildTopUpContainsLikePattern(options.Keyword)
		if err != nil {
			return nil, err
		}
		userIdKeyword, hasUserIdKeyword := parseTopUpUserIDKeyword(options.Keyword)
		matchedUserIds, matchErr := findUserIDsByUsernamePattern(db, pattern)
		if matchErr != nil {
			return nil, matchErr
		}
		if hasUserIdKeyword {
			matchedUserIds = append(matchedUserIds, userIdKeyword)
		}
		query = query.Where("(trade_no LIKE ? ESCAPE '!' OR user_id IN ?)", pattern, uniqueIntSlice(matchedUserIds))
	}
	var err error
	query, err = applyTopUpExactFilters(query, options)
	if err != nil {
		return nil, err
	}
	if options.PendingRefund {
		query = applyPendingTopUpRefundFilter(db, query)
	}
	return query, nil
}

func GetUserTopUps(userId int, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	return GetUserTopUpsWithOptions(userId, TopUpQueryOptions{}, pageInfo)
}

// topUpReadTxOptions 返回账单列表 COUNT/SUM/SELECT 所用的只读事务选项。
// 使用 ReadOnly 让驱动可以走只读优化；显式 RepeatableRead 让 PostgreSQL
// 也能在同一请求内获得稳定快照（MySQL InnoDB 默认即为 RepeatableRead）；
// SQLite 驱动只支持 Serializable，故走默认隔离级别（WAL 模式下事务即快照）。
func topUpReadTxOptions() *sql.TxOptions {
	if common.UsingSQLite {
		return &sql.TxOptions{ReadOnly: true}
	}
	return &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead}
}

func scanTopUpTotalMoney(query *gorm.DB, totalMoney *float64) error {
	var total sql.NullFloat64
	if err := query.Session(&gorm.Session{}).
		Where("status = ?", common.TopUpStatusSuccess).
		Select("COALESCE(SUM(money), 0)").
		Scan(&total).Error; err != nil {
		return err
	}
	if total.Valid {
		*totalMoney = total.Float64
	} else {
		*totalMoney = 0
	}
	return nil
}

func shouldLimitTopUpCount(options TopUpQueryOptions) bool {
	return options.Keyword != "" || options.Username != "" || options.TradeNo != ""
}

func GetUserTopUpsResultWithOptions(userId int, options TopUpQueryOptions, pageInfo *common.PageInfo) (result TopUpQueryResult, err error) {
	err = DB.Transaction(func(tx *gorm.DB) error {
		query, qErr := applyUserTopUpQueryOptions(tx, tx.Model(&TopUp{}), userId, options)
		if qErr != nil {
			return qErr
		}

		countQuery := query
		if shouldLimitTopUpCount(options) {
			countQuery = countQuery.Limit(searchTopUpCountHardLimit)
		}
		if cErr := countQuery.Count(&result.Total).Error; cErr != nil {
			return cErr
		}
		if sErr := scanTopUpTotalMoney(query, &result.TotalMoney); sErr != nil {
			return sErr
		}

		return query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&result.Items).Error
	}, topUpReadTxOptions())
	if err != nil {
		return TopUpQueryResult{}, err
	}

	// /topup/self 视图不展示用户名列，无需回填，省一次额外查询
	fillTopUpPendingRefundRequests(result.Items)

	return result, nil
}

func GetUserTopUpsWithOptions(userId int, options TopUpQueryOptions, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	result, err := GetUserTopUpsResultWithOptions(userId, options, pageInfo)
	if err != nil {
		return nil, 0, err
	}
	return result.Items, result.Total, nil
}

// GetAllTopUps 获取全平台的充值记录（管理员使用，不限制时间窗口）
func GetAllTopUps(pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	return GetAllTopUpsWithOptions(TopUpQueryOptions{}, pageInfo)
}

func GetAllTopUpsWithOptions(options TopUpQueryOptions, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	result, err := GetAllTopUpsResultWithOptions(options, pageInfo)
	if err != nil {
		return nil, 0, err
	}
	return result.Items, result.Total, nil
}

func GetAllTopUpsResultWithOptions(options TopUpQueryOptions, pageInfo *common.PageInfo) (result TopUpQueryResult, err error) {
	err = DB.Transaction(func(tx *gorm.DB) error {
		query, qErr := applyAllTopUpQueryOptions(tx, tx.Model(&TopUp{}), options)
		if qErr != nil {
			return qErr
		}

		countQuery := query
		if shouldLimitTopUpCount(options) {
			countQuery = countQuery.Limit(searchTopUpCountHardLimit)
		}
		if cErr := countQuery.Count(&result.Total).Error; cErr != nil {
			return cErr
		}
		if sErr := scanTopUpTotalMoney(query, &result.TotalMoney); sErr != nil {
			return sErr
		}

		return query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&result.Items).Error
	}, topUpReadTxOptions())
	if err != nil {
		return TopUpQueryResult{}, err
	}

	fillTopUpUsernames(result.Items)
	fillTopUpPendingRefundRequests(result.Items)

	return result, nil
}

// searchTopUpCountHardLimit 搜索充值记录时 COUNT 的安全上限，
// 防止对超大表执行无界 COUNT 触发 DoS。
const searchTopUpCountHardLimit = 10000

// SearchUserTopUps 按订单号搜索某用户的充值记录
func SearchUserTopUps(userId int, keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	return GetUserTopUpsWithOptions(userId, TopUpQueryOptions{Keyword: keyword}, pageInfo)
}

// SearchAllTopUps 按用户 ID、用户名或订单号搜索全平台充值记录（管理员使用，不限制时间窗口）
func SearchAllTopUps(keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	return GetAllTopUpsWithOptions(TopUpQueryOptions{Keyword: keyword}, pageInfo)
}

func IsSubscriptionTopUpRecord(topUp *TopUp) bool {
	if topUp == nil || topUp.Amount != 0 {
		return false
	}
	tradeNo := strings.ToUpper(strings.TrimSpace(topUp.TradeNo))
	return strings.HasPrefix(tradeNo, "SUB") ||
		strings.HasPrefix(tradeNo, "ALIPAYSUB") ||
		strings.HasPrefix(tradeNo, "WXSUB")
}

func adminTopUpTradeNo() string {
	return fmt.Sprintf("ADMIN_%d_%s", common.GetTimestamp(), strings.ToUpper(common.GetUUID()[:8]))
}

func adminTopUpRefundNo() string {
	return fmt.Sprintf("ADMIN_RF_%d_%s", common.GetTimestamp(), strings.ToUpper(common.GetUUID()[:8]))
}

func CreateAdminBalanceTopUp(userId int, quota int) (*TopUp, error) {
	if userId <= 0 {
		return nil, errors.New("无效用户")
	}
	if quota <= 0 {
		return nil, errors.New("无效的充值额度")
	}

	now := common.GetTimestamp()
	topUp := &TopUp{
		UserId:          userId,
		Amount:          int64(quota),
		Money:           0,
		TradeNo:         adminTopUpTradeNo(),
		PaymentMethod:   PaymentMethodAdminAdd,
		PaymentProvider: PaymentProviderAdmin,
		CreateTime:      now,
		CompleteTime:    now,
		Status:          common.TopUpStatusSuccess,
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := withRowLock(tx).Select("id").Where("id = ?", userId).First(&user).Error; err != nil {
			return err
		}
		if err := tx.Create(topUp).Error; err != nil {
			return err
		}
		return tx.Model(&User{}).Where("id = ?", userId).Update("quota", gorm.Expr("quota + ?", quota)).Error
	})
	if err != nil {
		return nil, err
	}
	return topUp, nil
}

func RefundAdminBalanceTopUp(tradeNo string, refundQuota int64, reason string) (*TopUpRefund, error) {
	tradeNo = strings.TrimSpace(tradeNo)
	if tradeNo == "" {
		return nil, errors.New("未提供订单号")
	}
	if refundQuota <= 0 {
		return nil, errors.New("退款额度必须大于 0")
	}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	var refund *TopUpRefund
	err := DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		if err := withRowLock(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if !IsAdminTopUpRecord(topUp) {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status != common.TopUpStatusSuccess && topUp.Status != common.TopUpStatusPartialRefunded {
			return ErrTopUpStatusInvalid
		}

		totalQuota := topUpCreditedQuota(*topUp)
		remainingQuota := totalQuota - topUp.RefundedQuota
		if remainingQuota <= 0 {
			return errors.New("订单已无可退额度")
		}
		if refundQuota > remainingQuota {
			return errors.New("退款额度超过可退额度")
		}

		result := tx.Model(&User{}).
			Where("id = ? AND quota >= ?", topUp.UserId, refundQuota).
			Update("quota", gorm.Expr("quota - ?", refundQuota))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("用户当前余额不足以扣回退款额度")
		}

		now := common.GetTimestamp()
		refund = &TopUpRefund{
			TopUpId:         topUp.Id,
			UserId:          topUp.UserId,
			TradeNo:         topUp.TradeNo,
			OutRequestNo:    adminTopUpRefundNo(),
			PaymentMethod:   topUp.PaymentMethod,
			PaymentProvider: topUp.PaymentProvider,
			RefundAmount:    0,
			RefundQuota:     refundQuota,
			Reason:          normalizeRefundRequestReason(reason),
			Status:          TopUpRefundStatusSuccess,
			CreateTime:      now,
			CompleteTime:    now,
		}
		if err := tx.Create(refund).Error; err != nil {
			return err
		}

		topUp.RefundedQuota += refundQuota
		if topUp.RefundedQuota >= totalQuota {
			topUp.Status = common.TopUpStatusRefunded
		} else {
			topUp.Status = common.TopUpStatusPartialRefunded
		}
		return tx.Save(topUp).Error
	})
	if err != nil {
		return nil, err
	}
	return refund, nil
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
		if err := withRowLock(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return errors.New("充值订单不存在")
		}

		// 幂等处理：已成功直接返回
		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}

		if topUp.Status != common.TopUpStatusPending &&
			!(topUp.Status == common.TopUpStatusExpired && IsOfficialPaymentProvider(topUp.PaymentProvider)) {
			return errors.New("订单状态不是待支付或官方支付已超时，无法补单")
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

// ManualCompleteOfficialSubscriptionTopUp 管理员补齐官方订阅订单。
func ManualCompleteOfficialSubscriptionTopUp(tradeNo string) error {
	if tradeNo == "" {
		return errors.New("未提供订单号")
	}
	order := GetSubscriptionOrderByTradeNo(tradeNo)
	if order == nil {
		return ErrSubscriptionOrderNotFound
	}
	if !IsOfficialPaymentProvider(order.PaymentProvider) {
		return ErrPaymentMethodMismatch
	}
	return CompleteSubscriptionOrder(tradeNo, `{"source":"admin"}`, order.PaymentProvider, order.PaymentMethod)
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
		err := withRowLock(tx).Where(refCol+" = ?", referenceId).First(topUp).Error
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
		err := withRowLock(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error
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
		err := withRowLock(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error
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
		err := withRowLock(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error
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

		if topUp.Status != common.TopUpStatusPending &&
			!(topUp.Status == common.TopUpStatusExpired && IsOfficialPaymentProvider(topUp.PaymentProvider)) {
			return errors.New("充值订单状态错误")
		}

		if len(paidMoney) > 0 {
			expectedMoney := decimal.NewFromFloat(topUp.PaidMoney()).Round(2)
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
