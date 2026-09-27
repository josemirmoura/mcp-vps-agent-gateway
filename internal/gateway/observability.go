package gateway

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

type toolMetric struct {
	Calls        uint64 `json:"calls"`
	Successes    uint64 `json:"successes"`
	Failures     uint64 `json:"failures"`
	Denies       uint64 `json:"denies"`
	LatencyNanos uint64 `json:"latency_nanos_total"`
}

type runtimeMetrics struct {
	mu           sync.Mutex
	started      time.Time
	httpRequests uint64
	httpFailures uint64
	authFailures uint64
	tools        map[string]toolMetric
}

func newRuntimeMetrics() *runtimeMetrics {
	return &runtimeMetrics{started: time.Now().UTC(), tools: make(map[string]toolMetric)}
}

func (m *runtimeMetrics) observeTool(tool string, started time.Time, err error) {
	m.mu.Lock()
	tm := m.tools[tool]
	tm.Calls++
	tm.LatencyNanos += uint64(time.Since(started))
	if err == nil {
		tm.Successes++
	} else {
		tm.Failures++
		if isPolicyDeny(err) { tm.Denies++ }
	}
	m.tools[tool] = tm
	m.mu.Unlock()
	level := slog.LevelInfo
	if err != nil { level = slog.LevelWarn }
	slog.Log(nil, level, "mcp_tool", "tool", tool, "duration_ms", time.Since(started).Milliseconds(), "ok", err == nil)
}

func isPolicyDeny(err error) bool {
	if err == nil { return false }
	s := err.Error()
	for _, prefix := range []string{"permission_denied:", "identity_mismatch:", "grant_required:", "grant_invalid:", "full_disabled:", "admin_required:"} {
		if strings.HasPrefix(s, prefix) { return true }
	}
	return false
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (m *runtimeMetrics) wrapHTTP(instanceID, instanceName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		m.mu.Lock()
		m.httpRequests++
		if sw.status >= 400 { m.httpFailures++ }
		if sw.status == http.StatusUnauthorized || sw.status == http.StatusForbidden { m.authFailures++ }
		m.mu.Unlock()
		level := slog.LevelInfo
		if sw.status >= 400 { level = slog.LevelWarn }
		slog.Log(r.Context(), level, "http_request",
			"component", "gateway", "instance_id", instanceID, "instance_name", instanceName,
			"method", r.Method, "path", r.URL.Path, "status", sw.status, "duration_ms", time.Since(started).Milliseconds())
	})
}

func (m *runtimeMetrics) handler(instanceID, instanceName string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		m.mu.Lock()
		tools := make(map[string]toolMetric, len(m.tools))
		for k, v := range m.tools { tools[k] = v }
		body := map[string]any{
			"instance_id": instanceID, "instance_name": instanceName,
			"uptime_seconds": int64(time.Since(m.started).Seconds()),
			"http_requests_total": m.httpRequests, "http_failures_total": m.httpFailures,
			"auth_failures_total": m.authFailures, "tools": tools,
		}
		m.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	})
}
