package backgroundgate

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testGate(t *testing.T) *Gate {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	g, err := New(filepath.Join(dir, "activate.json"), "candidate-v0216")
	if err != nil {
		t.Fatal(err)
	}
	g.Seal()
	t.Cleanup(g.Close)
	return g
}

func publish(t *testing.T, g *Gate, body string) {
	t.Helper()
	if err := os.WriteFile(g.path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

const validMarker = `{"release_id":"candidate-v0216","activate":true}`

func TestImmediateModePreservesStartStop(t *testing.T) {
	g, err := New("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	started, stopped := 0, 0
	g.Run("worker", func() { started++ })
	g.Run("worker", func() { t.Fatal("duplicate start") })
	g.Stop("worker", func() { stopped++ })
	g.Stop("worker", func() { t.Fatal("duplicate stop") })
	if started != 1 || stopped != 1 {
		t.Fatalf("starts=%d stops=%d", started, stopped)
	}
}

func TestDefaultPackageFunctionsPreserveIndependentInstances(t *testing.T) {
	old := processGate
	processGate, _ = New("", "")
	t.Cleanup(func() { processGate.Close(); processGate = old })
	starts, stops := 0, 0
	for i := 0; i < 2; i++ {
		Run("same-service-name", func() { starts++ })
		Stop("same-service-name", func() { stops++ })
	}
	if starts != 2 || stops != 2 {
		t.Fatalf("default mode changed independent lifecycles: %d/%d", starts, stops)
	}
}

func TestInitializeRejectsHalfConfiguredEnvironment(t *testing.T) {
	t.Setenv(ActivationFileEnv, "")
	t.Setenv(ReleaseIDEnv, "candidate")
	if err := InitializeFromEnv(); err == nil {
		t.Fatal("half configured process accepted")
	}
}

func TestRejectIncompleteOrRelativeConfiguration(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "activate.json")
	for _, tc := range [][2]string{{"", "candidate"}, {abs, ""}, {"relative.json", "candidate"}, {abs, " ../invalid"}, {abs, "candidate\n"}} {
		if _, err := New(tc[0], tc[1]); err == nil {
			t.Fatalf("accepted invalid config %q", tc)
		}
	}
}

func TestMissingOrInvalidMarkerCannotStartWorkers(t *testing.T) {
	g := testGate(t)
	var starts atomic.Int32
	g.Run("worker", func() { starts.Add(1) })
	if err := g.checkActivation(); !os.IsNotExist(err) {
		t.Fatalf("missing marker: %v", err)
	}
	for _, body := range []string{"", "{", `{"release_id":"wrong","activate":true}`, `{"release_id":"candidate-v0216","activate":false}`, `{"release_id":"candidate-v0216","activate":true,"force":true}`, validMarker + " {}", strings.Repeat("x", 4097)} {
		publish(t, g, body)
		if err := g.checkActivation(); err == nil {
			t.Fatalf("accepted %q", body[:min(len(body), 80)])
		}
	}
	if starts.Load() != 0 {
		t.Fatal("invalid marker started worker")
	}
	publish(t, g, validMarker)
	if err := g.checkActivation(); err != nil {
		t.Fatal(err)
	}
	if starts.Load() != 1 {
		t.Fatal("valid marker failed to start worker")
	}
}

func TestStopBeforeActivationSkipsUnsafeStopAndLateStart(t *testing.T) {
	g := testGate(t)
	g.Run("worker", func() { t.Error("stopped worker started") })
	g.Stop("worker", func() { t.Error("Stop of unstarted worker called") })
	g.Stop("not-registered", func() { t.Error("unknown worker stopped") })
	g.Run("not-registered", func() { t.Error("registered after Stop") })
	publish(t, g, validMarker)
	if err := g.checkActivation(); err != nil {
		t.Fatal(err)
	}
}

func TestCloseBeforeMarkerPreventsActivationAndReturns(t *testing.T) {
	g := testGate(t)
	g.Run("worker", func() { t.Error("closed gate started worker") })
	g.Watch()
	done := make(chan struct{})
	go func() { g.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown waited for missing marker")
	}
	publish(t, g, validMarker)
	if err := g.checkActivation(); err != nil {
		t.Fatal(err)
	}
	g.Run("late-worker", func() { t.Error("late registration started") })
}

func TestConcurrentActivationRegistersAndStartsExactlyOnce(t *testing.T) {
	g := testGate(t)
	publish(t, g, validMarker)
	var starts atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(n int) { defer wg.Done(); g.Run(fmt.Sprint(n%10), func() { starts.Add(1) }) }(i)
		go func() {
			defer wg.Done()
			if err := g.checkActivation(); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if starts.Load() != 10 {
		t.Fatalf("starts=%d want 10", starts.Load())
	}
}

func TestStopWaitsForStartAndRunsOnce(t *testing.T) {
	g := testGate(t)
	entered, release, stopped := make(chan struct{}), make(chan struct{}), make(chan struct{})
	g.Run("worker", func() { close(entered); <-release })
	publish(t, g, validMarker)
	activated := make(chan struct{})
	go func() { _ = g.checkActivation(); close(activated) }()
	<-entered
	go func() { g.Stop("worker", func() { close(stopped) }) }()
	select {
	case <-stopped:
		t.Fatal("stop ran before start returned")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("stop blocked")
	}
	<-activated
}

func TestFeatureDisabledStartIsNotForced(t *testing.T) {
	g := testGate(t)
	featureEnabled := false
	started := false
	g.Run("optional", func() {
		if featureEnabled {
			started = true
		}
	})
	publish(t, g, validMarker)
	if err := g.checkActivation(); err != nil {
		t.Fatal(err)
	}
	if started {
		t.Fatal("activation overrode feature config")
	}
}

func TestWatcherActivatesAfterAtomicPublication(t *testing.T) {
	g := testGate(t)
	started := make(chan struct{})
	g.Run("worker", func() { close(started) })
	g.Watch()
	select {
	case <-started:
		t.Fatal("worker started without marker")
	default:
	}
	tmp := g.path + ".tmp"
	if err := os.WriteFile(tmp, []byte(validMarker), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, g.path); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("watcher missed publication")
	}
}

func TestConcurrentCloseAndActivationCannotStartAfterClose(t *testing.T) {
	for i := 0; i < 30; i++ {
		g := testGate(t)
		var starts atomic.Int32
		g.Run("worker", func() { starts.Add(1) })
		publish(t, g, validMarker)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); _ = g.checkActivation() }()
		go func() { defer wg.Done(); g.Close() }()
		wg.Wait()
		before := starts.Load()
		_ = g.checkActivation()
		g.Run("late", func() { starts.Add(1) })
		if starts.Load() != before || before > 1 {
			t.Fatal("start occurred after close or duplicated")
		}
	}
}

func TestRejectSymlinkAndWritableMarker(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix deployment filesystem security test")
	}
	g := testGate(t)
	target := filepath.Join(filepath.Dir(g.path), "target")
	if err := os.WriteFile(target, []byte(validMarker), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, g.path); err != nil {
		t.Fatal(err)
	}
	if err := g.checkActivation(); err == nil {
		t.Fatal("accepted symlink")
	}
	if err := os.Remove(g.path); err != nil {
		t.Fatal(err)
	}
	publish(t, g, validMarker)
	if err := os.Chmod(g.path, 0666); err != nil {
		t.Fatal(err)
	}
	if err := g.checkActivation(); err == nil {
		t.Fatal("accepted writable marker")
	}
	if err := os.Chmod(g.path, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Dir(g.path), 0777); err != nil {
		t.Fatal(err)
	}
	if err := g.checkActivation(); err == nil {
		t.Fatal("accepted writable directory")
	}
}

func TestMarkerCannotActivateUntilRegistrationComplete(t *testing.T) {
	g := testGate(t)
	g.mu.Lock()
	g.sealed = false
	g.mu.Unlock()
	started := false
	g.Run("worker", func() { started = true })
	publish(t, g, validMarker)
	if err := g.checkActivation(); err != nil {
		t.Fatal(err)
	}
	if started || g.Snapshot().Active {
		t.Fatal("unsealed gate activated")
	}
	g.Seal()
	if err := g.checkActivation(); err != nil {
		t.Fatal(err)
	}
	if !started || !g.Snapshot().Ready {
		t.Fatalf("sealed gate not ready: %+v", g.Snapshot())
	}
}

func TestDeferredFailureIsVisibleAndStopsOnce(t *testing.T) {
	g := testGate(t)
	stops := 0
	g.register("failed", func() error { return fmt.Errorf("private provider failure") }, nil)
	publish(t, g, validMarker)
	if err := g.checkActivation(); err != nil {
		t.Fatal(err)
	}
	state := g.Snapshot()
	if state.Ready || state.Errors["failed"] != "startup failed" {
		t.Fatalf("unexpected status: %+v", state)
	}
	g.Stop("failed", func() { stops++ })
	g.Stop("failed", func() { stops++ })
	if stops != 1 {
		t.Fatalf("stops=%d", stops)
	}
}

func TestCloseWaitsForStartOutsideWatcher(t *testing.T) {
	g, err := New("", "")
	if err != nil {
		t.Fatal(err)
	}
	entered, release, closed := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go g.Run("slow", func() { close(entered); <-release })
	<-entered
	go func() { g.Close(); close(closed) }()
	select {
	case <-closed:
		t.Fatal("close returned during start callback")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("close did not finish")
	}
	g.Run("late", func() { t.Error("start after close") })
}

func TestManagedRegistrationCleanupCannotRaceActivation(t *testing.T) {
	g := testGate(t)
	started, stopped := 0, 0
	g.register("managed", func() error { started++; return nil }, func() { stopped++ })
	publish(t, g, validMarker)
	if err := g.checkActivation(); err != nil {
		t.Fatal(err)
	}
	g.Close()
	e := g.entries["managed"]
	g.Stop("managed", e.cleanup)
	g.Stop("managed", e.cleanup)
	if started != 1 || stopped != 1 {
		t.Fatalf("start/stop=%d/%d", started, stopped)
	}
}
