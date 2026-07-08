// Package proxy provides a local HTTP witness proxy that captures bounded
// request/response metadata without capturing body content by default.
package proxy

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

// SensitiveHeaders lists headers that contain credentials or sensitive data.
var SensitiveHeaders = []string{
	"Authorization",
	"Proxy-Authorization",
	"Cookie",
	"Set-Cookie",
	"X-Api-Key",
	"Api-Key",
}

// DefaultMaxRecords is the default maximum number of witness records to retain.
const DefaultMaxRecords = 100

// Config holds the configuration for a witness proxy.
type Config struct {
	// ListenAddr is the address on which the proxy listens.
	// Defaults to "127.0.0.1:0" if empty.
	ListenAddr string
	// UpstreamURL is the single target upstream URL.
	// Required; must not be empty.
	UpstreamURL string
	// MaxRecords is the maximum number of records to retain.
	// Defaults to 100 if <= 0.
	MaxRecords int
	// CaptureHeaders enables header capture in witness records.
	// When false, headers are not captured (default).
	CaptureHeaders bool
}

// WitnessRecord represents a captured proxy interaction.
type WitnessRecord struct {
	ID              string
	Method          string
	Path            string
	UpstreamURL     string
	RequestHeaders  map[string][]string
	ResponseHeaders map[string][]string
	StatusCode      int
	Error           string
	StartedAt       string
	CompletedAt     string
}

// WitnessProxy is a local HTTP proxy that captures witness evidence.
type WitnessProxy struct {
	config         Config
	upstreamURL    *url.URL
	records        []WitnessRecord
	mu             sync.Mutex
	sensitiveCheck map[string]bool
}

// New creates a new WitnessProxy from the given configuration.
// It validates that UpstreamURL is not empty and sets defaults.
func New(config Config) (*WitnessProxy, error) {
	if strings.TrimSpace(config.UpstreamURL) == "" {
		return nil, errors.New("upstream URL is required")
	}

	upstreamURL, err := url.Parse(config.UpstreamURL)
	if err != nil {
		return nil, errors.New("invalid upstream URL: " + err.Error())
	}

	if config.ListenAddr == "" {
		config.ListenAddr = "127.0.0.1:0"
	}

	if config.MaxRecords <= 0 {
		config.MaxRecords = DefaultMaxRecords
	}

	// Build sensitive header lookup map for O(1) checks.
	sensitiveMap := make(map[string]bool)
	for _, h := range SensitiveHeaders {
		sensitiveMap[strings.ToLower(h)] = true
	}

	return &WitnessProxy{
		config:         config,
		upstreamURL:    upstreamURL,
		records:        make([]WitnessRecord, 0, config.MaxRecords),
		sensitiveCheck: sensitiveMap,
	}, nil
}

// Handler returns an http.Handler that proxies requests to the configured upstream.
func (p *WitnessProxy) Handler() http.Handler {
	return http.HandlerFunc(p.serveHTTP)
}

// Records returns a defensive copy of all captured witness records.
func (p *WitnessProxy) Records() []WitnessRecord {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.records == nil {
		return nil
	}

	// Return a copy of the slice.
	copied := make([]WitnessRecord, len(p.records))
	copy(copied, p.records)
	return copied
}

// Reset clears all captured records.
func (p *WitnessProxy) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.records = p.records[:0]
}

// ListenAddr returns the configured listen address.
func (p *WitnessProxy) ListenAddr() string {
	return p.config.ListenAddr
}

// serveHTTP handles incoming HTTP requests and proxies them to the upstream.
func (p *WitnessProxy) serveHTTP(w http.ResponseWriter, r *http.Request) {
	record := WitnessRecord{
		ID:          generateID(),
		Method:      r.Method,
		Path:        r.URL.RequestURI(),
		UpstreamURL: p.upstreamURL.String(),
	}

	start := time.Now()
	defer func() {
		record.CompletedAt = start.Format(time.RFC3339Nano)
		p.addRecord(record)
	}()

	// Set started timestamp.
	record.StartedAt = start.Format(time.RFC3339Nano)

	// Capture request headers if enabled.
	if p.config.CaptureHeaders {
		record.RequestHeaders = p.sanitizeHeaders(r.Header)
	}

	// Create reverse proxy to the single upstream.
	proxy := httputil.NewSingleHostReverseProxy(p.upstreamURL)

	// Override director to preserve original method and path.
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Scheme = p.upstreamURL.Scheme
		req.URL.Host = p.upstreamURL.Host
		req.Host = p.upstreamURL.Host
	}

	// Capture response via wrapper.
	recorder := &responseCapture{ResponseWriter: w, statusCode: http.StatusOK}
	proxy.ServeHTTP(recorder, r)

	record.StatusCode = recorder.statusCode
	record.ResponseHeaders = p.sanitizeHeaders(recorder.Header())
}

// addRecord adds a record to the bounded ring buffer.
func (p *WitnessProxy) addRecord(record WitnessRecord) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.records = append(p.records, record)

	// Enforce max records by dropping oldest.
	if len(p.records) > p.config.MaxRecords {
		p.records = p.records[1:]
	}
}

// sanitizeHeaders redacts sensitive headers from the given headers.
// Returns a copy with sensitive values replaced by [REDACTED].
func (p *WitnessProxy) sanitizeHeaders(h http.Header) map[string][]string {
	if h == nil {
		return nil
	}

	result := make(map[string][]string)
	for key, values := range h {
		if p.sensitiveCheck[strings.ToLower(key)] {
			redacted := make([]string, len(values))
			for i := range values {
				redacted[i] = "[REDACTED]"
			}
			result[key] = redacted
		} else {
			// Copy non-sensitive values.
			result[key] = append([]string(nil), values...)
		}
	}
	return result
}

// responseCapture wraps http.ResponseWriter to capture the status code.
type responseCapture struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and delegates to the wrapped writer.
func (r *responseCapture) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// generateID generates a unique ID for a record.
// Using a simple counter + time for testability.
var (
	muID    sync.Mutex
	counter uint64
)

func generateID() string {
	muID.Lock()
	defer muID.Unlock()
	counter++
	return time.Now().Format("20060102150405.000000000") + "-" + string(rune('a'+counter%26))
}
