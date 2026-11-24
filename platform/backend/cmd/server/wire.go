//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/pkg/backgroundgate"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/server"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

type Application struct {
	Server                *http.Server
	PromptAudit           *securityaudit.PromptService
	PluginManager         *service.PluginManager
	UsageRecordWorkerPool *service.UsageRecordWorkerPool
	BalanceNotify         *service.BalanceNotifyService
	Cleanup               func()
}

func initializeApplication(buildInfo handler.BuildInfo) (*Application, error) {
	wire.Build(
		// Infrastructure layer ProviderSets
		config.ProviderSet,

		// Business layer ProviderSets
		repository.ProviderSet,
		service.ProviderSet,
		securityaudit.ProviderSet,
		payment.ProviderSet,
		middleware.ProviderSet,
		handler.ProviderSet,

		// Server layer ProviderSet
		server.ProviderSet,

		// Privacy client factory for OpenAI training opt-out
		providePrivacyClientFactory,

		// BuildInfo provider
		provideServiceBuildInfo,
		providePluginHostInfo,

		// Cleanup function provider
		provideCleanup,

		// Application struct
		wire.Struct(new(Application), "Server", "PromptAudit", "PluginManager", "UsageRecordWorkerPool", "BalanceNotify", "Cleanup"),
	)
	return nil, nil
}

func providePrivacyClientFactory() service.PrivacyClientFactory {
	return repository.CreatePrivacyReqClient
}

func provideServiceBuildInfo(buildInfo handler.BuildInfo) service.BuildInfo {
	return service.BuildInfo{
		Version:   buildInfo.Version,
		BuildType: buildInfo.BuildType,
	}
}

func providePluginHostInfo(buildInfo handler.BuildInfo) service.PluginHostInfo {
	return service.PluginHostInfo{
		Version:   buildInfo.Version,
		BuildType: buildInfo.BuildType,
	}
}

func provideCleanup(
	entClient *ent.Client,
	rdb *redis.Client,
	opsMetricsCollector *service.OpsMetricsCollector,
	opsAggregation *service.OpsAggregationService,
	opsAlertEvaluator *service.OpsAlertEvaluatorService,
	opsCleanup *service.OpsCleanupService,
	opsScheduledReport *service.OpsScheduledReportService,
	opsSystemLogSink *service.OpsSystemLogSink,
	opsService *service.OpsService,
	opsIngressReject *service.OpsIngressRejectAggregator,
	apiKeyService *service.APIKeyService,
	authCacheInvalidationWorker *service.AuthCacheInvalidationWorker,
	schedulerSnapshot *service.SchedulerSnapshotService,
	tokenRefresh *service.TokenRefreshService,
	accountExpiry *service.AccountExpiryService,
	cnProviderBalanceCheck *service.CNProviderBalanceCheckService,
	codexVersionSync *service.OpenAICodexVersionSyncService,
	claudeCodeVersionSync *service.ClaudeCodeVersionSyncService,
	proxyExpiry *service.ProxyExpiryService,
	subscriptionExpiry *service.SubscriptionExpiryService,
	usageCleanup *service.UsageCleanupService,
	idempotencyCleanup *service.IdempotencyCleanupService,
	geminiCachedContent *service.GeminiCachedContentService,
	batchImageCleanup *service.BatchImageCleanupService,
	batchImageWorker *service.BatchImageWorkerRuntime,
	pricing *service.PricingService,
	arenaLeaderboard *service.ArenaLeaderboardService,
	emailQueue *service.EmailQueueService,
	billingCache *service.BillingCacheService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	usageBillingLanes *service.UsageBillingLaneCombiner,
	balanceNotify *service.BalanceNotifyService,
	digestSessions *service.DigestSessionStore,
	subscriptionService *service.SubscriptionService,
	oauth *service.OAuthService,
	openaiOAuth *service.OpenAIOAuthService,
	geminiOAuth *service.GeminiOAuthService,
	antigravityOAuth *service.AntigravityOAuthService,
	grokOAuth *service.GrokOAuthService,
	openAIGateway *service.OpenAIGatewayService,
	scheduledTestRunner *service.ScheduledTestRunnerService,
	backupSvc *service.BackupService,
	paymentOrderExpiry *service.PaymentOrderExpiryService,
	channelMonitorRunner *service.ChannelMonitorRunner,
	channelMonitorV2Aggregator *service.ChannelMonitorV2Aggregator,
	quotaFlusher *service.UserPlatformQuotaUsageFlusher,
	upstreamBillingProbe *service.UpstreamBillingProbeService,
	ollamaCloudUsage *service.OllamaCloudUsageService,
	opencodeGoUsage *service.OpenCodeGoUsageService,
	auditLog *service.AuditLogService,
	openAIAutoReset *service.OpenAIQuotaAutoResetService,
	promptAudit *securityaudit.PromptService,
	pluginManager *service.PluginManager,
	timingWheel *service.TimingWheelService,
	deferred *service.DeferredService,
	userMessageQueue *service.UserMessageQueueService,
) func() {
	return func() {
		backgroundgate.Close() // no future shared worker may start during drain
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		type cleanupStep struct {
			name string
			fn   func() error
		}

		if usageRecordWorkerPool != nil {
			usageRecordWorkerPool.Stop()
		}
		usageBillingLanes.Stop()
		backgroundgate.StopManaged()
		// Usage tasks have drained; dependent request queues may now be flushed/stopped.
		serviceSteps := []cleanupStep{
			{"DigestSessionStore", func() error {
				if digestSessions != nil {
					digestSessions.Close()
				}
				return nil
			}},
			{"BalanceNotifyService", func() error {
				if balanceNotify != nil {
					balanceNotify.Stop()
				}
				return nil
			}},
			{"PluginManager", func() error {
				backgroundgate.Stop("PluginManager", func() {
					if pluginManager != nil {
						pluginManager.Stop()
					}
				})
				return nil
			}},
			{"OpenAIQuotaAutoResetService", func() error {
				backgroundgate.Stop("OpenAIQuotaAutoResetService", func() {
					if openAIAutoReset != nil {
						openAIAutoReset.Stop()
					}
				})
				return nil
			}},
			{"OpsIngressRejectAggregator", func() error {
				if opsIngressReject != nil {
					opsIngressReject.Stop()
				}
				return nil
			}},
			{"AuthCacheInvalidationWorker", func() error {
				if authCacheInvalidationWorker != nil {
					authCacheInvalidationWorker.Stop()
				}
				return nil
			}},
			{"AuthCacheInvalidationSubscriber", func() error {
				if apiKeyService != nil {
					apiKeyService.StopAuthCacheInvalidationSubscriber()
				}
				return nil
			}},
			{"OpsRuntimeSettingsRefresh", func() error {
				if opsService != nil {
					opsService.StopRuntimeSettingsRefresh()
				}
				return nil
			}},
			{"PromptAuditService", func() error {
				if promptAudit != nil {
					return promptAudit.Shutdown(ctx)
				}
				return nil
			}},
			{"OpsScheduledReportService", func() error {
				backgroundgate.Stop("OpsScheduledReportService", func() {
					if opsScheduledReport != nil {
						opsScheduledReport.Stop()
					}
				})
				return nil
			}},
			{"OpsCleanupService", func() error {
				backgroundgate.Stop("OpsCleanupService", func() {
					if opsCleanup != nil {
						opsCleanup.Stop()
					}
				})
				return nil
			}},
			{"OpsSystemLogSink", func() error {
				if opsSystemLogSink != nil {
					opsSystemLogSink.Stop()
				}
				return nil
			}},
			{"AuditLogService", func() error {
				if auditLog != nil {
					auditLog.Stop()
				}
				return nil
			}},
			{"OpsAlertEvaluatorService", func() error {
				backgroundgate.Stop("OpsAlertEvaluatorService", func() {
					if opsAlertEvaluator != nil {
						opsAlertEvaluator.Stop()
					}
				})
				return nil
			}},
			{"OpsAggregationService", func() error {
				backgroundgate.Stop("OpsAggregationService", func() {
					if opsAggregation != nil {
						opsAggregation.Stop()
					}
				})
				return nil
			}},
			{"OpsMetricsCollector", func() error {
				backgroundgate.Stop("OpsMetricsCollector", func() {
					if opsMetricsCollector != nil {
						opsMetricsCollector.Stop()
					}
				})
				return nil
			}},
			{"SchedulerSnapshotService", func() error {
				backgroundgate.Stop("SchedulerSnapshotService", func() {
					if schedulerSnapshot != nil {
						schedulerSnapshot.Stop()
					}
				})
				return nil
			}},
			{"UsageCleanupService", func() error {
				backgroundgate.Stop("UsageCleanupService", func() {
					if usageCleanup != nil {
						usageCleanup.Stop()
					}
				})
				return nil
			}},
			{"IdempotencyCleanupService", func() error {
				backgroundgate.Stop("IdempotencyCleanupService", func() {
					if idempotencyCleanup != nil {
						idempotencyCleanup.Stop()
					}
				})
				return nil
			}},
			{"GeminiCachedContentService", func() error {
				backgroundgate.Stop("GeminiCachedContentService", func() {
					if geminiCachedContent != nil {
						geminiCachedContent.Stop()
					}
				})
				return nil
			}},
			{"BatchImageCleanupService", func() error {
				backgroundgate.Stop("BatchImageCleanupService", func() {
					if batchImageCleanup != nil {
						batchImageCleanup.Stop()
					}
				})
				return nil
			}},
			{"BatchImageWorkerRuntime", func() error {
				backgroundgate.Stop("BatchImageWorkerRuntime", func() {
					if batchImageWorker != nil {
						batchImageWorker.Stop()
					}
				})
				return nil
			}},
			{"TokenRefreshService", func() error {
				backgroundgate.Stop("TokenRefreshService", func() {
					tokenRefresh.Stop()
				})
				return nil
			}},
			{"AccountExpiryService", func() error {
				backgroundgate.Stop("AccountExpiryService", func() {
					accountExpiry.Stop()
				})
				return nil
			}},
			{"CNProviderBalanceCheckService", func() error {
				backgroundgate.Stop("CNProviderBalanceCheckService", func() {
					if cnProviderBalanceCheck != nil {
						cnProviderBalanceCheck.Stop()
					}
				})
				return nil
			}},
			{"OpenAICodexVersionSyncService", func() error {
				backgroundgate.Stop("OpenAICodexVersionSyncService", func() {
					codexVersionSync.Stop()
				})
				return nil
			}},
			{"ClaudeCodeVersionSyncService", func() error {
				backgroundgate.Stop("ClaudeCodeVersionSyncService", func() {
					claudeCodeVersionSync.Stop()
				})
				return nil
			}},
			{"ProxyExpiryService", func() error {
				backgroundgate.Stop("ProxyExpiryService", func() {
					proxyExpiry.Stop()
				})
				return nil
			}},
			{"SubscriptionExpiryService", func() error {
				backgroundgate.Stop("SubscriptionExpiryService", func() {
					subscriptionExpiry.Stop()
				})
				return nil
			}},
			{"SubscriptionService", func() error {
				if subscriptionService != nil {
					subscriptionService.Stop()
				}
				return nil
			}},
			{"PricingService", func() error {
				pricing.Stop()
				return nil
			}},
			{"ArenaLeaderboardService", func() error {
				if arenaLeaderboard != nil {
					arenaLeaderboard.Stop()
				}
				return nil
			}},
			{"EmailQueueService", func() error {
				emailQueue.Stop()
				return nil
			}},
			{"OAuthService", func() error {
				oauth.Stop()
				return nil
			}},
			{"OpenAIOAuthService", func() error {
				openaiOAuth.Stop()
				return nil
			}},
			{"GeminiOAuthService", func() error {
				geminiOAuth.Stop()
				return nil
			}},
			{"AntigravityOAuthService", func() error {
				antigravityOAuth.Stop()
				return nil
			}},
			{"GrokOAuthService", func() error {
				if grokOAuth != nil {
					grokOAuth.Stop()
				}
				return nil
			}},
			{"OpenAIWSPool", func() error {
				if openAIGateway != nil {
					openAIGateway.CloseOpenAIWSPool()
				}
				return nil
			}},
			{"ScheduledTestRunnerService", func() error {
				backgroundgate.Stop("ScheduledTestRunnerService", func() {
					if scheduledTestRunner != nil {
						scheduledTestRunner.Stop()
					}
				})
				return nil
			}},
			{"BackupService", func() error {
				backgroundgate.Stop("BackupService", func() {
					if backupSvc != nil {
						backupSvc.Stop()
					}
				})
				return nil
			}},
			{"PaymentOrderExpiryService", func() error {
				backgroundgate.Stop("PaymentOrderExpiryService", func() {
					if paymentOrderExpiry != nil {
						paymentOrderExpiry.Stop()
					}
				})
				return nil
			}},
			{"ChannelMonitorV2Aggregator", func() error {
				backgroundgate.Stop("ChannelMonitorV2Aggregator", func() {
					if channelMonitorV2Aggregator != nil {
						channelMonitorV2Aggregator.Stop()
					}
				})
				return nil
			}},
			{"ChannelMonitorRunner", func() error {
				backgroundgate.Stop("ChannelMonitorRunner", func() {
					if channelMonitorRunner != nil {
						channelMonitorRunner.Stop()
					}
				})
				return nil
			}},
			{"UserPlatformQuotaUsageFlusher", func() error {
				backgroundgate.Stop("UserPlatformQuotaUsageFlusher", func() {
					if quotaFlusher != nil {
						quotaFlusher.Stop()
					}
				})
				return nil
			}},
			{"UpstreamBillingProbeService", func() error {
				backgroundgate.Stop("UpstreamBillingProbeService", func() {
					if upstreamBillingProbe != nil {
						upstreamBillingProbe.Stop()
					}
				})
				return nil
			}},
			{"OllamaCloudUsageService", func() error {
				if ollamaCloudUsage != nil {
					ollamaCloudUsage.Stop()
				}
				return nil
			}},
			{"OpenCodeGoUsageService", func() error {
				backgroundgate.Stop("OpenCodeGoUsageService", func() {
					if opencodeGoUsage != nil {
						opencodeGoUsage.Stop()
					}
				})
				return nil
			}},
		}

		serviceSteps = append(serviceSteps,
			cleanupStep{"UserMessageQueueCleanup", func() error {
				backgroundgate.Stop("UserMessageQueueCleanup", func() {
					if userMessageQueue != nil {
						userMessageQueue.Stop()
					}
				})
				return nil
			}},
			cleanupStep{"DeferredService", func() error {
				if deferred != nil {
					deferred.Stop()
				}
				return nil
			}},
			cleanupStep{"TimingWheelService", func() error {
				if timingWheel != nil {
					timingWheel.Stop()
				}
				return nil
			}},
		)

		// All usage producers and request-local queues finish before billing cache
		// writers are flushed, and shared infrastructure is always closed last.
		infraSteps := []cleanupStep{
			{"BillingCacheService", func() error {
				if billingCache != nil {
					billingCache.Stop()
				}
				return nil
			}},
			{"Redis", func() error {
				if rdb == nil {
					return nil
				}
				return rdb.Close()
			}},
			{"Ent", func() error {
				if entClient == nil {
					return nil
				}
				return entClient.Close()
			}},
		}

		runSequential := func(steps []cleanupStep) {
			for i := range steps {
				step := steps[i]
				if err := step.fn(); err != nil {
					log.Printf("[Cleanup] %s failed: %v", step.name, err)
					continue
				}
				log.Printf("[Cleanup] %s succeeded", step.name)
			}
		}

		runSequential(serviceSteps)
		runSequential(infraSteps)

		// Check if context timed out
		select {
		case <-ctx.Done():
			log.Printf("[Cleanup] Warning: cleanup timed out after 10 seconds")
		default:
			log.Printf("[Cleanup] All cleanup steps completed")
		}
	}
}
