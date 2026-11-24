package main

//go:generate go run github.com/google/wire/cmd/wire

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/pkg/backgroundgate"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/setup"
	"github.com/Wei-Shaw/sub2api/internal/web"

	"github.com/gin-gonic/gin"
)

//go:embed VERSION
var embeddedVersion string

// Build-time variables (can be set by ldflags)
var (
	Version   = ""
	Commit    = "unknown"
	Date      = "unknown"
	BuildType = "source" // "source" for manual builds, "release" for CI builds (set by ldflags)
)

func init() {
	// 如果 Version 已通过 ldflags 注入（例如 -X main.Version=...），则不要覆盖。
	if strings.TrimSpace(Version) != "" {
		return
	}

	// 默认从 embedded VERSION 文件读取版本号（编译期打包进二进制）。
	Version = strings.TrimSpace(embeddedVersion)
	if Version == "" {
		Version = "0.0.0-dev"
	}
}

// initLogger configures the default slog handler based on gin.Mode().
// In non-release mode, Debug level logs are enabled.
func main() {
	logger.InitBootstrap()
	defer logger.Sync()

	// Parse command line flags
	setupMode := flag.Bool("setup", false, "Run setup wizard in CLI mode")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		log.Printf("Sub2API %s (commit: %s, built: %s)\n", Version, Commit, Date)
		return
	}

	if err := backgroundgate.InitializeFromEnv(); err != nil {
		log.Fatalf("Invalid rolling background configuration: %v", err)
	}
	defer backgroundgate.Close()
	if backgroundgate.Configured() && *setupMode {
		log.Fatal("Rolling mode cannot run the setup CLI")
	}

	// CLI setup mode
	if *setupMode {
		if err := setup.RunCLI(); err != nil {
			log.Fatalf("Setup failed: %v", err)
		}
		return
	}

	// Check if setup is needed
	if setup.NeedsSetup() {
		if backgroundgate.Configured() {
			log.Fatal("Rolling mode requires an already installed application; setup is disabled")
		}
		// Check if auto-setup is enabled (for Docker deployment)
		if setup.AutoSetupEnabled() {
			log.Println("Auto setup mode enabled...")
			if err := setup.AutoSetupFromEnv(); err != nil {
				log.Fatalf("Auto setup failed: %v", err)
			}
			// Continue to main server after auto-setup
		} else {
			log.Println("First run detected, starting setup wizard...")
			runSetupServer()
			return
		}
	}

	// Normal server mode
	if err := runMainServer(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func runSetupServer() {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS(config.CORSConfig{}))
	r.Use(middleware.SecurityHeaders(config.CSPConfig{Enabled: true, Policy: config.DefaultCSPPolicy}, nil))

	// Register setup routes
	setup.RegisterRoutes(r)

	// Serve embedded frontend if available
	if web.HasEmbeddedFrontend() {
		r.Use(web.ServeEmbeddedFrontend())
	}

	// Get server address from config.yaml or environment variables (SERVER_HOST, SERVER_PORT)
	// This allows users to run setup on a different address if needed
	addr := config.GetServerAddress()
	log.Printf("Setup wizard available at http://%s", addr)
	log.Println("Complete the setup wizard to configure Sub2API")

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	server := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 30 * time.Second,
		IdleTimeout:       120 * time.Second,
		Protocols:         protocols,
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Failed to start setup server: %v", err)
	}
}

func runMainServer() error {
	cfg, err := config.LoadForBootstrap()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := logger.Init(logger.OptionsFromConfig(cfg.Log)); err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}
	if cfg.RunMode == config.RunModeSimple {
		log.Println("⚠️  WARNING: Running in SIMPLE mode - billing and quota checks are DISABLED")
	}

	buildInfo := handler.BuildInfo{
		Version:   Version,
		BuildType: BuildType,
	}

	app, err := initializeApplication(buildInfo)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}
	defer app.Cleanup()
	if app.PluginManager != nil {
		if err := backgroundgate.RunError("PluginManager", func() error { return app.PluginManager.Start(context.Background()) }); err != nil {
			log.Printf("Plugin manager started in degraded state: %v", err)
		}
	}
	if app.PromptAudit != nil {
		if err := app.PromptAudit.Start(context.Background()); err != nil {
			// Startup continues so unrelated APIs stay up. Fail-closed (unavailable)
			// applies only when a persisted blocking policy was observed; without
			// blocking intent, Prompt Audit stays ModeOff so the gateway remains
			// usable and administrators can still disable the feature (#4560).
			log.Printf("Prompt Audit started in degraded state: %v", err)
		}
	}

	tracker := &rollingConnections{}
	app.Server.Handler = rollingHealthHandler(app.Server.Handler, tracker, func() (uint64, int64, uint64) {
		stats := app.UsageRecordWorkerPool.Stats()
		var pending uint64
		if stats.SubmittedTasks > stats.DroppedTasks {
			accepted := stats.SubmittedTasks - stats.DroppedTasks
			if accepted > stats.CompletedTasks {
				pending = accepted - stats.CompletedTasks
			}
		}
		return stats.WaitingTasks, stats.RunningWorkers, pending
	}, app.BalanceNotify.UsageNotificationStats)
	listener, err := net.Listen("tcp", app.Server.Addr)
	if err != nil {
		return fmt.Errorf("bind server: %w", err)
	}
	if backgroundgate.Configured() {
		listener = &rollingListener{Listener: listener, tracker: tracker}
		previous := app.Server.ConnState
		app.Server.ConnState = func(conn net.Conn, state http.ConnState) {
			if previous != nil {
				previous(conn, state)
			}
			if state == http.StateHijacked {
				if conn, ok := conn.(*rollingConnection); ok {
					conn.markHijacked()
				}
			}
		}
	}
	// Bind before sealing: a pre-existing marker cannot start shared work while
	// providers are still registering or when this listener cannot be opened.
	backgroundgate.Seal()
	serveErrors := make(chan error, 1)
	go func() { serveErrors <- app.Server.Serve(listener) }()
	log.Printf("Server started on %s", app.Server.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)
	var serveErr error
	select {
	case <-quit:
	case serveErr = <-serveErrors:
	}
	log.Println("Shutting down server...")
	backgroundgate.Close()
	ctx := context.Background()
	if !backgroundgate.Configured() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}
	if err := app.Server.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP shutdown: %w", err)
	}
	// Shutdown does not wait for hijacked WebSockets. Their accepted connections
	// remain owned by this process until the handlers close them normally.
	if backgroundgate.Configured() {
		tracker.waitForHijacked()
	}
	log.Println("Server exited")
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", serveErr)
	}
	return nil
}

// rollingHealthHandler only observes state; activation is exclusively controlled
// by a protected on-disk marker after the previous writer has exited.
func rollingHealthHandler(next http.Handler, tracker *rollingConnections, usage func() (uint64, int64, uint64), notifications func() service.UsageNotificationStats) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health/rolling" {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			var waiting, pending uint64
			var running int64
			if usage != nil {
				waiting, running, pending = usage()
			}
			notificationStats := service.UsageNotificationStats{ShutdownState: "active"}
			if notifications != nil {
				notificationStats = notifications()
			}
			_ = json.NewEncoder(w).Encode(struct {
				backgroundgate.Status
				Requests     int64  `json:"in_flight_requests"`
				Hijacked     int64  `json:"hijacked_connections"`
				UsageWaiting uint64 `json:"usage_waiting_tasks"`
				UsageRunning int64  `json:"usage_running_workers"`
				UsagePending uint64 `json:"usage_pending_tasks"`
				service.UsageNotificationStats
			}{backgroundgate.Snapshot(), tracker.requests.Load(), tracker.hijacked.Load(), waiting, running, pending, notificationStats})
			return
		}
		tracker.requests.Add(1)
		defer tracker.requests.Add(-1)
		next.ServeHTTP(w, r)
	})
}

type rollingConnections struct{ requests, hijacked atomic.Int64 }

func (t *rollingConnections) waitForHijacked() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for t.hijacked.Load() != 0 {
		<-ticker.C
	}
}

type rollingListener struct {
	net.Listener
	tracker *rollingConnections
}

func (l *rollingListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &rollingConnection{Conn: conn, tracker: l.tracker}, nil
}

type rollingConnection struct {
	net.Conn
	tracker          *rollingConnections
	mu               sync.Mutex
	hijacked, closed bool
}

func (c *rollingConnection) markHijacked() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed && !c.hijacked {
		c.hijacked = true
		c.tracker.hijacked.Add(1)
	}
}
func (c *rollingConnection) Close() error {
	err := c.Conn.Close()
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		if c.hijacked {
			c.tracker.hijacked.Add(-1)
		}
	}
	return err
}
