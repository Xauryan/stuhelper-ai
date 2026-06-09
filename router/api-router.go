package router

import (
	"github.com/Xauryan/stuhelper-ai/controller"
	"github.com/Xauryan/stuhelper-ai/middleware"

	// Import oauth package to register providers via init()
	_ "github.com/Xauryan/stuhelper-ai/oauth"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetApiRouter(router *gin.Engine) {
	apiRouter := router.Group("/api")
	apiRouter.Use(middleware.RouteTag("api"))
	apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	apiRouter.Use(middleware.BodyStorageCleanup()) // 清理请求体存储
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	anonymousRequestBodyLimit := middleware.AnonymousRequestBodyLimit()
	{
		apiRouter.GET("/setup", controller.GetSetup)
		apiRouter.POST("/setup", anonymousRequestBodyLimit, controller.PostSetup)
		apiRouter.GET("/status", controller.GetStatus)
		apiRouter.GET("/uptime/status", controller.GetUptimeKumaStatus)
		apiRouter.GET("/models", middleware.UserAuth(), controller.DashboardListModels)
		apiRouter.GET("/status/test", middleware.AdminAuth(), controller.TestStatus)
		apiRouter.GET("/notice", controller.GetNotice)
		apiRouter.GET("/user-agreement", controller.GetUserAgreement)
		apiRouter.GET("/privacy-policy", controller.GetPrivacyPolicy)
		apiRouter.GET("/about", controller.GetAbout)
		//apiRouter.GET("/midjourney", controller.GetMidjourney)
		apiRouter.GET("/home_page_content", controller.GetHomePageContent)
		apiRouter.GET("/pricing", middleware.HeaderNavModuleAuth("pricing"), controller.GetPricing)
		perfMetricsRoute := apiRouter.Group("/perf-metrics")
		perfMetricsRoute.Use(middleware.HeaderNavModulePublicOrUserAuth("pricing"))
		{
			perfMetricsRoute.GET("/summary", controller.GetPerfMetricsSummary)
			perfMetricsRoute.GET("", controller.GetPerfMetrics)
		}
		apiRouter.GET("/rankings", middleware.TryUserAuth(), controller.GetRankings)
		apiRouter.GET("/rankings/users", middleware.TryUserAuth(), controller.GetUserRankings)
		apiRouter.GET("/verification", middleware.EmailVerificationRateLimit(), middleware.TurnstileCheck(), controller.SendEmailVerification)
		apiRouter.GET("/reset_password", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.SendPasswordResetEmail)
		apiRouter.POST("/user/reset", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.ResetPassword)
		// OAuth routes - specific routes must come before :provider wildcard
		apiRouter.GET("/oauth/state", middleware.CriticalRateLimit(), controller.GenerateOAuthCode)
		apiRouter.POST("/oauth/email/bind", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.EmailBind)
		// Non-standard OAuth (WeChat, Telegram) - keep original routes
		apiRouter.GET("/oauth/wechat", middleware.CriticalRateLimit(), controller.WeChatAuth)
		apiRouter.POST("/oauth/wechat/bind", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.WeChatBind)
		apiRouter.GET("/oauth/telegram/login", middleware.CriticalRateLimit(), controller.TelegramLogin)
		apiRouter.GET("/oauth/telegram/bind", middleware.CriticalRateLimit(), controller.TelegramBind)
		// Standard OAuth providers (GitHub, Discord, OIDC, LinuxDO) - unified route
		apiRouter.GET("/oauth/:provider", middleware.CriticalRateLimit(), controller.HandleOAuth)
		apiRouter.GET("/ratio_config", middleware.CriticalRateLimit(), controller.GetRatioConfig)

		apiRouter.POST("/stripe/webhook", anonymousRequestBodyLimit, controller.StripeWebhook)
		apiRouter.POST("/creem/webhook", anonymousRequestBodyLimit, controller.CreemWebhook)
		apiRouter.POST("/waffo/webhook", anonymousRequestBodyLimit, controller.WaffoWebhook)
		apiRouter.POST("/alipay/official/notify", anonymousRequestBodyLimit, controller.AlipayOfficialNotify)
		apiRouter.POST("/wechat-pay/official/notify", anonymousRequestBodyLimit, controller.WechatPayOfficialNotify)

		// Universal secure verification routes
		apiRouter.POST("/verify", middleware.UserAuth(), middleware.CriticalRateLimit(), controller.UniversalVerify)

		userRoute := apiRouter.Group("/user")
		{
			userRoute.POST("/register", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, middleware.TurnstileCheck(), controller.Register)
			userRoute.POST("/login", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, middleware.TurnstileCheck(), controller.Login)
			userRoute.POST("/login/2fa", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.Verify2FALogin)
			userRoute.POST("/passkey/login/begin", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.PasskeyLoginBegin)
			userRoute.POST("/passkey/login/finish", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.PasskeyLoginFinish)
			//userRoute.POST("/tokenlog", middleware.CriticalRateLimit(), controller.TokenLog)
			userRoute.GET("/logout", controller.Logout)
			userRoute.POST("/epay/notify", anonymousRequestBodyLimit, controller.EpayNotify)
			userRoute.GET("/epay/notify", controller.EpayNotify)
			userRoute.GET("/groups", controller.GetUserGroups)

			selfRoute := userRoute.Group("/")
			selfRoute.Use(middleware.UserAuth())
			{
				selfRoute.GET("/self/groups", controller.GetUserGroups)
				selfRoute.GET("/self", controller.GetSelf)
				selfRoute.GET("/models", controller.GetUserModels)
				selfRoute.PUT("/self", controller.UpdateSelf)
				selfRoute.DELETE("/self", controller.DeleteSelf)
				selfRoute.GET("/token", controller.GenerateAccessToken)
				selfRoute.GET("/passkey", controller.PasskeyStatus)
				selfRoute.POST("/passkey/register/begin", controller.PasskeyRegisterBegin)
				selfRoute.POST("/passkey/register/finish", controller.PasskeyRegisterFinish)
				selfRoute.POST("/passkey/verify/begin", controller.PasskeyVerifyBegin)
				selfRoute.POST("/passkey/verify/finish", controller.PasskeyVerifyFinish)
				selfRoute.DELETE("/passkey", controller.PasskeyDelete)
				selfRoute.GET("/aff", controller.GetAffCode)
				selfRoute.GET("/aff/commissions", controller.GetReferralCommissions)
				selfRoute.GET("/topup/info", controller.GetTopUpInfo)
				selfRoute.GET("/topup/self", controller.GetUserTopUps)
				selfRoute.POST("/topup", middleware.CriticalRateLimit(), controller.TopUp)
				selfRoute.POST("/pay", middleware.CriticalRateLimit(), controller.RequestEpay)
				selfRoute.POST("/amount", controller.RequestAmount)
				selfRoute.POST("/stripe/pay", middleware.CriticalRateLimit(), controller.RequestStripePay)
				selfRoute.POST("/stripe/amount", controller.RequestStripeAmount)
				selfRoute.POST("/creem/pay", middleware.CriticalRateLimit(), controller.RequestCreemPay)
				selfRoute.POST("/waffo/amount", controller.RequestWaffoAmount)
				selfRoute.POST("/waffo/pay", middleware.CriticalRateLimit(), controller.RequestWaffoPay)
				selfRoute.POST("/alipay/official/amount", controller.RequestAlipayOfficialAmount)
				selfRoute.POST("/alipay/official/pay", middleware.CriticalRateLimit(), controller.RequestAlipayOfficialPay)
				selfRoute.POST("/wechat-pay/official/amount", controller.RequestWechatPayOfficialAmount)
				selfRoute.POST("/wechat-pay/official/pay", middleware.CriticalRateLimit(), controller.RequestWechatPayOfficialPay)
				selfRoute.POST("/wechat-pay/official/status", controller.QueryWechatPayOfficialTopUpStatus)
				selfRoute.POST("/self-serve/preview", controller.RequestSelfServeTopUpPreview)
				selfRoute.POST("/self-serve/pay", middleware.CriticalRateLimit(), controller.RequestSelfServeTopUp)
				selfRoute.POST("/topup/official/refund/preview", controller.GetOfficialPaymentRefundPreview)
				selfRoute.POST("/topup/official/refund/apply", middleware.CriticalRateLimit(), controller.ApplyOfficialPaymentRefund)
				selfRoute.POST("/aff_transfer", controller.TransferAffQuota)
				selfRoute.PUT("/setting", controller.UpdateUserSetting)

				// 2FA routes
				selfRoute.GET("/2fa/status", controller.Get2FAStatus)
				selfRoute.POST("/2fa/setup", controller.Setup2FA)
				selfRoute.POST("/2fa/enable", controller.Enable2FA)
				selfRoute.POST("/2fa/disable", controller.Disable2FA)
				selfRoute.POST("/2fa/backup_codes", controller.RegenerateBackupCodes)

				// Check-in routes
				selfRoute.GET("/checkin", controller.GetCheckinStatus)
				selfRoute.POST("/checkin", middleware.TurnstileCheck(), controller.DoCheckin)

				// Custom OAuth bindings
				selfRoute.GET("/oauth/bindings", controller.GetUserOAuthBindings)
				selfRoute.DELETE("/oauth/bindings/:provider_id", controller.UnbindCustomOAuth)
			}

			adminRoute := userRoute.Group("/")
			adminRoute.Use(middleware.AuditAdminAuth())
			{
				adminRoute.GET("/", controller.GetAllUsers)
				adminRoute.GET("/search", controller.SearchUsers)
				adminRoute.GET("/referrals", middleware.RequireAuditOrAdminRole(), controller.GetAdminReferralRecords)
				adminRoute.GET("/referrals/:invitee_id/commissions", middleware.RequireAuditOrAdminRole(), controller.GetAdminReferralCommissions)
				adminRoute.GET("/topup", middleware.RequireAdminRole(), controller.GetAllTopUps)
				adminRoute.POST("/topup/complete", middleware.RequireAdminRole(), controller.AdminCompleteTopUp)
				adminRoute.POST("/topup/admin/update", middleware.RequireAdminRole(), controller.AdminUpdateAdminTopUp)
				adminRoute.POST("/topup/admin/refund", middleware.RequireAdminRole(), controller.AdminRefundAdminTopUp)
				adminRoute.POST("/topup/alipay-official/refund", middleware.RequireAdminRole(), controller.AdminRefundAlipayOfficialTopUp)
				adminRoute.POST("/topup/alipay-official/query", middleware.RequireAdminRole(), controller.AdminQueryAlipayOfficialTopUp)
				adminRoute.POST("/topup/alipay-official/close", middleware.RequireAdminRole(), controller.AdminCloseAlipayOfficialTopUp)
				adminRoute.POST("/topup/wechat-pay-official/refund", middleware.RequireAdminRole(), controller.AdminRefundWechatPayOfficialTopUp)
				adminRoute.POST("/topup/wechat-pay-official/query", middleware.RequireAdminRole(), controller.AdminQueryWechatPayOfficialTopUp)
				adminRoute.POST("/topup/wechat-pay-official/close", middleware.RequireAdminRole(), controller.AdminCloseWechatPayOfficialTopUp)
				adminRoute.POST("/topup/wechat-pay-official/refund-query", middleware.RequireAdminRole(), controller.AdminQueryWechatPayOfficialRefund)
				adminRoute.POST("/topup/official/refund-request/approve", middleware.RequireAdminRole(), controller.AdminApproveOfficialPaymentRefundRequest)
				adminRoute.POST("/topup/official/refund-request/reject", middleware.RequireAdminRole(), controller.AdminRejectOfficialPaymentRefundRequest)
				adminRoute.POST("/topup/self-serve/approve", middleware.RequireAdminRole(), controller.AdminApproveSelfServeTopUp)
				adminRoute.POST("/topup/self-serve/update", middleware.RequireAdminRole(), controller.AdminUpdateSelfServeTopUp)
				adminRoute.POST("/topup/self-serve/reject", middleware.RequireAdminRole(), controller.AdminRejectSelfServeTopUp)
				adminRoute.GET("/2fa/stats", middleware.RequireAdminRole(), controller.Admin2FAStats)
				adminRoute.GET("/:id/oauth/bindings", middleware.RequireAdminRole(), controller.GetUserOAuthBindingsByAdmin)
				adminRoute.DELETE("/:id/oauth/bindings/:provider_id", middleware.RequireAdminRole(), controller.UnbindCustomOAuthByAdmin)
				adminRoute.DELETE("/:id/bindings/:binding_type", middleware.RequireAdminRole(), controller.AdminClearUserBinding)
				adminRoute.GET("/:id", middleware.RequireAdminRole(), controller.GetUser)
				adminRoute.POST("/", middleware.RequireAdminRole(), controller.CreateUser)
				adminRoute.POST("/manage", middleware.RequireAdminRole(), controller.ManageUser)
				adminRoute.PUT("/", middleware.RequireAdminRole(), controller.UpdateUser)
				adminRoute.DELETE("/:id", middleware.RequireAdminRole(), controller.DeleteUser)
				adminRoute.DELETE("/:id/reset_passkey", middleware.RequireAdminRole(), controller.AdminResetPasskey)
				adminRoute.DELETE("/:id/2fa", middleware.RequireAdminRole(), controller.AdminDisable2FA)
			}
		}

		// Subscription billing (plans, purchase, admin management)
		subscriptionRoute := apiRouter.Group("/subscription")
		subscriptionRoute.Use(middleware.UserAuth())
		{
			subscriptionRoute.GET("/plans", controller.GetSubscriptionPlans)
			subscriptionRoute.GET("/self", controller.GetSubscriptionSelf)
			subscriptionRoute.PUT("/self/preference", controller.UpdateSubscriptionPreference)
			subscriptionRoute.POST("/balance/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestBalancePay)
			subscriptionRoute.POST("/epay/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestEpay)
			subscriptionRoute.POST("/stripe/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestStripePay)
			subscriptionRoute.POST("/creem/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestCreemPay)
			subscriptionRoute.POST("/alipay-official/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestAlipayOfficialPay)
			subscriptionRoute.POST("/wechat-pay-official/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestWechatPayOfficialPay)
		}
		subscriptionAdminRoute := apiRouter.Group("/subscription/admin")
		subscriptionAdminRoute.Use(middleware.AuditAdminAuth())
		{
			subscriptionAdminRoute.GET("/plans", controller.AdminListSubscriptionPlans)
			subscriptionAdminRoute.POST("/plans", middleware.RequireAdminRole(), controller.AdminCreateSubscriptionPlan)
			subscriptionAdminRoute.PUT("/plans/:id", middleware.RequireAdminRole(), controller.AdminUpdateSubscriptionPlan)
			subscriptionAdminRoute.PATCH("/plans/:id", middleware.RequireAdminRole(), controller.AdminUpdateSubscriptionPlanStatus)
			subscriptionAdminRoute.POST("/bind", middleware.RequireAdminRole(), controller.AdminBindSubscription)

			// User subscription management (admin)
			subscriptionAdminRoute.GET("/users/:id/subscriptions", middleware.RequireAdminRole(), controller.AdminListUserSubscriptions)
			subscriptionAdminRoute.POST("/users/:id/subscriptions", middleware.RequireAdminRole(), controller.AdminCreateUserSubscription)
			subscriptionAdminRoute.POST("/user_subscriptions/:id/invalidate", middleware.RequireAdminRole(), controller.AdminInvalidateUserSubscription)
			subscriptionAdminRoute.DELETE("/user_subscriptions/:id", middleware.RequireAdminRole(), controller.AdminDeleteUserSubscription)
		}

		// Subscription payment callbacks (no auth)
		apiRouter.POST("/subscription/epay/notify", anonymousRequestBodyLimit, controller.SubscriptionEpayNotify)
		apiRouter.GET("/subscription/epay/notify", controller.SubscriptionEpayNotify)
		apiRouter.GET("/subscription/epay/return", controller.SubscriptionEpayReturn)
		apiRouter.POST("/subscription/epay/return", anonymousRequestBodyLimit, controller.SubscriptionEpayReturn)
		optionRoute := apiRouter.Group("/option")
		optionRoute.Use(middleware.RootAuth())
		{
			optionRoute.GET("/", controller.GetOptions)
			optionRoute.PUT("/", controller.UpdateOption)
			optionRoute.GET("/channel_affinity_cache", controller.GetChannelAffinityCacheStats)
			optionRoute.DELETE("/channel_affinity_cache", controller.ClearChannelAffinityCache)
			optionRoute.POST("/rest_model_ratio", controller.ResetModelRatio)
			optionRoute.POST("/migrate_console_setting", controller.MigrateConsoleSetting) // 用于迁移检测的旧键，下个版本会删除
		}

		// Custom OAuth provider management (root only)
		customOAuthRoute := apiRouter.Group("/custom-oauth-provider")
		customOAuthRoute.Use(middleware.RootAuth())
		{
			customOAuthRoute.POST("/discovery", controller.FetchCustomOAuthDiscovery)
			customOAuthRoute.GET("/", controller.GetCustomOAuthProviders)
			customOAuthRoute.GET("/:id", controller.GetCustomOAuthProvider)
			customOAuthRoute.POST("/", controller.CreateCustomOAuthProvider)
			customOAuthRoute.PUT("/:id", controller.UpdateCustomOAuthProvider)
			customOAuthRoute.DELETE("/:id", controller.DeleteCustomOAuthProvider)
		}
		performanceRoute := apiRouter.Group("/performance")
		performanceRoute.Use(middleware.RootAuth())
		{
			performanceRoute.GET("/stats", controller.GetPerformanceStats)
			performanceRoute.DELETE("/disk_cache", controller.ClearDiskCache)
			performanceRoute.POST("/reset_stats", controller.ResetPerformanceStats)
			performanceRoute.POST("/gc", controller.ForceGC)
			performanceRoute.GET("/logs", controller.GetLogFiles)
			performanceRoute.DELETE("/logs", controller.CleanupLogFiles)
		}
		ratioSyncRoute := apiRouter.Group("/ratio_sync")
		ratioSyncRoute.Use(middleware.RootAuth())
		{
			ratioSyncRoute.GET("/channels", controller.GetSyncableChannels)
			ratioSyncRoute.POST("/fetch", controller.FetchUpstreamRatios)
		}
		channelRoute := apiRouter.Group("/channel")
		channelRoute.Use(middleware.AuditAdminAuth())
		{
			channelRoute.GET("/", controller.GetAllChannels)
			channelRoute.GET("/search", controller.SearchChannels)
			channelRoute.GET("/models", controller.ChannelListModels)
			channelRoute.GET("/models_enabled", controller.EnabledListModels)
			channelRoute.GET("/test", middleware.RequireAdminRole(), controller.TestAllChannels)
			channelRoute.GET("/test/:id", middleware.RequireAdminRole(), controller.TestChannel)
			channelRoute.GET("/update_balance", middleware.RequireAdminRole(), controller.UpdateAllChannelsBalance)
			channelRoute.GET("/update_balance/:id", middleware.RequireAdminRole(), controller.UpdateChannelBalance)
			channelRoute.POST("/", middleware.RequireAdminRole(), controller.AddChannel)
			channelRoute.PUT("/", middleware.RequireAdminRole(), controller.UpdateChannel)
			channelRoute.DELETE("/disabled", middleware.RequireAdminRole(), controller.DeleteDisabledChannel)
			channelRoute.POST("/tag/disabled", middleware.RequireAdminRole(), controller.DisableTagChannels)
			channelRoute.POST("/tag/enabled", middleware.RequireAdminRole(), controller.EnableTagChannels)
			channelRoute.PUT("/tag", middleware.RequireAdminRole(), controller.EditTagChannels)
			channelRoute.POST("/batch", middleware.RequireAdminRole(), controller.DeleteChannelBatch)
			channelRoute.POST("/fix", middleware.RequireAdminRole(), controller.FixChannelsAbilities)
			channelRoute.GET("/fetch_models/:id", middleware.RequireAdminRole(), controller.FetchUpstreamModels)
			channelRoute.POST("/fetch_models", middleware.RootAuth(), controller.FetchModels)
			channelRoute.POST("/codex/oauth/start", middleware.RequireAdminRole(), controller.StartCodexOAuth)
			channelRoute.POST("/codex/oauth/complete", middleware.RequireAdminRole(), controller.CompleteCodexOAuth)
			channelRoute.POST("/ollama/pull", middleware.RequireAdminRole(), controller.OllamaPullModel)
			channelRoute.POST("/ollama/pull/stream", middleware.RequireAdminRole(), controller.OllamaPullModelStream)
			channelRoute.DELETE("/ollama/delete", middleware.RequireAdminRole(), controller.OllamaDeleteModel)
			channelRoute.GET("/ollama/version/:id", middleware.RequireAdminRole(), controller.OllamaVersion)
			channelRoute.POST("/batch/tag", middleware.RequireAdminRole(), controller.BatchSetChannelTag)
			channelRoute.GET("/tag/models", middleware.RequireAdminRole(), controller.GetTagModels)
			channelRoute.POST("/copy/:id", middleware.RequireAdminRole(), controller.CopyChannel)
			channelRoute.POST("/multi_key/manage", middleware.RequireAdminRole(), controller.ManageMultiKeys)
			channelRoute.POST("/upstream_updates/apply", middleware.RequireAdminRole(), controller.ApplyChannelUpstreamModelUpdates)
			channelRoute.POST("/upstream_updates/apply_all", middleware.RequireAdminRole(), controller.ApplyAllChannelUpstreamModelUpdates)
			channelRoute.POST("/upstream_updates/detect", middleware.RequireAdminRole(), controller.DetectChannelUpstreamModelUpdates)
			channelRoute.POST("/upstream_updates/detect_all", middleware.RequireAdminRole(), controller.DetectAllChannelUpstreamModelUpdates)
			channelRoute.GET("/:id", middleware.RequireAdminRole(), controller.GetChannel)
			channelRoute.DELETE("/:id", middleware.RequireAdminRole(), controller.DeleteChannel)
			channelRoute.POST("/:id/key", middleware.RootAuth(), middleware.CriticalRateLimit(), middleware.DisableCache(), middleware.SecureVerificationRequired(), controller.GetChannelKey)
			channelRoute.POST("/:id/codex/oauth/start", middleware.RequireAdminRole(), controller.StartCodexOAuthForChannel)
			channelRoute.POST("/:id/codex/oauth/complete", middleware.RequireAdminRole(), controller.CompleteCodexOAuthForChannel)
			channelRoute.POST("/:id/codex/refresh", middleware.RequireAdminRole(), controller.RefreshCodexChannelCredential)
			channelRoute.GET("/:id/codex/usage", middleware.RequireAdminRole(), controller.GetCodexChannelUsage)
		}
		tokenRoute := apiRouter.Group("/token")
		tokenRoute.Use(middleware.UserAuth())
		{
			tokenRoute.GET("/", controller.GetAllTokens)
			tokenRoute.GET("/search", middleware.SearchRateLimit(), controller.SearchTokens)
			tokenRoute.GET("/:id", controller.GetToken)
			tokenRoute.POST("/:id/key", middleware.CriticalRateLimit(), middleware.DisableCache(), controller.GetTokenKey)
			tokenRoute.POST("/", controller.AddToken)
			tokenRoute.PUT("/", controller.UpdateToken)
			tokenRoute.DELETE("/:id", controller.DeleteToken)
			tokenRoute.POST("/batch", controller.DeleteTokenBatch)
			tokenRoute.POST("/batch/keys", middleware.CriticalRateLimit(), middleware.DisableCache(), controller.GetTokenKeysBatch)
		}

		usageRoute := apiRouter.Group("/usage")
		usageRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			tokenUsageRoute := usageRoute.Group("/token")
			tokenUsageRoute.Use(middleware.TokenAuthReadOnly())
			{
				tokenUsageRoute.GET("/", controller.GetTokenUsage)
			}
		}

		redemptionRoute := apiRouter.Group("/redemption")
		redemptionRoute.Use(middleware.AuditAdminAuth())
		{
			redemptionRoute.GET("/", controller.GetAllRedemptions)
			redemptionRoute.GET("/search", controller.SearchRedemptions)
			redemptionRoute.GET("/:id", middleware.RequireAdminRole(), controller.GetRedemption)
			redemptionRoute.POST("/", middleware.RequireAdminRole(), controller.AddRedemption)
			redemptionRoute.PUT("/", middleware.RequireAdminRole(), controller.UpdateRedemption)
			redemptionRoute.DELETE("/invalid", middleware.RequireAdminRole(), controller.DeleteInvalidRedemption)
			redemptionRoute.DELETE("/:id", middleware.RequireAdminRole(), controller.DeleteRedemption)
		}
		logRoute := apiRouter.Group("/log")
		logRoute.GET("/", middleware.AuditAdminAuth(), controller.GetAllLogs)
		logRoute.DELETE("/", middleware.AdminAuth(), controller.DeleteHistoryLogs)
		logRoute.GET("/stat", middleware.AuditAdminAuth(), controller.GetLogsStat)
		logRoute.GET("/self/stat", middleware.UserAuth(), controller.GetLogsSelfStat)
		logRoute.GET("/channel_affinity_usage_cache", middleware.AuditAdminAuth(), controller.GetChannelAffinityUsageCacheStats)
		logRoute.GET("/search", middleware.AuditAdminAuth(), controller.SearchAllLogs)
		logRoute.GET("/self", middleware.UserAuth(), controller.GetUserLogs)
		logRoute.GET("/self/search", middleware.UserAuth(), middleware.SearchRateLimit(), controller.SearchUserLogs)

		dataRoute := apiRouter.Group("/data")
		dataRoute.GET("/", middleware.AdminAuth(), controller.GetAllQuotaDates)
		dataRoute.GET("/users", middleware.AdminAuth(), controller.GetQuotaDatesByUser)
		dataRoute.GET("/self", middleware.UserAuth(), controller.GetUserQuotaDates)

		logRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			logRoute.GET("/token", middleware.TokenAuthReadOnly(), controller.GetLogByKey)
		}
		groupRoute := apiRouter.Group("/group")
		groupRoute.Use(middleware.AuditAdminAuth())
		{
			groupRoute.GET("/", controller.GetGroups)
		}

		prefillGroupRoute := apiRouter.Group("/prefill_group")
		prefillGroupRoute.Use(middleware.AuditAdminAuth())
		{
			prefillGroupRoute.GET("/", controller.GetPrefillGroups)
			prefillGroupRoute.POST("/", middleware.RequireAdminRole(), controller.CreatePrefillGroup)
			prefillGroupRoute.PUT("/", middleware.RequireAdminRole(), controller.UpdatePrefillGroup)
			prefillGroupRoute.DELETE("/:id", middleware.RequireAdminRole(), controller.DeletePrefillGroup)
		}

		mjRoute := apiRouter.Group("/mj")
		mjRoute.GET("/self", middleware.UserAuth(), controller.GetUserMidjourney)
		mjRoute.GET("/", middleware.AuditAdminAuth(), controller.GetAllMidjourney)

		taskRoute := apiRouter.Group("/task")
		{
			taskRoute.GET("/self", middleware.UserAuth(), controller.GetUserTask)
			taskRoute.GET("/", middleware.AuditAdminAuth(), controller.GetAllTask)
		}

		vendorRoute := apiRouter.Group("/vendors")
		vendorRoute.Use(middleware.AuditAdminAuth())
		{
			vendorRoute.GET("/", controller.GetAllVendors)
			vendorRoute.GET("/search", controller.SearchVendors)
			vendorRoute.GET("/:id", middleware.RequireAdminRole(), controller.GetVendorMeta)
			vendorRoute.POST("/", middleware.RequireAdminRole(), controller.CreateVendorMeta)
			vendorRoute.PUT("/", middleware.RequireAdminRole(), controller.UpdateVendorMeta)
			vendorRoute.DELETE("/:id", middleware.RequireAdminRole(), controller.DeleteVendorMeta)
		}

		modelsRoute := apiRouter.Group("/models")
		modelsRoute.Use(middleware.AuditAdminAuth())
		{
			modelsRoute.GET("/sync_upstream/preview", middleware.RequireAdminRole(), controller.SyncUpstreamPreview)
			modelsRoute.POST("/sync_upstream", middleware.RequireAdminRole(), controller.SyncUpstreamModels)
			modelsRoute.GET("/missing", middleware.RequireAdminRole(), controller.GetMissingModels)
			modelsRoute.GET("/", controller.GetAllModelsMeta)
			modelsRoute.GET("/search", controller.SearchModelsMeta)
			modelsRoute.GET("/:id", middleware.RequireAdminRole(), controller.GetModelMeta)
			modelsRoute.POST("/", middleware.RequireAdminRole(), controller.CreateModelMeta)
			modelsRoute.PUT("/", middleware.RequireAdminRole(), controller.UpdateModelMeta)
			modelsRoute.DELETE("/:id", middleware.RequireAdminRole(), controller.DeleteModelMeta)
		}

	}
}
