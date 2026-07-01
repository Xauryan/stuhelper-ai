package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/pkg/cachex"
	"github.com/Xauryan/stuhelper-ai/setting"
	"github.com/samber/hot"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Subscription duration units
const (
	SubscriptionDurationYear   = "year"
	SubscriptionDurationMonth  = "month"
	SubscriptionDurationDay    = "day"
	SubscriptionDurationHour   = "hour"
	SubscriptionDurationCustom = "custom"
)

// Subscription quota reset period
const (
	SubscriptionResetNever   = "never"
	SubscriptionResetDaily   = "daily"
	SubscriptionResetWeekly  = "weekly"
	SubscriptionResetMonthly = "monthly"
	SubscriptionResetCustom  = "custom"
)

var (
	ErrSubscriptionOrderNotFound      = errors.New("subscription order not found")
	ErrSubscriptionOrderStatusInvalid = errors.New("subscription order status invalid")
	ErrSubscriptionPurchaseLimit      = errors.New("已达到该套餐购买上限")
)

const (
	subscriptionPlanCacheNamespace     = "StuHelper AI:subscription_plan:v1"
	subscriptionPlanInfoCacheNamespace = "StuHelper AI:subscription_plan_info:v1"
)

var (
	subscriptionPlanCacheOnce     sync.Once
	subscriptionPlanInfoCacheOnce sync.Once

	subscriptionPlanCache     *cachex.HybridCache[SubscriptionPlan]
	subscriptionPlanInfoCache *cachex.HybridCache[SubscriptionPlanInfo]
)

func subscriptionPlanCacheTTL() time.Duration {
	ttlSeconds := common.GetEnvOrDefault("SUBSCRIPTION_PLAN_CACHE_TTL", 300)
	if ttlSeconds <= 0 {
		ttlSeconds = 300
	}
	return time.Duration(ttlSeconds) * time.Second
}

func subscriptionPlanInfoCacheTTL() time.Duration {
	ttlSeconds := common.GetEnvOrDefault("SUBSCRIPTION_PLAN_INFO_CACHE_TTL", 120)
	if ttlSeconds <= 0 {
		ttlSeconds = 120
	}
	return time.Duration(ttlSeconds) * time.Second
}

func subscriptionPlanCacheCapacity() int {
	capacity := common.GetEnvOrDefault("SUBSCRIPTION_PLAN_CACHE_CAP", 5000)
	if capacity <= 0 {
		capacity = 5000
	}
	return capacity
}

func subscriptionPlanInfoCacheCapacity() int {
	capacity := common.GetEnvOrDefault("SUBSCRIPTION_PLAN_INFO_CACHE_CAP", 10000)
	if capacity <= 0 {
		capacity = 10000
	}
	return capacity
}

func getSubscriptionPlanCache() *cachex.HybridCache[SubscriptionPlan] {
	subscriptionPlanCacheOnce.Do(func() {
		ttl := subscriptionPlanCacheTTL()
		subscriptionPlanCache = cachex.NewHybridCache[SubscriptionPlan](cachex.HybridCacheConfig[SubscriptionPlan]{
			Namespace: cachex.Namespace(subscriptionPlanCacheNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[SubscriptionPlan]{},
			Memory: func() *hot.HotCache[string, SubscriptionPlan] {
				return hot.NewHotCache[string, SubscriptionPlan](hot.LRU, subscriptionPlanCacheCapacity()).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return subscriptionPlanCache
}

func getSubscriptionPlanInfoCache() *cachex.HybridCache[SubscriptionPlanInfo] {
	subscriptionPlanInfoCacheOnce.Do(func() {
		ttl := subscriptionPlanInfoCacheTTL()
		subscriptionPlanInfoCache = cachex.NewHybridCache[SubscriptionPlanInfo](cachex.HybridCacheConfig[SubscriptionPlanInfo]{
			Namespace: cachex.Namespace(subscriptionPlanInfoCacheNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[SubscriptionPlanInfo]{},
			Memory: func() *hot.HotCache[string, SubscriptionPlanInfo] {
				return hot.NewHotCache[string, SubscriptionPlanInfo](hot.LRU, subscriptionPlanInfoCacheCapacity()).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return subscriptionPlanInfoCache
}

func subscriptionPlanCacheKey(id int) string {
	if id <= 0 {
		return ""
	}
	return strconv.Itoa(id)
}

func InvalidateSubscriptionPlanCache(planId int) {
	if planId <= 0 {
		return
	}
	cache := getSubscriptionPlanCache()
	_, _ = cache.DeleteMany([]string{subscriptionPlanCacheKey(planId)})
	infoCache := getSubscriptionPlanInfoCache()
	_ = infoCache.Purge()
}

// Subscription plan
type SubscriptionPlan struct {
	Id int `json:"id"`

	Title    string `json:"title" gorm:"type:varchar(128);not null"`
	Subtitle string `json:"subtitle" gorm:"type:varchar(255);default:''"`

	// Display money amount (follow existing code style: float64 for money)
	PriceAmount float64 `json:"price_amount" gorm:"type:decimal(10,6);not null;default:0"`
	Currency    string  `json:"currency" gorm:"type:varchar(8);not null;default:'USD'"`

	DurationUnit  string `json:"duration_unit" gorm:"type:varchar(16);not null;default:'month'"`
	DurationValue int    `json:"duration_value" gorm:"type:int;not null;default:1"`
	CustomSeconds int64  `json:"custom_seconds" gorm:"type:bigint;not null;default:0"`

	Enabled   bool `json:"enabled" gorm:"default:true"`
	SortOrder int  `json:"sort_order" gorm:"type:int;default:0"`
	// Recommended controls the user-facing recommendation badge and highlight.
	Recommended bool `json:"recommended" gorm:"default:false"`
	// AllowBalancePay controls whether users can purchase this plan with wallet quota.
	AllowBalancePay *bool `json:"allow_balance_pay"`
	// Allow falling back to wallet balance after subscription quota is exhausted (empty = true).
	AllowWalletOverflow *bool `json:"allow_wallet_overflow"`

	StripePriceId  string `json:"stripe_price_id" gorm:"type:varchar(128);default:''"`
	CreemProductId string `json:"creem_product_id" gorm:"type:varchar(128);default:''"`

	// Max purchases per user (0 = unlimited)
	MaxPurchasePerUser int `json:"max_purchase_per_user" gorm:"type:int;default:0"`

	// Upgrade user group after purchase (empty = no change)
	UpgradeGroup string `json:"upgrade_group" gorm:"type:varchar(64);default:''"`
	// Downgrade user group on expiry (empty = revert to the group held before purchase)
	DowngradeGroup string `json:"downgrade_group" gorm:"type:varchar(64);default:''"`

	// Total quota (amount in quota units, 0 = unlimited)
	TotalAmount int64 `json:"total_amount" gorm:"type:bigint;not null;default:0"`

	// Quota reset period for plan
	QuotaResetPeriod        string `json:"quota_reset_period" gorm:"type:varchar(16);default:'never'"`
	QuotaResetCustomSeconds int64  `json:"quota_reset_custom_seconds" gorm:"type:bigint;default:0"`

	// Model limits restrict which request models can be billed by this plan.
	ModelLimitsEnabled bool   `json:"model_limits_enabled" gorm:"default:false"`
	ModelLimits        string `json:"model_limits" gorm:"type:text;default:''"`

	CreatedAt int64 `json:"created_at" gorm:"bigint"`
	UpdatedAt int64 `json:"updated_at" gorm:"bigint"`
}

func normalizeSubscriptionModelLimits(enabled bool, csv string) []string {
	if !enabled || strings.TrimSpace(csv) == "" {
		return []string{}
	}
	parts := strings.Split(csv, ",")
	limits := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		modelName := strings.TrimSpace(part)
		if modelName == "" {
			continue
		}
		if _, ok := seen[modelName]; ok {
			continue
		}
		seen[modelName] = struct{}{}
		limits = append(limits, modelName)
	}
	return limits
}

// NormalizeModelLimitsForSave returns a stable enabled flag and CSV for storage.
func NormalizeModelLimitsForSave(enabled bool, csv string) (bool, string) {
	limits := normalizeSubscriptionModelLimits(enabled, csv)
	if len(limits) == 0 {
		return false, ""
	}
	return true, strings.Join(limits, ",")
}

// GetModelLimits returns the normalized unique model list for enabled limits.
func (p *SubscriptionPlan) GetModelLimits() []string {
	if p == nil {
		return []string{}
	}
	return normalizeSubscriptionModelLimits(p.ModelLimitsEnabled, p.ModelLimits)
}

// IsModelAllowed reports whether this plan can bill the requested model.
func (p *SubscriptionPlan) IsModelAllowed(modelName string) bool {
	if p == nil {
		return false
	}
	modelName = strings.TrimSpace(modelName)
	if !p.ModelLimitsEnabled || modelName == "" {
		return true
	}
	limits := p.GetModelLimits()
	if len(limits) == 0 {
		return true
	}
	for _, allowed := range limits {
		if allowed == modelName {
			return true
		}
	}
	return false
}

func (p *SubscriptionPlan) NormalizeDefaults() {
	if p.AllowBalancePay == nil {
		p.AllowBalancePay = common.GetPointer(true)
	}
	if p.AllowWalletOverflow == nil {
		p.AllowWalletOverflow = common.GetPointer(true)
	}
}

func (p *SubscriptionPlan) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	p.CreatedAt = now
	p.UpdatedAt = now
	p.ModelLimitsEnabled, p.ModelLimits = NormalizeModelLimitsForSave(p.ModelLimitsEnabled, p.ModelLimits)
	return nil
}

func (p *SubscriptionPlan) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = common.GetTimestamp()
	p.ModelLimitsEnabled, p.ModelLimits = NormalizeModelLimitsForSave(p.ModelLimitsEnabled, p.ModelLimits)
	return nil
}

// Subscription order (payment -> webhook -> create UserSubscription)
type SubscriptionOrder struct {
	Id     int     `json:"id"`
	UserId int     `json:"user_id" gorm:"index"`
	PlanId int     `json:"plan_id" gorm:"index"`
	Money  float64 `json:"money"`
	Fee    float64 `json:"fee" gorm:"default:0"`

	TradeNo         string `json:"trade_no" gorm:"unique;type:varchar(255);index"`
	PaymentMethod   string `json:"payment_method" gorm:"type:varchar(50)"`
	PaymentProvider string `json:"payment_provider" gorm:"type:varchar(50);default:''"`
	Status          string `json:"status"`
	CreateTime      int64  `json:"create_time"`
	CompleteTime    int64  `json:"complete_time"`

	ProviderPayload string `json:"provider_payload" gorm:"type:text"`
}

const subscriptionProviderPayloadIDKey = "subscription_id"

func providerPayloadWithSubscriptionId(payload string, subscriptionId int) string {
	payload = strings.TrimSpace(payload)
	if subscriptionId <= 0 {
		return payload
	}
	if _, ok := providerPayloadSubscriptionID(payload); ok {
		return payload
	}
	if strings.HasPrefix(payload, "{") {
		var data map[string]interface{}
		if err := common.Unmarshal([]byte(payload), &data); err == nil && data != nil {
			data[subscriptionProviderPayloadIDKey] = subscriptionId
			if encoded, err := common.Marshal(data); err == nil {
				return string(encoded)
			}
		}
	}
	idPart := fmt.Sprintf("%s=%d", subscriptionProviderPayloadIDKey, subscriptionId)
	if payload == "" {
		return idPart
	}
	return payload + ";" + idPart
}

func providerPayloadSubscriptionID(payload string) (int, bool) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return 0, false
	}
	if strings.HasPrefix(payload, "{") {
		var data map[string]interface{}
		if err := common.Unmarshal([]byte(payload), &data); err == nil && data != nil {
			if id, ok := providerPayloadValueAsPositiveInt(data[subscriptionProviderPayloadIDKey]); ok {
				return id, true
			}
		}
	}
	for _, part := range strings.FieldsFunc(payload, func(r rune) bool {
		return r == '&' || r == ';' || r == ',' || r == '\n' || r == '\r'
	}) {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok || strings.TrimSpace(key) != subscriptionProviderPayloadIDKey {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil && id > 0 {
			return id, true
		}
	}
	return 0, false
}

func providerPayloadValueAsPositiveInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, v > 0
	case int64:
		if v > 0 && v <= int64(^uint(0)>>1) {
			return int(v), true
		}
	case float64:
		id := int(v)
		if v > 0 && float64(id) == v {
			return id, true
		}
	case string:
		id, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil && id > 0 {
			return id, true
		}
	}
	return 0, false
}

func (o SubscriptionOrder) PaidMoney() float64 {
	money := decimal.NewFromFloat(o.Money).Round(2)
	fee := decimal.NewFromFloat(o.Fee).Round(2)
	if fee.IsNegative() {
		fee = decimal.Zero
	}
	return money.Add(fee).Round(2).InexactFloat64()
}

func (o *SubscriptionOrder) Insert() error {
	if o.CreateTime == 0 {
		o.CreateTime = common.GetTimestamp()
	}
	if strings.TrimSpace(o.Status) == "" {
		o.Status = common.TopUpStatusPending
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := reserveSubscriptionPurchaseSlotTx(tx, o); err != nil {
			return err
		}
		if err := tx.Create(o).Error; err != nil {
			return err
		}
		return upsertSubscriptionTopUpTx(tx, o)
	})
}

func (o *SubscriptionOrder) Update() error {
	return DB.Save(o).Error
}

func GetSubscriptionOrderByTradeNo(tradeNo string) *SubscriptionOrder {
	if tradeNo == "" {
		return nil
	}
	var order SubscriptionOrder
	if err := DB.Where("trade_no = ?", tradeNo).First(&order).Error; err != nil {
		return nil
	}
	return &order
}

// User subscription instance
type UserSubscription struct {
	Id     int `json:"id"`
	UserId int `json:"user_id" gorm:"index;index:idx_user_sub_active,priority:1"`
	PlanId int `json:"plan_id" gorm:"index"`

	AmountTotal int64 `json:"amount_total" gorm:"type:bigint;not null;default:0"`
	AmountUsed  int64 `json:"amount_used" gorm:"type:bigint;not null;default:0"`

	StartTime int64  `json:"start_time" gorm:"bigint"`
	EndTime   int64  `json:"end_time" gorm:"bigint;index;index:idx_user_sub_active,priority:3"`
	Status    string `json:"status" gorm:"type:varchar(32);index;index:idx_user_sub_active,priority:2"` // active/expired/cancelled

	Source string `json:"source" gorm:"type:varchar(32);default:'order'"` // order/admin

	LastResetTime int64 `json:"last_reset_time" gorm:"type:bigint;default:0"`
	NextResetTime int64 `json:"next_reset_time" gorm:"type:bigint;default:0;index"`

	UpgradeGroup  string `json:"upgrade_group" gorm:"type:varchar(64);default:''"`
	PrevUserGroup string `json:"prev_user_group" gorm:"type:varchar(64);default:''"`
	// Downgrade target group on expiry (snapshot from plan; empty = revert to PrevUserGroup)
	DowngradeGroup string `json:"downgrade_group" gorm:"type:varchar(64);default:''"`
	// Whether wallet fallback is allowed after this subscription's quota is exhausted.
	AllowWalletOverflow *bool `json:"allow_wallet_overflow"`

	CreatedAt int64 `json:"created_at" gorm:"bigint"`
	UpdatedAt int64 `json:"updated_at" gorm:"bigint"`
}

func (s *UserSubscription) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	s.CreatedAt = now
	s.UpdatedAt = now
	return nil
}

func (s *UserSubscription) BeforeUpdate(tx *gorm.DB) error {
	s.UpdatedAt = common.GetTimestamp()
	return nil
}

func subscriptionUpgradeGroupActiveAt(sub *UserSubscription, now int64) bool {
	if sub == nil {
		return false
	}
	return sub.Status == "active" &&
		strings.TrimSpace(sub.UpgradeGroup) != "" &&
		sub.StartTime <= now &&
		sub.EndTime > now
}

type SubscriptionSummary struct {
	Subscription *UserSubscription     `json:"subscription"`
	Plan         *SubscriptionPlanInfo `json:"plan,omitempty"`
}

func calcPlanEndTime(start time.Time, plan *SubscriptionPlan) (int64, error) {
	if plan == nil {
		return 0, errors.New("plan is nil")
	}
	if plan.DurationValue <= 0 && plan.DurationUnit != SubscriptionDurationCustom {
		return 0, errors.New("duration_value must be > 0")
	}
	switch plan.DurationUnit {
	case SubscriptionDurationYear:
		return start.AddDate(plan.DurationValue, 0, 0).Unix(), nil
	case SubscriptionDurationMonth:
		return start.AddDate(0, plan.DurationValue, 0).Unix(), nil
	case SubscriptionDurationDay:
		return start.Add(time.Duration(plan.DurationValue) * 24 * time.Hour).Unix(), nil
	case SubscriptionDurationHour:
		return start.Add(time.Duration(plan.DurationValue) * time.Hour).Unix(), nil
	case SubscriptionDurationCustom:
		if plan.CustomSeconds <= 0 {
			return 0, errors.New("custom_seconds must be > 0")
		}
		return start.Add(time.Duration(plan.CustomSeconds) * time.Second).Unix(), nil
	default:
		return 0, fmt.Errorf("invalid duration_unit: %s", plan.DurationUnit)
	}
}

func NormalizeResetPeriod(period string) string {
	switch strings.TrimSpace(period) {
	case SubscriptionResetDaily, SubscriptionResetWeekly, SubscriptionResetMonthly, SubscriptionResetCustom:
		return strings.TrimSpace(period)
	default:
		return SubscriptionResetNever
	}
}

func calcNextResetTime(base time.Time, plan *SubscriptionPlan, endUnix int64) int64 {
	if plan == nil {
		return 0
	}
	period := NormalizeResetPeriod(plan.QuotaResetPeriod)
	if period == SubscriptionResetNever {
		return 0
	}
	var next time.Time
	switch period {
	case SubscriptionResetDaily:
		next = time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, base.Location()).
			AddDate(0, 0, 1)
	case SubscriptionResetWeekly:
		// Align to next Monday 00:00
		weekday := int(base.Weekday()) // Sunday=0
		// Convert to Monday=1..Sunday=7
		if weekday == 0 {
			weekday = 7
		}
		daysUntil := 8 - weekday
		next = time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, base.Location()).
			AddDate(0, 0, daysUntil)
	case SubscriptionResetMonthly:
		// Align to first day of next month 00:00
		next = time.Date(base.Year(), base.Month(), 1, 0, 0, 0, 0, base.Location()).
			AddDate(0, 1, 0)
	case SubscriptionResetCustom:
		if plan.QuotaResetCustomSeconds <= 0 {
			return 0
		}
		next = base.Add(time.Duration(plan.QuotaResetCustomSeconds) * time.Second)
	default:
		return 0
	}
	if endUnix > 0 && next.Unix() > endUnix {
		return 0
	}
	return next.Unix()
}

func GetSubscriptionPlanById(id int) (*SubscriptionPlan, error) {
	return getSubscriptionPlanByIdTx(nil, id)
}

func getSubscriptionPlanByIdTx(tx *gorm.DB, id int) (*SubscriptionPlan, error) {
	if id <= 0 {
		return nil, errors.New("invalid plan id")
	}
	key := subscriptionPlanCacheKey(id)
	if key != "" {
		if cached, found, err := getSubscriptionPlanCache().Get(key); err == nil && found {
			cached.NormalizeDefaults()
			return &cached, nil
		}
	}
	var plan SubscriptionPlan
	query := DB
	if tx != nil {
		query = tx
	}
	if err := query.Where("id = ?", id).First(&plan).Error; err != nil {
		return nil, err
	}
	plan.NormalizeDefaults()
	_ = getSubscriptionPlanCache().SetWithTTL(key, plan, subscriptionPlanCacheTTL())
	return &plan, nil
}

func CountUserSubscriptionsByPlan(userId int, planId int) (int64, error) {
	if userId <= 0 || planId <= 0 {
		return 0, errors.New("invalid userId or planId")
	}
	var count int64
	if err := DB.Model(&UserSubscription{}).
		Where("user_id = ? AND plan_id = ?", userId, planId).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func reserveSubscriptionPurchaseSlotTx(tx *gorm.DB, order *SubscriptionOrder) error {
	if tx == nil || order == nil {
		return errors.New("invalid subscription order")
	}
	if order.UserId <= 0 || order.PlanId <= 0 {
		return errors.New("invalid subscription order")
	}
	if order.Status != common.TopUpStatusPending && order.Status != common.TopUpStatusSuccess {
		return nil
	}
	if err := lockSubscriptionPurchaseUserTx(tx, order.UserId); err != nil {
		return err
	}
	plan, err := getSubscriptionPlanByIdTx(tx, order.PlanId)
	if err != nil {
		return err
	}
	if plan.MaxPurchasePerUser <= 0 {
		return nil
	}
	var subscriptionCount int64
	if err := tx.Model(&UserSubscription{}).
		Where("user_id = ? AND plan_id = ?", order.UserId, order.PlanId).
		Count(&subscriptionCount).Error; err != nil {
		return err
	}
	var pendingOrderCount int64
	if err := tx.Model(&SubscriptionOrder{}).
		Where("user_id = ? AND plan_id = ? AND status = ?", order.UserId, order.PlanId, common.TopUpStatusPending).
		Count(&pendingOrderCount).Error; err != nil {
		return err
	}
	if subscriptionCount+pendingOrderCount >= int64(plan.MaxPurchasePerUser) {
		return ErrSubscriptionPurchaseLimit
	}
	return nil
}

func getUserGroupByIdTx(tx *gorm.DB, userId int) (string, error) {
	if userId <= 0 {
		return "", errors.New("invalid userId")
	}
	if tx == nil {
		tx = DB
	}
	var group string
	if err := tx.Model(&User{}).Where("id = ?", userId).Select(commonGroupCol).Find(&group).Error; err != nil {
		return "", err
	}
	return group, nil
}

func downgradeUserGroupForSubscriptionTx(tx *gorm.DB, sub *UserSubscription, now int64) (string, error) {
	if tx == nil || sub == nil {
		return "", errors.New("invalid downgrade args")
	}
	downgradeGroup := strings.TrimSpace(sub.DowngradeGroup)
	upgradeGroup := strings.TrimSpace(sub.UpgradeGroup)
	if downgradeGroup == "" && upgradeGroup == "" {
		return "", nil
	}
	currentGroup, err := getUserGroupByIdTx(tx, sub.UserId)
	if err != nil {
		return "", err
	}
	var activeSub UserSubscription
	activeQuery := tx.Where("user_id = ? AND status = ? AND start_time <= ? AND end_time > ? AND id <> ? AND upgrade_group <> ''",
		sub.UserId, "active", now, now, sub.Id).
		Order("end_time desc, id desc").
		Limit(1).
		Find(&activeSub)
	if activeQuery.Error == nil && activeQuery.RowsAffected > 0 {
		return "", nil
	}
	target := downgradeGroup
	if target == "" {
		if currentGroup != upgradeGroup {
			return "", nil
		}
		target = strings.TrimSpace(sub.PrevUserGroup)
	}
	if target == "" || target == currentGroup {
		return "", nil
	}
	if err := tx.Model(&User{}).Where("id = ?", sub.UserId).
		Update("group", target).Error; err != nil {
		return "", err
	}
	return target, nil
}

func CreateUserSubscriptionFromPlanTx(tx *gorm.DB, userId int, plan *SubscriptionPlan, source string) (*UserSubscription, error) {
	if tx == nil {
		return nil, errors.New("tx is nil")
	}
	if plan == nil || plan.Id == 0 {
		return nil, errors.New("invalid plan")
	}
	if userId <= 0 {
		return nil, errors.New("invalid user id")
	}
	if err := lockSubscriptionPurchaseUserTx(tx, userId); err != nil {
		return nil, err
	}
	return createUserSubscriptionFromPlanTxLocked(tx, userId, plan, source)
}

func createUserSubscriptionFromPlanTxLocked(tx *gorm.DB, userId int, plan *SubscriptionPlan, source string) (*UserSubscription, error) {
	if tx == nil {
		return nil, errors.New("tx is nil")
	}
	if plan == nil || plan.Id == 0 {
		return nil, errors.New("invalid plan")
	}
	if userId <= 0 {
		return nil, errors.New("invalid user id")
	}
	if plan.MaxPurchasePerUser > 0 {
		var count int64
		if err := tx.Model(&UserSubscription{}).
			Where("user_id = ? AND plan_id = ?", userId, plan.Id).
			Count(&count).Error; err != nil {
			return nil, err
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			return nil, ErrSubscriptionPurchaseLimit
		}
	}
	nowUnix := getDBTimestampTx(tx)
	startUnix, previousChainSub, err := nextSubscriptionStartTimeTx(tx, userId, plan.Id, nowUnix)
	if err != nil {
		return nil, err
	}
	start := time.Unix(startUnix, 0)
	endUnix, err := calcPlanEndTime(start, plan)
	if err != nil {
		return nil, err
	}
	resetBase := start
	nextReset := calcNextResetTime(resetBase, plan, endUnix)
	lastReset := int64(0)
	if nextReset > 0 {
		lastReset = start.Unix()
	}
	upgradeGroup := strings.TrimSpace(plan.UpgradeGroup)
	prevGroup := ""
	if upgradeGroup != "" {
		currentGroup, err := getUserGroupByIdTx(tx, userId)
		if err != nil {
			return nil, err
		}
		if previousChainSub != nil && strings.TrimSpace(previousChainSub.PrevUserGroup) != "" {
			prevGroup = strings.TrimSpace(previousChainSub.PrevUserGroup)
		}
		if currentGroup != upgradeGroup {
			if prevGroup == "" {
				prevGroup = currentGroup
			}
			if startUnix <= nowUnix {
				if err := tx.Model(&User{}).Where("id = ?", userId).
					Update("group", upgradeGroup).Error; err != nil {
					return nil, err
				}
			}
		}
	}
	allowWalletOverflow := true
	if plan.AllowWalletOverflow != nil {
		allowWalletOverflow = *plan.AllowWalletOverflow
	}
	sub := &UserSubscription{
		UserId:              userId,
		PlanId:              plan.Id,
		AmountTotal:         plan.TotalAmount,
		AmountUsed:          0,
		StartTime:           start.Unix(),
		EndTime:             endUnix,
		Status:              "active",
		Source:              source,
		LastResetTime:       lastReset,
		NextResetTime:       nextReset,
		UpgradeGroup:        upgradeGroup,
		PrevUserGroup:       prevGroup,
		DowngradeGroup:      strings.TrimSpace(plan.DowngradeGroup),
		AllowWalletOverflow: common.GetPointer(allowWalletOverflow),
		CreatedAt:           nowUnix,
		UpdatedAt:           nowUnix,
	}
	if err := tx.Create(sub).Error; err != nil {
		return nil, err
	}
	return sub, nil
}

func nextSubscriptionStartTimeTx(tx *gorm.DB, userId int, planId int, nowUnix int64) (int64, *UserSubscription, error) {
	if tx == nil {
		return 0, nil, errors.New("tx is nil")
	}
	if userId <= 0 || planId <= 0 {
		return 0, nil, errors.New("invalid subscription start args")
	}
	var latest UserSubscription
	query := tx.Where("user_id = ? AND plan_id = ? AND status = ? AND end_time > ?",
		userId,
		planId,
		"active",
		nowUnix,
	).
		Order("end_time desc, id desc").
		Limit(1).
		Find(&latest)
	if query.Error != nil {
		return 0, nil, query.Error
	}
	if query.RowsAffected > 0 && latest.EndTime > nowUnix {
		return latest.EndTime, &latest, nil
	}
	return nowUnix, nil, nil
}

// Complete a subscription order (idempotent). Creates a UserSubscription snapshot from the plan.
// expectedPaymentProvider guards against cross-gateway callback attacks (empty skips the check).
// actualPaymentMethod updates the order's PaymentMethod to reflect the real payment type used (empty skips update).
func CompleteSubscriptionOrder(tradeNo string, providerPayload string, expectedPaymentProvider string, actualPaymentMethod string) error {
	if tradeNo == "" {
		return errors.New("tradeNo is empty")
	}
	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}
	var logUserId int
	var logPlanTitle string
	var logMoney float64
	var logPaymentMethod string
	var upgradeGroup string
	upgradeGroupApplied := false
	var logOrderId int
	var referralResult *ReferralCommissionCreditResult
	err := DB.Transaction(func(tx *gorm.DB) error {
		var order SubscriptionOrder
		if err := withRowLock(tx).Where(refCol+" = ?", tradeNo).First(&order).Error; err != nil {
			return ErrSubscriptionOrderNotFound
		}
		if expectedPaymentProvider != "" && order.PaymentProvider != expectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if order.Status == common.TopUpStatusSuccess {
			return nil
		}
		if order.Status != common.TopUpStatusPending &&
			!(order.Status == common.TopUpStatusExpired && IsOfficialPaymentProvider(order.PaymentProvider)) {
			return ErrSubscriptionOrderStatusInvalid
		}
		plan, err := getSubscriptionPlanByIdTx(tx, order.PlanId)
		if err != nil {
			return err
		}
		if !plan.Enabled {
			// still allow completion for already purchased orders
		}
		if actualPaymentMethod != "" && order.PaymentMethod != actualPaymentMethod {
			order.PaymentMethod = actualPaymentMethod
		}
		upgradeGroup = strings.TrimSpace(plan.UpgradeGroup)
		subscription, err := CreateUserSubscriptionFromPlanTx(tx, order.UserId, plan, "order")
		if err != nil {
			return err
		}
		upgradeGroupApplied = subscriptionUpgradeGroupActiveAt(subscription, getDBTimestampTx(tx))
		order.Status = common.TopUpStatusSuccess
		order.CompleteTime = common.GetTimestamp()
		payload := providerPayload
		if payload == "" {
			payload = order.ProviderPayload
		}
		payload = providerPayloadWithSubscriptionId(payload, subscription.Id)
		if payload != "" {
			order.ProviderPayload = payload
		}
		if err := upsertSubscriptionTopUpTx(tx, &order); err != nil {
			return err
		}
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		logUserId = order.UserId
		logPlanTitle = plan.Title
		logMoney = order.Money
		logPaymentMethod = order.PaymentMethod
		logOrderId = order.Id
		referralResult, err = CreditInviteRewardsAfterPaymentTx(tx, order.UserId, order.Money, order.PaymentMethod, ReferralCommissionSourceSubscription, order.Id)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	if upgradeGroup != "" && upgradeGroupApplied && logUserId > 0 {
		_ = UpdateUserGroupCache(logUserId, upgradeGroup)
	}
	if logUserId > 0 {
		msg := fmt.Sprintf("订阅购买成功，套餐: %s，支付金额: %.2f，支付方式: %s", logPlanTitle, logMoney, logPaymentMethod)
		RecordLog(logUserId, LogTypeTopup, msg)
		common.SysLog(fmt.Sprintf("订阅订单完成 subscription_order_id=%d", logOrderId))
		RecordReferralCommissionLog(referralResult)
	}
	return nil
}

func upsertSubscriptionTopUpTx(tx *gorm.DB, order *SubscriptionOrder) error {
	if tx == nil || order == nil {
		return errors.New("invalid subscription order")
	}
	status := strings.TrimSpace(order.Status)
	if status == "" {
		status = common.TopUpStatusPending
	}
	completeTime := order.CompleteTime
	switch status {
	case common.TopUpStatusSuccess, common.TopUpStatusExpired, common.TopUpStatusFailed:
		if completeTime == 0 {
			completeTime = common.GetTimestamp()
		}
	default:
		completeTime = 0
	}
	var topup TopUp
	if err := tx.Where("trade_no = ?", order.TradeNo).First(&topup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			topup = TopUp{
				UserId:          order.UserId,
				Amount:          0,
				Money:           order.Money,
				Fee:             order.Fee,
				TradeNo:         order.TradeNo,
				PaymentMethod:   order.PaymentMethod,
				PaymentProvider: order.PaymentProvider,
				CreateTime:      order.CreateTime,
				CompleteTime:    completeTime,
				Status:          status,
			}
			return tx.Create(&topup).Error
		}
		return err
	}
	topup.Money = order.Money
	topup.Fee = order.Fee
	if topup.PaymentMethod == "" || (order.PaymentProvider == PaymentProviderEpay && order.PaymentMethod != "") {
		topup.PaymentMethod = order.PaymentMethod
	} else if topup.PaymentMethod != order.PaymentMethod {
		return ErrPaymentMethodMismatch
	}
	if topup.CreateTime == 0 {
		topup.CreateTime = order.CreateTime
	}
	if topup.PaymentProvider == "" {
		topup.PaymentProvider = order.PaymentProvider
	} else if order.PaymentProvider != "" && topup.PaymentProvider != order.PaymentProvider {
		return ErrPaymentMethodMismatch
	}
	topup.CompleteTime = completeTime
	topup.Status = status
	return tx.Save(&topup).Error
}

func ExpireSubscriptionOrder(tradeNo string, expectedPaymentProvider string) error {
	if tradeNo == "" {
		return errors.New("tradeNo is empty")
	}
	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var order SubscriptionOrder
		if err := withRowLock(tx).Where(refCol+" = ?", tradeNo).First(&order).Error; err != nil {
			return ErrSubscriptionOrderNotFound
		}
		if expectedPaymentProvider != "" && order.PaymentProvider != expectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if order.Status != common.TopUpStatusPending {
			return nil
		}
		order.Status = common.TopUpStatusExpired
		order.CompleteTime = common.GetTimestamp()
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		return upsertSubscriptionTopUpTx(tx, &order)
	})
}

func RefundSubscriptionOrder(tradeNo string, expectedPaymentProvider string, refundAmount float64, fullRefund bool) error {
	tradeNo = strings.TrimSpace(tradeNo)
	if tradeNo == "" {
		return errors.New("tradeNo is empty")
	}
	refundMoney := decimal.NewFromFloat(refundAmount).Round(2)
	if !refundMoney.IsPositive() {
		return errors.New("退款金额必须大于 0")
	}
	return SyncSubscriptionOrderRefundState(tradeNo, expectedPaymentProvider, fullRefund)
}

func SyncSubscriptionOrderRefundState(tradeNo string, expectedPaymentProvider string, fullRefund bool) error {
	tradeNo = strings.TrimSpace(tradeNo)
	if tradeNo == "" {
		return errors.New("tradeNo is empty")
	}
	now := common.GetTimestamp()
	cacheGroup := ""
	cacheUserId := 0
	if err := DB.Transaction(func(tx *gorm.DB) error {
		userId, group, err := syncSubscriptionOrderRefundStateTx(tx, tradeNo, expectedPaymentProvider, fullRefund, now)
		cacheUserId = userId
		cacheGroup = group
		return err
	}); err != nil {
		return err
	}
	if cacheGroup != "" && cacheUserId > 0 {
		_ = UpdateUserGroupCache(cacheUserId, cacheGroup)
	}
	return nil
}

func syncSubscriptionOrderRefundStateTx(tx *gorm.DB, tradeNo string, expectedPaymentProvider string, fullRefund bool, now int64) (int, string, error) {
	if tx == nil {
		return 0, "", errors.New("tx is nil")
	}
	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}
	var order SubscriptionOrder
	if err := withRowLock(tx).Where(refCol+" = ?", tradeNo).First(&order).Error; err != nil {
		return 0, "", ErrSubscriptionOrderNotFound
	}
	if expectedPaymentProvider != "" && order.PaymentProvider != expectedPaymentProvider {
		return 0, "", ErrPaymentMethodMismatch
	}
	if order.Status != common.TopUpStatusSuccess &&
		order.Status != common.TopUpStatusPartialRefunded &&
		order.Status != common.TopUpStatusRefunded {
		return 0, "", ErrSubscriptionOrderStatusInvalid
	}
	orderMoney := decimal.NewFromFloat(order.Money).Round(2)
	topUp := &TopUp{}
	if err := withRowLock(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
		return 0, "", err
	}
	refundedMoney := decimal.NewFromFloat(topUp.RefundedMoney).Round(2)
	if refundedMoney.GreaterThan(orderMoney) {
		return 0, "", errors.New("退款金额超过可退金额")
	}
	nextStatus := common.TopUpStatusSuccess
	if refundedMoney.IsPositive() {
		nextStatus = common.TopUpStatusPartialRefunded
	}
	if refundedMoney.IsPositive() && (fullRefund || !refundedMoney.LessThan(orderMoney)) {
		nextStatus = common.TopUpStatusRefunded
	}
	order.Status = nextStatus
	if nextStatus != common.TopUpStatusSuccess {
		order.CompleteTime = now
	}
	if err := tx.Save(&order).Error; err != nil {
		return 0, "", err
	}
	topUp.Status = nextStatus
	topUp.RefundedMoney = refundedMoney.InexactFloat64()
	if order.PaymentProvider != PaymentProviderBalance && topUp.PaymentProvider != PaymentProviderBalance {
		topUp.RefundedQuota = 0
	}
	if err := tx.Save(topUp).Error; err != nil {
		return 0, "", err
	}

	if err := setReferralCommissionRefundTargetTx(tx, ReferralCommissionSourceSubscription, order.Id, refundedMoney, orderMoney); err != nil {
		return 0, "", err
	}

	cacheGroup := ""
	cacheUserId := 0
	sub, err := findRefundableUserSubscriptionForOrderTx(tx, order)
	if err != nil {
		return 0, "", err
	}
	if sub != nil {
		switch {
		case nextStatus == common.TopUpStatusRefunded || nextStatus == common.TopUpStatusPartialRefunded:
			if err := tx.Model(&UserSubscription{}).
				Where("id = ?", sub.Id).
				Updates(map[string]interface{}{
					"status":     "cancelled",
					"updated_at": now,
				}).Error; err != nil {
				return 0, "", err
			}
			target, err := downgradeUserGroupForSubscriptionTx(tx, sub, now)
			if err != nil {
				return 0, "", err
			}
			cacheGroup = target
			cacheUserId = sub.UserId
		case nextStatus == common.TopUpStatusSuccess && sub.Status == "cancelled":
			if err := tx.Model(&UserSubscription{}).
				Where("id = ?", sub.Id).
				Updates(map[string]interface{}{
					"status":     "active",
					"updated_at": now,
				}).Error; err != nil {
				return 0, "", err
			}
			if strings.TrimSpace(sub.UpgradeGroup) != "" {
				cacheGroup = strings.TrimSpace(sub.UpgradeGroup)
				cacheUserId = sub.UserId
			}
		}
	}
	if nextStatus == common.TopUpStatusRefunded {
		return cacheUserId, cacheGroup, reverseInviterRewardForFullRefundTx(tx, order.UserId)
	}
	return cacheUserId, cacheGroup, nil
}

// Admin bind (no payment). Creates a UserSubscription from a plan.
func AdminBindSubscription(userId int, planId int, sourceNote string) (string, error) {
	if userId <= 0 || planId <= 0 {
		return "", errors.New("invalid userId or planId")
	}
	plan, err := GetSubscriptionPlanById(planId)
	if err != nil {
		return "", err
	}
	var subscription *UserSubscription
	err = DB.Transaction(func(tx *gorm.DB) error {
		subscription, err = CreateUserSubscriptionFromPlanTx(tx, userId, plan, "admin")
		return err
	})
	if err != nil {
		return "", err
	}
	if subscriptionUpgradeGroupActiveAt(subscription, GetDBTimestamp()) {
		_ = UpdateUserGroupCache(userId, plan.UpgradeGroup)
		return fmt.Sprintf("用户分组将升级到 %s", plan.UpgradeGroup), nil
	}
	return "", nil
}

func calcSubscriptionBalanceQuota(priceAmount float64) (int, error) {
	if priceAmount <= 0 {
		return 0, nil
	}
	if common.QuotaPerUnit <= 0 {
		return 0, errors.New("额度单位配置错误")
	}
	quota := decimal.NewFromFloat(priceAmount).
		Mul(decimal.NewFromFloat(common.QuotaPerUnit)).
		Ceil().
		IntPart()
	return int(quota), nil
}

// PurchaseSubscriptionWithBalance creates a subscription by deducting the user's wallet quota.
func PurchaseSubscriptionWithBalance(userId int, planId int) error {
	if userId <= 0 || planId <= 0 {
		return errors.New("invalid userId or planId")
	}

	var logPlanTitle string
	var logMoney float64
	var chargedQuota int
	var upgradeGroup string
	upgradeGroupApplied := false
	var logOrderId int
	err := DB.Transaction(func(tx *gorm.DB) error {
		plan, err := getSubscriptionPlanByIdTx(tx, planId)
		if err != nil {
			return err
		}
		if !plan.Enabled {
			return errors.New("套餐未启用")
		}
		if plan.PriceAmount < 0 {
			return errors.New("套餐价格不能为负数")
		}
		if plan.AllowBalancePay != nil && !*plan.AllowBalancePay {
			return errors.New("该套餐不允许使用余额兑换")
		}

		requiredQuota, err := calcSubscriptionBalanceQuota(plan.PriceAmount)
		if err != nil {
			return err
		}

		var user User
		if err := withRowLock(tx).Where("id = ?", userId).First(&user).Error; err != nil {
			return err
		}
		if requiredQuota > 0 && user.Quota < requiredQuota {
			return errors.New("余额不足")
		}
		if requiredQuota > 0 {
			result := tx.Model(&User{}).Where("id = ? AND quota >= ?", userId, requiredQuota).
				Update("quota", gorm.Expr("quota - ?", requiredQuota))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return errors.New("余额不足")
			}
		}

		subscription, err := createUserSubscriptionFromPlanTxLocked(tx, userId, plan, "order")
		if err != nil {
			return err
		}

		now := common.GetTimestamp()
		tradeNo := BuildBalancePaymentTradeNo(userId)
		order := &SubscriptionOrder{
			UserId:          userId,
			PlanId:          plan.Id,
			Money:           plan.PriceAmount,
			TradeNo:         tradeNo,
			PaymentMethod:   PaymentMethodBalance,
			PaymentProvider: PaymentProviderBalance,
			Status:          common.TopUpStatusSuccess,
			CreateTime:      now,
			CompleteTime:    now,
			ProviderPayload: providerPayloadWithSubscriptionId(fmt.Sprintf("charged_quota=%d", requiredQuota), subscription.Id),
		}
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		if err := upsertSubscriptionTopUpTx(tx, order); err != nil {
			return err
		}

		logPlanTitle = plan.Title
		logMoney = plan.PriceAmount
		chargedQuota = requiredQuota
		upgradeGroup = strings.TrimSpace(plan.UpgradeGroup)
		upgradeGroupApplied = subscriptionUpgradeGroupActiveAt(subscription, getDBTimestampTx(tx))
		logOrderId = order.Id
		return nil
	})
	if err != nil {
		return err
	}

	if chargedQuota > 0 {
		if err := cacheDecrUserQuota(userId, int64(chargedQuota)); err != nil {
			common.SysLog("failed to decrease user quota cache after subscription balance purchase: " + err.Error())
		}
	}
	if upgradeGroup != "" && upgradeGroupApplied {
		_ = UpdateUserGroupCache(userId, upgradeGroup)
	}
	msg := fmt.Sprintf("使用余额购买订阅成功，套餐: %s，支付金额: %.2f，扣除额度: %d", logPlanTitle, logMoney, chargedQuota)
	RecordLog(userId, LogTypeTopup, msg)
	common.SysLog(fmt.Sprintf("余额订阅订单完成 subscription_order_id=%d", logOrderId))
	return nil
}

type SelfServeSubscriptionPurchaseParams struct {
	UserId        int
	PlanId        int
	PaymentMethod string
	DeclaredMoney float64
	TransactionNo string
}

type SelfServeSubscriptionPurchaseResult struct {
	Order         *SubscriptionOrder   `json:"order"`
	TopUp         *TopUp               `json:"topup"`
	Audit         *SelfServeTopUpAudit `json:"audit"`
	Subscription  *UserSubscription    `json:"subscription"`
	ExpectedMoney float64              `json:"expected_money"`
}

func selfServeSubscriptionTradeNo(paymentMethod string, userId int) string {
	return BuildSelfServePaymentTradeNo(paymentMethod, true, userId)
}

func calculateSelfServeSubscriptionMoney(priceAmount float64) decimal.Decimal {
	price := decimal.NewFromFloat(priceAmount)
	unitPrice := decimal.NewFromFloat(setting.SelfServeTopUpUnitPrice)
	return price.Mul(unitPrice).RoundCeil(2)
}

func PurchaseSubscriptionWithSelfServe(params SelfServeSubscriptionPurchaseParams) (*SelfServeSubscriptionPurchaseResult, error) {
	if params.UserId <= 0 || params.PlanId <= 0 {
		return nil, errors.New("invalid userId or planId")
	}
	paymentMethod := NormalizeSelfServePaymentMethod(params.PaymentMethod)
	if paymentMethod == "" {
		return nil, ErrPaymentMethodMismatch
	}
	declaredMoney := normalizeSelfServeMoney(params.DeclaredMoney)
	if !declaredMoney.IsPositive() {
		return nil, errors.New("支付金额必须大于 0")
	}
	if setting.SelfServeTopUpUnitPrice <= 0 {
		return nil, errors.New("自助充值价格配置错误")
	}

	var result SelfServeSubscriptionPurchaseResult
	upgradeGroup := ""
	upgradeGroupApplied := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		plan, err := getSubscriptionPlanByIdTx(tx, params.PlanId)
		if err != nil {
			return err
		}
		if !plan.Enabled {
			return errors.New("套餐未启用")
		}
		if plan.PriceAmount < 0.01 {
			return errors.New("套餐金额过低")
		}
		expectedMoney := calculateSelfServeSubscriptionMoney(plan.PriceAmount)
		if !declaredMoney.Equal(expectedMoney) {
			return fmt.Errorf("自助订阅支付金额应为 %.2f 元", expectedMoney.InexactFloat64())
		}
		if _, err := validateSelfServeTopUpMoneyTx(tx, params.UserId, declaredMoney, 0); err != nil {
			return err
		}
		transactionNo, err := selfServeResolveAuditTransactionNo(paymentMethod, true, params.UserId, params.TransactionNo)
		if err != nil {
			return err
		}
		var existingAudit SelfServeTopUpAudit
		if err := tx.Where("transaction_no = ?", transactionNo).First(&existingAudit).Error; err == nil {
			return errors.New("该交易订单号已提交")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var user User
		if err := withRowLock(tx).Select("id").Where("id = ?", params.UserId).First(&user).Error; err != nil {
			return err
		}
		subscription, err := createUserSubscriptionFromPlanTxLocked(tx, params.UserId, plan, "order")
		if err != nil {
			return err
		}
		now := common.GetTimestamp()
		tradeNo := selfServeSubscriptionTradeNo(paymentMethod, params.UserId)
		order := &SubscriptionOrder{
			UserId:          params.UserId,
			PlanId:          plan.Id,
			Money:           expectedMoney.InexactFloat64(),
			Fee:             0,
			TradeNo:         tradeNo,
			PaymentMethod:   paymentMethod,
			PaymentProvider: PaymentProviderSelfServe,
			Status:          common.TopUpStatusSuccess,
			CreateTime:      now,
			CompleteTime:    now,
			ProviderPayload: providerPayloadWithSubscriptionId(fmt.Sprintf("transaction_no=%s", transactionNo), subscription.Id),
		}
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		if err := upsertSubscriptionTopUpTx(tx, order); err != nil {
			return err
		}
		topUp := &TopUp{}
		if err := tx.Where("trade_no = ?", tradeNo).First(topUp).Error; err != nil {
			return err
		}
		audit := &SelfServeTopUpAudit{
			TopUpId:       topUp.Id,
			UserId:        params.UserId,
			TradeNo:       tradeNo,
			TransactionNo: transactionNo,
			PaymentMethod: paymentMethod,
			DeclaredMoney: expectedMoney.InexactFloat64(),
			CreditedQuota: 0,
			Status:        SelfServeTopUpAuditStatusPending,
			CreateTime:    now,
			UpdateTime:    now,
		}
		if err := tx.Create(audit).Error; err != nil {
			return err
		}
		upgradeGroup = strings.TrimSpace(plan.UpgradeGroup)
		upgradeGroupApplied = subscriptionUpgradeGroupActiveAt(subscription, getDBTimestampTx(tx))
		result = SelfServeSubscriptionPurchaseResult{
			Order:         order,
			TopUp:         topUp,
			Audit:         audit,
			Subscription:  subscription,
			ExpectedMoney: expectedMoney.InexactFloat64(),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if upgradeGroup != "" && upgradeGroupApplied {
		_ = UpdateUserGroupCache(params.UserId, upgradeGroup)
	}
	_ = InvalidateUserCache(params.UserId)
	msg := fmt.Sprintf("使用自助支付购买订阅成功，订单号：%s，支付金额：%.2f，支付方式：%s，等待管理员审核", result.Order.TradeNo, result.Order.Money, result.Order.PaymentMethod)
	RecordLog(params.UserId, LogTypeTopup, msg)
	return &result, nil
}

// GetAllActiveUserSubscriptions returns all active subscriptions for a user.
func GetAllActiveUserSubscriptions(userId int) ([]SubscriptionSummary, error) {
	if userId <= 0 {
		return nil, errors.New("invalid userId")
	}
	now := common.GetTimestamp()
	var subs []UserSubscription
	err := DB.Where("user_id = ? AND status = ? AND start_time <= ? AND end_time > ?", userId, "active", now, now).
		Order("end_time desc, id desc").
		Find(&subs).Error
	if err != nil {
		return nil, err
	}
	return buildSubscriptionSummaries(subs), nil
}

// HasActiveUserSubscription returns whether the user has any active subscription.
// This is a lightweight existence check to avoid heavy pre-consume transactions.
func HasActiveUserSubscription(userId int) (bool, error) {
	if userId <= 0 {
		return false, errors.New("invalid userId")
	}
	now := common.GetTimestamp()
	var count int64
	if err := DB.Model(&UserSubscription{}).
		Where("user_id = ? AND status = ? AND start_time <= ? AND end_time > ?", userId, "active", now, now).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// UserActiveSubscriptionsAllowWalletOverflow returns whether wallet balance may be used
// after the user's subscription quota is exhausted. Any active subscription snapshot
// with allow_wallet_overflow=false blocks the fallback.
func UserActiveSubscriptionsAllowWalletOverflow(userId int) (bool, error) {
	if userId <= 0 {
		return false, errors.New("invalid userId")
	}
	now := common.GetTimestamp()
	var strictCount int64
	if err := DB.Model(&UserSubscription{}).
		Where("user_id = ? AND status = ? AND start_time <= ? AND end_time > ? AND allow_wallet_overflow = ?",
			userId, "active", now, now, false).
		Count(&strictCount).Error; err != nil {
		return false, err
	}
	return strictCount == 0, nil
}

// GetAllUserSubscriptions returns all subscriptions (active and expired) for a user.
func GetAllUserSubscriptions(userId int) ([]SubscriptionSummary, error) {
	if userId <= 0 {
		return nil, errors.New("invalid userId")
	}
	var subs []UserSubscription
	err := DB.Where("user_id = ?", userId).
		Order("end_time desc, id desc").
		Find(&subs).Error
	if err != nil {
		return nil, err
	}
	return buildSubscriptionSummaries(subs), nil
}

func GetUserSubscriptionById(userSubscriptionId int) (*UserSubscription, error) {
	if userSubscriptionId <= 0 {
		return nil, errors.New("invalid userSubscriptionId")
	}
	var sub UserSubscription
	if err := DB.Where("id = ?", userSubscriptionId).First(&sub).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func buildSubscriptionSummaries(subs []UserSubscription) []SubscriptionSummary {
	if len(subs) == 0 {
		return []SubscriptionSummary{}
	}
	planIds := make([]int, 0)
	seenPlanIds := make(map[int]struct{})
	for _, sub := range subs {
		if sub.PlanId <= 0 {
			continue
		}
		if _, ok := seenPlanIds[sub.PlanId]; ok {
			continue
		}
		seenPlanIds[sub.PlanId] = struct{}{}
		planIds = append(planIds, sub.PlanId)
	}

	planInfoById := make(map[int]*SubscriptionPlanInfo, len(planIds))
	if len(planIds) > 0 {
		var plans []SubscriptionPlan
		if err := DB.Select("id", "title").Where("id IN ?", planIds).Find(&plans).Error; err == nil {
			for _, plan := range plans {
				planInfoById[plan.Id] = &SubscriptionPlanInfo{
					PlanId:    plan.Id,
					PlanTitle: plan.Title,
				}
			}
		}
	}

	result := make([]SubscriptionSummary, 0, len(subs))
	for _, sub := range subs {
		subCopy := sub
		summary := SubscriptionSummary{
			Subscription: &subCopy,
		}
		if planInfo, ok := planInfoById[sub.PlanId]; ok {
			summary.Plan = planInfo
		}
		result = append(result, summary)
	}
	return result
}

// AdminInvalidateUserSubscription marks a user subscription as cancelled and ends it immediately.
func AdminInvalidateUserSubscription(userSubscriptionId int) (string, error) {
	if userSubscriptionId <= 0 {
		return "", errors.New("invalid userSubscriptionId")
	}
	now := common.GetTimestamp()
	cacheGroup := ""
	downgradeGroup := ""
	var userId int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var sub UserSubscription
		if err := withRowLock(tx).
			Where("id = ?", userSubscriptionId).First(&sub).Error; err != nil {
			return err
		}
		userId = sub.UserId
		if err := tx.Model(&sub).Updates(map[string]interface{}{
			"status":     "cancelled",
			"end_time":   now,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		target, err := downgradeUserGroupForSubscriptionTx(tx, &sub, now)
		if err != nil {
			return err
		}
		if target != "" {
			cacheGroup = target
			downgradeGroup = target
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if cacheGroup != "" && userId > 0 {
		_ = UpdateUserGroupCache(userId, cacheGroup)
	}
	if downgradeGroup != "" {
		return fmt.Sprintf("用户分组将回退到 %s", downgradeGroup), nil
	}
	return "", nil
}

// AdminDeleteUserSubscription hard-deletes a user subscription.
func AdminDeleteUserSubscription(userSubscriptionId int) (string, error) {
	if userSubscriptionId <= 0 {
		return "", errors.New("invalid userSubscriptionId")
	}
	now := common.GetTimestamp()
	cacheGroup := ""
	downgradeGroup := ""
	var userId int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var sub UserSubscription
		if err := withRowLock(tx).
			Where("id = ?", userSubscriptionId).First(&sub).Error; err != nil {
			return err
		}
		userId = sub.UserId
		target, err := downgradeUserGroupForSubscriptionTx(tx, &sub, now)
		if err != nil {
			return err
		}
		if target != "" {
			cacheGroup = target
			downgradeGroup = target
		}
		if err := tx.Where("id = ?", userSubscriptionId).Delete(&UserSubscription{}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if cacheGroup != "" && userId > 0 {
		_ = UpdateUserGroupCache(userId, cacheGroup)
	}
	if downgradeGroup != "" {
		return fmt.Sprintf("用户分组将回退到 %s", downgradeGroup), nil
	}
	return "", nil
}

type SubscriptionPreConsumeResult struct {
	UserSubscriptionId int
	PreConsumed        int64
	AmountTotal        int64
	AmountUsedBefore   int64
	AmountUsedAfter    int64
}

// ExpireDueSubscriptions marks expired subscriptions and handles group downgrade.
func ExpireDueSubscriptions(limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	now := GetDBTimestamp()
	var subs []UserSubscription
	if err := DB.Where("status = ? AND end_time > 0 AND end_time <= ?", "active", now).
		Order("end_time asc, id asc").
		Limit(limit).
		Find(&subs).Error; err != nil {
		return 0, err
	}
	if len(subs) == 0 {
		return 0, nil
	}
	expiredCount := 0
	userIds := make(map[int]struct{}, len(subs))
	for _, sub := range subs {
		if sub.UserId > 0 {
			userIds[sub.UserId] = struct{}{}
		}
	}
	for userId := range userIds {
		cacheGroup := ""
		err := DB.Transaction(func(tx *gorm.DB) error {
			res := tx.Model(&UserSubscription{}).
				Where("user_id = ? AND status = ? AND end_time > 0 AND end_time <= ?", userId, "active", now).
				Updates(map[string]interface{}{
					"status":     "expired",
					"updated_at": common.GetTimestamp(),
				})
			if res.Error != nil {
				return res.Error
			}
			expiredCount += int(res.RowsAffected)

			// If there's an active upgraded subscription, keep current group.
			var activeSub UserSubscription
			activeQuery := tx.Where("user_id = ? AND status = ? AND start_time <= ? AND end_time > ? AND upgrade_group <> ''",
				userId, "active", now, now).
				Order("end_time desc, id desc").
				Limit(1).
				Find(&activeSub)
			if activeQuery.Error == nil && activeQuery.RowsAffected > 0 {
				return nil
			}

			// No active upgraded subscription, downgrade to explicit target or previous group if needed.
			var lastExpired UserSubscription
			expiredQuery := tx.Where("user_id = ? AND status = ? AND (downgrade_group <> '' OR upgrade_group <> '')",
				userId, "expired").
				Order("end_time desc, id desc").
				Limit(1).
				Find(&lastExpired)
			if expiredQuery.Error != nil || expiredQuery.RowsAffected == 0 {
				return nil
			}
			currentGroup, err := getUserGroupByIdTx(tx, userId)
			if err != nil {
				return err
			}
			target := strings.TrimSpace(lastExpired.DowngradeGroup)
			if target == "" {
				upgradeGroup := strings.TrimSpace(lastExpired.UpgradeGroup)
				prevGroup := strings.TrimSpace(lastExpired.PrevUserGroup)
				if upgradeGroup == "" || prevGroup == "" {
					return nil
				}
				if currentGroup != upgradeGroup {
					return nil
				}
				target = prevGroup
			}
			if target == "" || target == currentGroup {
				return nil
			}
			if err := tx.Model(&User{}).Where("id = ?", userId).
				Update("group", target).Error; err != nil {
				return err
			}
			cacheGroup = target
			return nil
		})
		if err != nil {
			return expiredCount, err
		}
		if cacheGroup != "" {
			_ = UpdateUserGroupCache(userId, cacheGroup)
		}
	}
	return expiredCount, nil
}

// ApplyStartedSubscriptionUpgradeGroups makes queued renewal subscriptions take
// effect once their start_time has arrived.
func ApplyStartedSubscriptionUpgradeGroups(limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	now := GetDBTimestamp()
	var subs []UserSubscription
	if err := DB.Model(&UserSubscription{}).
		Select("user_subscriptions.*").
		Joins("JOIN users ON users.id = user_subscriptions.user_id").
		Where("user_subscriptions.status = ? AND user_subscriptions.start_time <= ? AND user_subscriptions.end_time > ? AND user_subscriptions.upgrade_group <> ''", "active", now, now).
		Where("users."+commonGroupCol+" <> user_subscriptions.upgrade_group").
		Where(`NOT EXISTS (
			SELECT 1 FROM user_subscriptions AS newer
			WHERE newer.user_id = user_subscriptions.user_id
				AND newer.status = ?
				AND newer.start_time <= ?
				AND newer.end_time > ?
				AND newer.upgrade_group <> ''
				AND (
					newer.end_time > user_subscriptions.end_time
					OR (newer.end_time = user_subscriptions.end_time AND newer.id > user_subscriptions.id)
				)
		)`, "active", now, now).
		Order("user_subscriptions.user_id asc, user_subscriptions.end_time desc, user_subscriptions.id desc").
		Limit(limit).
		Find(&subs).Error; err != nil {
		return 0, err
	}
	if len(subs) == 0 {
		return 0, nil
	}
	appliedCount := 0
	for _, sub := range subs {
		if sub.UserId <= 0 {
			continue
		}
		cacheGroup := ""
		subCopy := sub
		err := DB.Transaction(func(tx *gorm.DB) error {
			var selected UserSubscription
			query := withRowLock(tx).Where("user_id = ? AND status = ? AND start_time <= ? AND end_time > ? AND upgrade_group <> ''",
				subCopy.UserId,
				"active",
				now,
				now,
			).
				Order("end_time desc, id desc").
				Limit(1).
				Find(&selected)
			if query.Error != nil {
				return query.Error
			}
			if query.RowsAffected == 0 {
				return nil
			}
			targetGroup := strings.TrimSpace(selected.UpgradeGroup)
			if targetGroup == "" {
				return nil
			}
			currentGroup, err := getUserGroupByIdTx(tx, selected.UserId)
			if err != nil {
				return err
			}
			if currentGroup == targetGroup {
				return nil
			}
			if err := tx.Model(&User{}).Where("id = ?", selected.UserId).
				Update("group", targetGroup).Error; err != nil {
				return err
			}
			cacheGroup = targetGroup
			appliedCount++
			return nil
		})
		if err != nil {
			return appliedCount, err
		}
		if cacheGroup != "" {
			_ = UpdateUserGroupCache(sub.UserId, cacheGroup)
		}
	}
	return appliedCount, nil
}

// SubscriptionPreConsumeRecord stores idempotent pre-consume operations per request.
type SubscriptionPreConsumeRecord struct {
	Id                 int    `json:"id"`
	RequestId          string `json:"request_id" gorm:"type:varchar(64);uniqueIndex"`
	UserId             int    `json:"user_id" gorm:"index"`
	UserSubscriptionId int    `json:"user_subscription_id" gorm:"index"`
	PreConsumed        int64  `json:"pre_consumed" gorm:"type:bigint;not null;default:0"`
	Status             string `json:"status" gorm:"type:varchar(32);index"` // consumed/refunded
	CreatedAt          int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt          int64  `json:"updated_at" gorm:"bigint;index"`
}

func (r *SubscriptionPreConsumeRecord) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	r.CreatedAt = now
	r.UpdatedAt = now
	return nil
}

func (r *SubscriptionPreConsumeRecord) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = common.GetTimestamp()
	return nil
}

func maybeResetUserSubscriptionWithPlanTx(tx *gorm.DB, sub *UserSubscription, plan *SubscriptionPlan, now int64) error {
	if tx == nil || sub == nil || plan == nil {
		return errors.New("invalid reset args")
	}
	if sub.NextResetTime > 0 && sub.NextResetTime > now {
		return nil
	}
	if NormalizeResetPeriod(plan.QuotaResetPeriod) == SubscriptionResetNever {
		return nil
	}
	baseUnix := sub.LastResetTime
	if baseUnix <= 0 {
		baseUnix = sub.StartTime
	}
	base := time.Unix(baseUnix, 0)
	next := calcNextResetTime(base, plan, sub.EndTime)
	advanced := false
	for next > 0 && next <= now {
		advanced = true
		base = time.Unix(next, 0)
		next = calcNextResetTime(base, plan, sub.EndTime)
	}
	if !advanced {
		if sub.NextResetTime == 0 && next > 0 {
			sub.NextResetTime = next
			sub.LastResetTime = base.Unix()
			return tx.Save(sub).Error
		}
		return nil
	}
	sub.AmountUsed = 0
	sub.LastResetTime = base.Unix()
	sub.NextResetTime = next
	return tx.Save(sub).Error
}

// PreConsumeUserSubscription pre-consumes from any active subscription total quota.
func PreConsumeUserSubscription(requestId string, userId int, modelName string, quotaType int, amount int64) (*SubscriptionPreConsumeResult, error) {
	if userId <= 0 {
		return nil, errors.New("invalid userId")
	}
	if strings.TrimSpace(requestId) == "" {
		return nil, errors.New("requestId is empty")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be > 0")
	}
	modelName = strings.TrimSpace(modelName)
	now := GetDBTimestamp()

	returnValue := &SubscriptionPreConsumeResult{}

	err := DB.Transaction(func(tx *gorm.DB) error {
		var existing SubscriptionPreConsumeRecord
		query := tx.Where("request_id = ?", requestId).Limit(1).Find(&existing)
		if query.Error != nil {
			return query.Error
		}
		if query.RowsAffected > 0 {
			if existing.Status == "refunded" {
				return errors.New("subscription pre-consume already refunded")
			}
			var sub UserSubscription
			if err := tx.Where("id = ?", existing.UserSubscriptionId).First(&sub).Error; err != nil {
				return err
			}
			returnValue.UserSubscriptionId = sub.Id
			returnValue.PreConsumed = existing.PreConsumed
			returnValue.AmountTotal = sub.AmountTotal
			returnValue.AmountUsedBefore = sub.AmountUsed
			returnValue.AmountUsedAfter = sub.AmountUsed
			return nil
		}

		var subs []UserSubscription
		if err := withRowLock(tx).
			Where("user_id = ? AND status = ? AND start_time <= ? AND end_time > ?", userId, "active", now, now).
			Order("end_time asc, id asc").
			Find(&subs).Error; err != nil {
			return errors.New("no active subscription")
		}
		if len(subs) == 0 {
			return errors.New("no active subscription")
		}
		anyPlanAllowsModel := false
		for _, candidate := range subs {
			sub := candidate
			plan, err := getSubscriptionPlanByIdTx(tx, sub.PlanId)
			if err != nil {
				return err
			}
			if modelName != "" && !plan.IsModelAllowed(modelName) {
				continue
			}
			if modelName != "" {
				anyPlanAllowsModel = true
			}
			if err := maybeResetUserSubscriptionWithPlanTx(tx, &sub, plan, now); err != nil {
				return err
			}
			usedBefore := sub.AmountUsed
			if sub.AmountTotal > 0 {
				remain := sub.AmountTotal - usedBefore
				if remain < amount {
					continue
				}
			}
			record := &SubscriptionPreConsumeRecord{
				RequestId:          requestId,
				UserId:             userId,
				UserSubscriptionId: sub.Id,
				PreConsumed:        amount,
				Status:             "consumed",
			}
			if err := tx.Create(record).Error; err != nil {
				var dup SubscriptionPreConsumeRecord
				if err2 := tx.Where("request_id = ?", requestId).First(&dup).Error; err2 == nil {
					if dup.Status == "refunded" {
						return errors.New("subscription pre-consume already refunded")
					}
					returnValue.UserSubscriptionId = sub.Id
					returnValue.PreConsumed = dup.PreConsumed
					returnValue.AmountTotal = sub.AmountTotal
					returnValue.AmountUsedBefore = sub.AmountUsed
					returnValue.AmountUsedAfter = sub.AmountUsed
					return nil
				}
				return err
			}
			sub.AmountUsed += amount
			if err := tx.Save(&sub).Error; err != nil {
				return err
			}
			returnValue.UserSubscriptionId = sub.Id
			returnValue.PreConsumed = amount
			returnValue.AmountTotal = sub.AmountTotal
			returnValue.AmountUsedBefore = usedBefore
			returnValue.AmountUsedAfter = sub.AmountUsed
			return nil
		}
		if modelName != "" && !anyPlanAllowsModel {
			return fmt.Errorf("no subscription allows model %s", modelName)
		}
		return fmt.Errorf("subscription quota insufficient, need=%d", amount)
	})
	if err != nil {
		return nil, err
	}
	return returnValue, nil
}

// RefundSubscriptionPreConsume is idempotent and refunds pre-consumed subscription quota by requestId.
func RefundSubscriptionPreConsume(requestId string) error {
	if strings.TrimSpace(requestId) == "" {
		return errors.New("requestId is empty")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var record SubscriptionPreConsumeRecord
		if err := withRowLock(tx).
			Where("request_id = ?", requestId).First(&record).Error; err != nil {
			return err
		}
		if record.Status == "refunded" {
			return nil
		}
		if record.PreConsumed <= 0 {
			record.Status = "refunded"
			return tx.Save(&record).Error
		}
		if err := postConsumeUserSubscriptionDeltaTx(tx, record.UserSubscriptionId, -record.PreConsumed, false); err != nil {
			return err
		}
		record.Status = "refunded"
		return tx.Save(&record).Error
	})
}

// ResetDueSubscriptions resets subscriptions whose next_reset_time has passed.
func ResetDueSubscriptions(limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	now := GetDBTimestamp()
	var subs []UserSubscription
	if err := DB.Where("next_reset_time > 0 AND next_reset_time <= ? AND status = ?", now, "active").
		Order("next_reset_time asc").
		Limit(limit).
		Find(&subs).Error; err != nil {
		return 0, err
	}
	if len(subs) == 0 {
		return 0, nil
	}
	resetCount := 0
	for _, sub := range subs {
		subCopy := sub
		plan, err := getSubscriptionPlanByIdTx(nil, sub.PlanId)
		if err != nil || plan == nil {
			continue
		}
		err = DB.Transaction(func(tx *gorm.DB) error {
			var locked UserSubscription
			if err := withRowLock(tx).
				Where("id = ? AND next_reset_time > 0 AND next_reset_time <= ?", subCopy.Id, now).
				First(&locked).Error; err != nil {
				return nil
			}
			if err := maybeResetUserSubscriptionWithPlanTx(tx, &locked, plan, now); err != nil {
				return err
			}
			resetCount++
			return nil
		})
		if err != nil {
			return resetCount, err
		}
	}
	return resetCount, nil
}

// CleanupSubscriptionPreConsumeRecords removes old idempotency records to keep table small.
func CleanupSubscriptionPreConsumeRecords(olderThanSeconds int64) (int64, error) {
	if olderThanSeconds <= 0 {
		olderThanSeconds = 7 * 24 * 3600
	}
	cutoff := GetDBTimestamp() - olderThanSeconds
	res := DB.Where("updated_at < ?", cutoff).Delete(&SubscriptionPreConsumeRecord{})
	return res.RowsAffected, res.Error
}

type SubscriptionPlanInfo struct {
	PlanId    int    `json:"plan_id"`
	PlanTitle string `json:"plan_title"`
}

func GetSubscriptionPlanInfoByUserSubscriptionId(userSubscriptionId int) (*SubscriptionPlanInfo, error) {
	if userSubscriptionId <= 0 {
		return nil, errors.New("invalid userSubscriptionId")
	}
	cacheKey := fmt.Sprintf("sub:%d", userSubscriptionId)
	if cached, found, err := getSubscriptionPlanInfoCache().Get(cacheKey); err == nil && found {
		return &cached, nil
	}
	var sub UserSubscription
	if err := DB.Where("id = ?", userSubscriptionId).First(&sub).Error; err != nil {
		return nil, err
	}
	plan, err := getSubscriptionPlanByIdTx(nil, sub.PlanId)
	if err != nil {
		return nil, err
	}
	info := &SubscriptionPlanInfo{
		PlanId:    sub.PlanId,
		PlanTitle: plan.Title,
	}
	_ = getSubscriptionPlanInfoCache().SetWithTTL(cacheKey, *info, subscriptionPlanInfoCacheTTL())
	return info, nil
}

// Update subscription used amount by delta (positive consume more, negative refund).
// This is the admission-time/pre-reserve path and rejects positive deltas that
// would exceed the subscription's total quota.
func PostConsumeUserSubscriptionDelta(userSubscriptionId int, delta int64) error {
	if userSubscriptionId <= 0 {
		return errors.New("invalid userSubscriptionId")
	}
	if delta == 0 {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		return postConsumeUserSubscriptionDeltaTx(tx, userSubscriptionId, delta, false)
	})
}

// SettleUserSubscriptionDelta records the final delta for a request/task that
// has already succeeded. Positive settlement deltas may exceed amount_total:
// the request was admitted by pre-consume, so the final usage must be recorded
// fully and future admission will see no remaining quota.
func SettleUserSubscriptionDelta(userSubscriptionId int, delta int64) error {
	if userSubscriptionId <= 0 {
		return errors.New("invalid userSubscriptionId")
	}
	if delta == 0 {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		return postConsumeUserSubscriptionDeltaTx(tx, userSubscriptionId, delta, true)
	})
}

func postConsumeUserSubscriptionDeltaTx(tx *gorm.DB, userSubscriptionId int, delta int64, allowOverLimit bool) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if userSubscriptionId <= 0 {
		return errors.New("invalid userSubscriptionId")
	}
	if delta == 0 {
		return nil
	}
	var sub UserSubscription
	if err := withRowLock(tx).
		Where("id = ?", userSubscriptionId).
		First(&sub).Error; err != nil {
		return err
	}
	newUsed := sub.AmountUsed + delta
	if newUsed < 0 {
		newUsed = 0
	}
	if !allowOverLimit && sub.AmountTotal > 0 && newUsed > sub.AmountTotal {
		return fmt.Errorf("subscription used exceeds total, used=%d total=%d", newUsed, sub.AmountTotal)
	}
	sub.AmountUsed = newUsed
	return tx.Save(&sub).Error
}

func lockSubscriptionPurchaseUserTx(tx *gorm.DB, userId int) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if userId <= 0 {
		return errors.New("invalid userId")
	}
	var user User
	return withRowLock(tx).
		Select("id").
		Where("id = ?", userId).
		First(&user).Error
}
