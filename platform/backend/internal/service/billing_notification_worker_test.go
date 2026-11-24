package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUsageNotificationsDrainQueuedWorkOnStop(t *testing.T) {
	w := newUsageNotificationWorker()
	t.Cleanup(w.stop)
	started := make(chan struct{}, usageNotificationWorkers)
	release := make(chan struct{})
	var queuedRan atomic.Int64
	for range usageNotificationWorkers {
		require.True(t, w.submit(func(ctx context.Context) {
			started <- struct{}{}
			select {
			case <-release:
			case <-ctx.Done():
			}
		}))
	}
	for range usageNotificationWorkers {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("notification worker did not start")
		}
	}
	for range usageNotificationQueueSize {
		require.True(t, w.submit(func(context.Context) { queuedRan.Add(1) }))
	}
	require.False(t, w.submit(func(context.Context) { queuedRan.Add(1) }))
	require.Equal(t, uint64(1), w.dropped.Load())

	stopped := make(chan struct{})
	go func() { w.stop(); close(stopped) }()
	select {
	case <-stopped:
		t.Fatal("stop returned before active notifications drained")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("notification queue did not drain")
	}
	w.stop()
	require.Equal(t, int64(usageNotificationQueueSize), queuedRan.Load())
	require.Zero(t, w.pool.WaitingTasks())
	require.Zero(t, w.shutdownCancelled.Load())
	require.False(t, w.submit(func(context.Context) { t.Error("stopped worker accepted work") }))
}

func TestUsageNotificationsReportAndCancelAfterDrainTimeout(t *testing.T) {
	w := newUsageNotificationWorker()
	w.drainTimeout = 20 * time.Millisecond
	started := make(chan struct{}, usageNotificationWorkers)
	for range usageNotificationWorkers {
		require.True(t, w.submit(func(ctx context.Context) {
			started <- struct{}{}
			<-ctx.Done()
		}))
	}
	for range usageNotificationWorkers {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("notification worker did not start")
		}
	}
	for range 3 {
		require.True(t, w.submit(func(context.Context) {}))
	}
	w.stop()
	require.Zero(t, w.pool.WaitingTasks())
	require.Equal(t, uint64(usageNotificationWorkers+3), w.shutdownCancelled.Load())
}

type notificationCancellationSettings struct {
	SettingRepository
	started chan context.Context
	release chan struct{}
}

func (s *notificationCancellationSettings) GetMultiple(ctx context.Context, _ []string) (map[string]string, error) {
	s.started <- ctx
	select {
	case <-s.release:
		return map[string]string{}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestBalanceNotifyStopWaitsForSettingsAndNotificationDrain(t *testing.T) {
	settings := &notificationCancellationSettings{started: make(chan context.Context, 1), release: make(chan struct{})}
	s := NewBalanceNotifyService(NewEmailService(settings, nil), settings, nil)
	t.Cleanup(s.Stop)
	s.submitUsageNotifications(&postUsageBillingParams{
		Cost: &CostBreakdown{ActualCost: 1},
		User: &User{ID: 1, BalanceNotifyEnabled: true},
	}, &billingDeps{balanceNotifyService: s}, nil)
	select {
	case ctx := <-settings.started:
		_, ok := ctx.Deadline()
		require.True(t, ok)
	case <-time.After(2 * time.Second):
		t.Fatal("notification settings read did not start")
	}
	stopped := make(chan struct{})
	go func() { s.Stop(); close(stopped) }()
	select {
	case <-stopped:
		t.Fatal("notification service stopped before its active read completed")
	case <-time.After(50 * time.Millisecond):
	}
	close(settings.release)
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("notification service did not stop after work completed")
	}
}
