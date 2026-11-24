// Package backgroundgate defers shared background workers during a rolling
// handoff. Request-local queues and cache subscribers must not use this gate.
package backgroundgate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	ActivationFileEnv = "GABUGU_BACKGROUND_ACTIVATION_FILE"
	ReleaseIDEnv      = "GABUGU_RELEASE_ID"
	maxMarkerBytes    = 4096
)

type entry struct {
	mu         sync.Mutex
	start      func() error
	started    bool
	stopped    bool
	startError string
	cleanup    func()
}

// Gate is a process-local lifecycle registry, not a distributed lease. The
// release controller must prove that the old worker process has exited before
// publishing the activation marker. Removing the marker does not stop workers.
type Gate struct {
	lifecycle sync.RWMutex
	mu        sync.Mutex
	path      string
	releaseID string
	active    bool
	closed    bool
	sealed    bool
	entries   map[string]*entry
	stop      chan struct{}
	done      chan struct{}
	watchOnce sync.Once
	closeOnce sync.Once
}

func New(path, releaseID string) (*Gate, error) {
	if (path == "") != (releaseID == "") {
		return nil, errors.New("background activation file and release ID must both be configured")
	}
	if path != "" {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return nil, errors.New("background activation file must be a clean absolute path")
		}
		if strings.TrimSpace(releaseID) != releaseID || strings.ContainsAny(releaseID, "/\\\r\n") || len(releaseID) > 128 {
			return nil, errors.New("invalid background release ID")
		}
	}
	return &Gate{path: path, releaseID: releaseID, active: path == "", entries: make(map[string]*entry), stop: make(chan struct{}), done: make(chan struct{})}, nil
}

var processGate, _ = New("", "")

// InitializeFromEnv must run before constructing the application providers.
func InitializeFromEnv() error {
	g, err := New(os.Getenv(ActivationFileEnv), os.Getenv(ReleaseIDEnv))
	if err != nil {
		return err
	}
	processGate = g
	g.Watch()
	return nil
}

func Run(name string, start func()) {
	if !Configured() {
		start()
		return
	}
	processGate.Run(name, start)
}
func Stop(name string, stop func()) {
	if !Configured() {
		if stop != nil {
			stop()
		}
		return
	}
	processGate.Stop(name, stop)
}
func Close()           { processGate.Close() }
func Configured() bool { return processGate.path != "" }

// Status is a read-only lifecycle snapshot. Ready means every registered start
// callback returned; controllers must still verify actual worker health.
type Status struct {
	Configured           bool              `json:"configured"`
	ReleaseID            string            `json:"release_id,omitempty"`
	Active               bool              `json:"active"`
	Closed               bool              `json:"closed"`
	RegistrationComplete bool              `json:"registration_complete"`
	Registered           int               `json:"registered"`
	Started              int               `json:"started"`
	Stopped              int               `json:"stopped"`
	Ready                bool              `json:"ready"`
	Tasks                []string          `json:"tasks"`
	Errors               map[string]string `json:"errors,omitempty"`
}

func Snapshot() Status { return processGate.Snapshot() }
func Seal()            { processGate.Seal() }
func (g *Gate) Seal()  { g.mu.Lock(); g.sealed = true; g.mu.Unlock() }

func (g *Gate) Snapshot() Status {
	g.mu.Lock()
	s := Status{Configured: g.path != "", ReleaseID: g.releaseID, Active: g.active, Closed: g.closed, RegistrationComplete: g.sealed}
	entries := make(map[string]*entry, len(g.entries))
	for name, e := range g.entries {
		s.Tasks = append(s.Tasks, name)
		entries[name] = e
	}
	g.mu.Unlock()
	s.Registered = len(entries)
	for _, name := range s.Tasks {
		e := entries[name]
		if !e.mu.TryLock() {
			continue
		}
		if e.started {
			s.Started++
		}
		if e.stopped {
			s.Stopped++
		}
		if e.startError != "" {
			if s.Errors == nil {
				s.Errors = map[string]string{}
			}
				s.Errors[name] = e.startError
		}
		e.mu.Unlock()
	}
	sort.Strings(s.Tasks)
	s.Ready = s.Active && !s.Closed && s.RegistrationComplete && s.Started == s.Registered && s.Stopped == 0 && len(s.Errors) == 0
	return s
}

// RunError retains deferred startup failure in the observable state. The
// message must be a safe task-level description, never credential-bearing text.
func RunError(name string, start func() error) error {
	if !Configured() {
		return start()
	}
	processGate.register(name, start, nil)
	return nil
}

// RunManaged pairs a deferred task with cleanup when its ordinary provider has
// no Stop method. Normal mode preserves the provider's original lifecycle.
func RunManaged(name string, start, stop func()) {
	if !Configured() {
		start()
		return
	}
	processGate.register(name, func() error { start(); return nil }, stop)
}

func StopManaged() {
	processGate.mu.Lock()
	entries := make(map[string]*entry, len(processGate.entries))
	for name, e := range processGate.entries {
		entries[name] = e
	}
	processGate.mu.Unlock()
	for name, e := range entries {
		if e.cleanup != nil {
			processGate.Stop(name, e.cleanup)
		}
	}
}

// Run registers one task. Duplicate registration never runs a task twice.
// The task's own Start remains responsible for respecting its feature config.
func (g *Gate) Run(name string, start func()) {
	if name == "" || start == nil {
		panic("backgroundgate: task name and start callback are required")
	}
	g.register(name, func() error { start(); return nil }, nil)
}

func (g *Gate) register(name string, start func() error, cleanup func()) {
	if name == "" || start == nil {
		panic("backgroundgate: task name and start callback are required")
	}
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return
	}
	if _, exists := g.entries[name]; exists {
		g.mu.Unlock()
		return
	}
	e := &entry{start: start, cleanup: cleanup}
	g.entries[name] = e
	active := g.active
	g.mu.Unlock()
	if active {
		g.startEntry(name, e)
	}
}

func (g *Gate) startEntry(name string, e *entry) {
	g.lifecycle.RLock()
	defer g.lifecycle.RUnlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	g.mu.Lock()
	allowed := g.active && !g.closed
	g.mu.Unlock()
	if !allowed || e.started || e.stopped {
		return
	}
	if err := e.start(); err != nil {
		e.startError = "startup failed"
		slog.Error("background_gate_task_failed", "task", name)
	}
	e.started = true
	slog.Info("background_gate_task_started", "task", name)
}

// Stop serializes against an in-progress start and skips workers never started.
// It is safe to call twice, including before the marker exists.
func (g *Gate) Stop(name string, stop func()) {
	g.mu.Lock()
	e := g.entries[name]
	if e == nil {
		e = &entry{stopped: true}
		g.entries[name] = e
		g.mu.Unlock()
		return
	}
	g.mu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stopped {
		return
	}
	e.stopped = true
	if e.started && stop != nil {
		stop()
	}
}

type marker struct {
	ReleaseID string `json:"release_id"`
	Activate  bool   `json:"activate"`
}

// validateMarker rejects partial writes, symlinks, wrong releases and markers
// writable by other Unix users. Publish by atomic rename in a protected dir.
func (g *Gate) validateMarker() error {
	info, err := os.Lstat(g.path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > maxMarkerBytes {
		return errors.New("activation marker must be a small regular file")
	}
	dir := filepath.Dir(g.path)
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return err
	}
	if filepath.Clean(resolved) != dir {
		return errors.New("activation marker parent must not contain symlinks")
	}
	parent, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if runtime.GOOS != "windows" && (info.Mode().Perm()&0077 != 0 || parent.Mode().Perm()&0022 != 0) {
		return errors.New("activation marker must be private and its parent must not be group or world writable")
	}
	f, err := os.Open(g.path)
	if err != nil {
		return err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return errors.New("activation marker changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxMarkerBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxMarkerBytes {
		return errors.New("activation marker too large")
	}
	current, err := os.Lstat(g.path)
	if err != nil {
		return err
	}
	if !os.SameFile(opened, current) || current.Size() != opened.Size() || !current.ModTime().Equal(opened.ModTime()) {
		return errors.New("activation marker changed while reading")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var m marker
	if err := dec.Decode(&m); err != nil {
		return fmt.Errorf("invalid activation marker: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return errors.New("activation marker contains trailing data")
	}
	if m.ReleaseID != g.releaseID || !m.Activate {
		return errors.New("activation marker does not authorize this release")
	}
	return nil
}

// checkActivation is intentionally private: activation always requires a valid
// on-disk marker, never an HTTP endpoint or an in-process force-enable method.
func (g *Gate) checkActivation() error {
	g.mu.Lock()
	if g.active || g.closed || !g.sealed {
		g.mu.Unlock()
		return nil
	}
	g.mu.Unlock()
	if err := g.validateMarker(); err != nil {
		return err
	}
	g.mu.Lock()
	if g.active || g.closed || !g.sealed {
		g.mu.Unlock()
		return nil
	}
	g.active = true
	names := make([]string, 0, len(g.entries))
	for name := range g.entries {
		names = append(names, name)
	}
	sort.Strings(names)
	entries := make([]*entry, len(names))
	for i, name := range names {
		entries[i] = g.entries[name]
	}
	g.mu.Unlock()
	slog.Info("background_gate_activated", "release_id", g.releaseID, "registered_tasks", len(names))
	for i, e := range entries {
		g.startEntry(names[i], e)
	}
	slog.Info("background_gate_activation_complete", "release_id", g.releaseID, "registered_tasks", len(names))
	return nil
}

func (g *Gate) Watch() {
	g.watchOnce.Do(func() {
		go func() {
			defer close(g.done)
			if g.path == "" {
				return
			}
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			lastError := ""
			for {
				if err := g.checkActivation(); err != nil && !os.IsNotExist(err) && err.Error() != lastError {
					lastError = err.Error()
					slog.Warn("background_gate_marker_rejected", "reason", lastError)
				}
				select {
				case <-g.stop:
					return
				case <-ticker.C:
				}
			}
		}()
	})
}

// Close prevents all future starts. Existing workers are stopped by their
// ordinary application cleanup callbacks, after HTTP requests have drained.
func (g *Gate) Close() {
	g.closeOnce.Do(func() {
		// Wait for any start callback already in progress before cleanup begins.
		g.lifecycle.Lock()
		g.mu.Lock()
		g.closed = true
		g.mu.Unlock()
		g.lifecycle.Unlock()
		close(g.stop)
	})
	g.Watch()
	<-g.done
}
