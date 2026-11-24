package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestRollingHealthIsReadOnlyAndReportsInflight(t *testing.T) {
	tracker := &rollingConnections{}
	entered, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); close(entered); <-release })
	handler := rollingHealthHandler(next, tracker, func() (uint64, int64, uint64) { return 3, 2, 4 }, func() service.UsageNotificationStats {
		return service.UsageNotificationStats{Pending: 5, Running: 2, Dropped: 1, ShutdownCancelled: 3, ShutdownState: "active"}
	})
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/stream", nil))
		close(done)
	}()
	<-entered
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/rolling", nil))
	var state struct {
		InFlight              int64  `json:"in_flight_requests"`
		Waiting               uint64 `json:"usage_waiting_tasks"`
		Running               int64  `json:"usage_running_workers"`
		Active                bool   `json:"active"`
		NotificationPending   uint64 `json:"usage_notification_pending"`
		NotificationRunning   int64  `json:"usage_notification_running"`
		NotificationDropped   uint64 `json:"usage_notification_dropped"`
		NotificationCancelled uint64 `json:"usage_notification_shutdown_cancelled"`
		NotificationState     string `json:"usage_notification_shutdown_state"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state.InFlight != 1 || state.Waiting != 3 || state.Running != 2 {
		t.Fatalf("state=%+v", state)
	}
	if state.NotificationPending != 5 || state.NotificationRunning != 2 || state.NotificationDropped != 1 || state.NotificationCancelled != 3 || state.NotificationState != "active" {
		t.Fatalf("notification drain state=%+v", state)
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(method, "/health/rolling", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s status=%d", method, rec.Code)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("health forwarded to application handler")
	}
	close(release)
	<-done
	if tracker.requests.Load() != 0 {
		t.Fatal("request counter leaked")
	}
}

func TestRollingTracksHijackedConnectionAfterHTTPShutdown(t *testing.T) {
	tracker := &rollingConnections{}
	raw, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	listener := &rollingListener{Listener: raw, tracker: tracker}
	hijacked := make(chan net.Conn, 1)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		hijacked <- conn
	}), ConnState: func(conn net.Conn, state http.ConnState) {
		if state == http.StateHijacked {
			conn.(*rollingConnection).markHijacked()
		}
	}}
	go func() { _ = server.Serve(listener) }()
	client, err := net.Dial("tcp", raw.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Write([]byte("GET /ws HTTP/1.1\r\nHost: local\r\n\r\n")); err != nil {
		t.Fatal(err)
	}
	var conn net.Conn
	select {
	case conn = <-hijacked:
	case <-time.After(time.Second):
		t.Fatal("hijack timed out")
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if tracker.hijacked.Load() != 1 {
		t.Fatal("Shutdown lost hijacked connection")
	}
	drained := make(chan struct{})
	go func() { tracker.waitForHijacked(); close(drained) }()
	select {
	case <-drained:
		t.Fatal("drain ignored open websocket")
	case <-time.After(20 * time.Millisecond):
	}
	_ = conn.Close()
	_ = conn.Close()
	select {
	case <-drained:
	case <-time.After(time.Second):
		t.Fatal("drain did not finish")
	}
	if tracker.hijacked.Load() != 0 {
		t.Fatal("connection counted twice")
	}
}
