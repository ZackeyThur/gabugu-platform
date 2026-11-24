package service

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alitto/pond/v2"
)

const (
	usageNotificationWorkers   = 4
	usageNotificationQueueSize = 128
	usageNotificationTimeout   = 30 * time.Second
)

// Notification work is best effort and must not create an unbounded queue or
// compete with committed-charge persistence for usage workers. A task includes
// both settings reads and email delivery, including all quota dimensions.
type usageNotificationWorker struct {
	pool              pond.Pool
	ctx               context.Context
	cancel            context.CancelFunc
	mu                sync.Mutex
	stopping          bool
	stopOnce          sync.Once
	dropped           atomic.Uint64
	completed         atomic.Uint64
	shutdownCancelled atomic.Uint64
	shutdownState     atomic.Uint32
	lastDropLog       atomic.Int64
	drainTimeout      time.Duration
}

type UsageNotificationStats struct {
	Pending           uint64 `json:"usage_notification_pending"`
	Running           int64  `json:"usage_notification_running"`
	Dropped           uint64 `json:"usage_notification_dropped"`
	ShutdownCancelled uint64 `json:"usage_notification_shutdown_cancelled"`
	ShutdownState     string `json:"usage_notification_shutdown_state"`
}

func newUsageNotificationWorker() *usageNotificationWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &usageNotificationWorker{
		pool:         pond.NewPool(usageNotificationWorkers, pond.WithQueueSize(usageNotificationQueueSize)),
		ctx:          ctx,
		cancel:       cancel,
		drainTimeout: 60 * time.Second,
	}
}

func (w *usageNotificationWorker) submit(task func(context.Context)) bool {
	if w == nil || task == nil {
		return false
	}
	w.mu.Lock()
	if w.stopping {
		w.mu.Unlock()
		w.dropped.Add(1)
		return false
	}
	_, accepted := w.pool.TrySubmit(func() {
		if w.ctx.Err() != nil {
			w.shutdownCancelled.Add(1)
			return
		}
		ctx, cancel := context.WithTimeout(w.ctx, usageNotificationTimeout)
		defer cancel()
		defer w.completed.Add(1)
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("panic in usage notification", "recover", recovered)
			}
		}()
		task(ctx)
		if w.ctx.Err() != nil {
			w.shutdownCancelled.Add(1)
		}
	})
	w.mu.Unlock()
	if !accepted {
		count := w.dropped.Add(1)
		now := time.Now().UnixNano()
		last := w.lastDropLog.Load()
		if now-last >= int64(5*time.Second) && w.lastDropLog.CompareAndSwap(last, now) {
			slog.Warn("usage notification queue unavailable; notification skipped", "dropped_total", count)
		}
	}
	return accepted
}

func (w *usageNotificationWorker) stop() {
	if w == nil {
		return
	}
	w.stopOnce.Do(func() {
		w.mu.Lock()
		w.stopping = true
		w.mu.Unlock()
		w.shutdownState.Store(1)

		started := time.Now()
		stopTask := w.pool.Stop()
		slog.Info("usage_notification_drain_started", "queued", w.pool.WaitingTasks(), "running", w.pool.RunningWorkers())
		timer := time.NewTimer(w.drainTimeout)
		defer timer.Stop()
		select {
		case <-stopTask.Done():
			w.shutdownState.Store(2)
			slog.Info("usage_notification_drain_completed", "elapsed_ms", time.Since(started).Milliseconds(), "completed", w.completed.Load(), "dropped", w.dropped.Load())
		case <-timer.C:
			w.shutdownState.Store(3)
			queued := w.pool.WaitingTasks()
			running := w.pool.RunningWorkers()
			slog.Error("usage_notification_drain_timeout", "queued", queued, "running", running, "completed", w.completed.Load())
			w.cancel()
			stopTask.Wait()
			slog.Error("usage_notification_drain_cancelled", "queued_at_timeout", queued, "running_at_timeout", running, "cancelled_queued", w.shutdownCancelled.Load(), "elapsed_ms", time.Since(started).Milliseconds())
		}
	})
}

func (s *BalanceNotifyService) UsageNotificationStats() UsageNotificationStats {
	if s == nil || s.usageNotifications == nil {
		return UsageNotificationStats{ShutdownState: "active"}
	}
	w := s.usageNotifications
	state := "active"
	switch w.shutdownState.Load() {
	case 1:
		state = "draining"
	case 2:
		state = "drained"
	case 3:
		state = "timed_out"
	}
	return UsageNotificationStats{
		Pending:           w.pool.WaitingTasks(),
		Running:           w.pool.RunningWorkers(),
		Dropped:           w.dropped.Load(),
		ShutdownCancelled: w.shutdownCancelled.Load(),
		ShutdownState:     state,
	}
}

func (s *BalanceNotifyService) submitUsageNotifications(p *postUsageBillingParams, deps *billingDeps, result *UsageBillingApplyResult) {
	if s == nil || s.usageNotifications == nil || p == nil || p.Cost == nil {
		return
	}
	balanceEnabled := !p.IsSubscriptionBill && p.Cost.ActualCost > 0 && s.canNotifyBalance(p.User)
	quotaEnabled := p.Cost.TotalCost > 0 && p.Account != nil && p.Account.SupportsAccountQuota() && p.Account.HasAnyQuotaLimit()
	if !balanceEnabled && !quotaEnabled {
		return
	}
	s.usageNotifications.submit(func(ctx context.Context) {
		notifyBalanceLow(ctx, p, deps, result)
		if ctx.Err() == nil {
			notifyAccountQuota(ctx, p, deps, result)
		}
	})
}

// Stop cancels notification work and waits for the bounded workers to exit.
// Billing persistence belongs to usage tasks and is never dropped here.
func (s *BalanceNotifyService) Stop() {
	if s != nil {
		s.usageNotifications.stop()
	}
}
