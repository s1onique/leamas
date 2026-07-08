package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func resetGlobalCounter() {
	muID.Lock()
	counter = 0
	muID.Unlock()
}

// Test 1: New rejects empty upstream URL
func TestNewRejectsEmptyUpstreamURL(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"tab and spaces", "\t  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(Config{UpstreamURL: tt.value})
			if err == nil {
				t.Error("expected error for empty upstream URL, got nil")
			}
			if !strings.Contains(err.Error(), "upstream URL") {
				t.Errorf("expected 'upstream URL' in error, got: %v", err)
			}
		})
	}
}

// Test 2: New uses loopback/local default listen address
func TestNewUsesLoopbackDefaultListenAddr(t *testing.T) {
	resetGlobalCounter()
	p, err := New(Config{UpstreamURL: "http://example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.config.ListenAddr != "127.0.0.1:0" {
		t.Errorf("expected default listen addr '127.0.0.1:0', got: %s", p.config.ListenAddr)
	}
}

// Test 3: proxy forwards GET request to upstream
func TestProxyForwardsGETRequest(t *testing.T) {
	resetGlobalCounter()
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(Config{UpstreamURL: upstream.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proxy := httptest.NewServer(p.Handler())
	defer proxy.Close()

	resp, err := http.Get(proxy.URL)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if !upstreamCalled {
		t.Error("upstream server was not called")
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got: %d", resp.StatusCode)
	}
}

// Test 4: proxy preserves query string
func TestProxyPreservesQueryString(t *testing.T) {
	resetGlobalCounter()
	var receivedPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.RequestURI()
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(Config{UpstreamURL: upstream.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proxy := httptest.NewServer(p.Handler())
	defer proxy.Close()

	resp, err := http.Get(proxy.URL + "/api/test?foo=bar&baz=qux")
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if receivedPath != "/api/test?foo=bar&baz=qux" {
		t.Errorf("expected path '/api/test?foo=bar&baz=qux', got: %s", receivedPath)
	}
}

// Test: UpstreamURL validation rejects invalid URLs
func TestNewRejectsInvalidUpstreamURL(t *testing.T) {
	_, err := New(Config{UpstreamURL: "http://[invalid"})
	if err == nil {
		t.Error("expected error for invalid upstream URL, got nil")
	}
}
