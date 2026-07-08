package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test 9: default config does not capture headers
func TestDefaultConfigDoesNotCaptureHeaders(t *testing.T) {
	resetGlobalCounter()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(Config{
		UpstreamURL:    upstream.URL,
		CaptureHeaders: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proxy := httptest.NewServer(p.Handler())
	defer proxy.Close()

	req, _ := http.NewRequest(http.MethodGet, proxy.URL, nil)
	req.Header.Set("X-Custom-Header", "test-value")
	client := &http.Client{}
	client.Do(req)

	records := p.Records()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got: %d", len(records))
	}

	if records[0].RequestHeaders != nil {
		t.Error("expected nil RequestHeaders when CaptureHeaders is false")
	}
}

// Test 10: CaptureHeaders=true captures non-sensitive headers
func TestCaptureHeadersCapturesNonSensitiveHeaders(t *testing.T) {
	resetGlobalCounter()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Response-Header", "response-value")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(Config{
		UpstreamURL:    upstream.URL,
		CaptureHeaders: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proxy := httptest.NewServer(p.Handler())
	defer proxy.Close()

	req, _ := http.NewRequest(http.MethodGet, proxy.URL, nil)
	req.Header.Set("X-Request-Header", "request-value")
	client := &http.Client{}
	resp, _ := client.Do(req)
	resp.Body.Close()

	records := p.Records()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got: %d", len(records))
	}

	if records[0].RequestHeaders == nil {
		t.Fatal("expected non-nil RequestHeaders")
	}
	if records[0].RequestHeaders["X-Request-Header"] == nil {
		t.Error("expected X-Request-Header in captured headers")
	}
	if records[0].ResponseHeaders == nil {
		t.Fatal("expected non-nil ResponseHeaders")
	}
}

// Test 11: sensitive request headers are redacted
func TestSensitiveRequestHeadersAreRedacted(t *testing.T) {
	resetGlobalCounter()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(Config{
		UpstreamURL:    upstream.URL,
		CaptureHeaders: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proxy := httptest.NewServer(p.Handler())
	defer proxy.Close()

	req, _ := http.NewRequest(http.MethodGet, proxy.URL, nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Cookie", "session=abc123")
	req.Header.Set("X-Api-Key", "key-12345")
	client := &http.Client{}
	resp, _ := client.Do(req)
	resp.Body.Close()

	records := p.Records()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got: %d", len(records))
	}

	headers := records[0].RequestHeaders

	if val, ok := headers["Authorization"]; ok {
		for _, v := range val {
			if v != "[REDACTED]" {
				t.Errorf("Authorization header should be [REDACTED], got: %s", v)
			}
		}
	} else {
		t.Error("Authorization header should be present but redacted")
	}

	if val, ok := headers["Cookie"]; ok {
		for _, v := range val {
			if v != "[REDACTED]" {
				t.Errorf("Cookie header should be [REDACTED], got: %s", v)
			}
		}
	}

	if val, ok := headers["X-Api-Key"]; ok {
		for _, v := range val {
			if v != "[REDACTED]" {
				t.Errorf("X-Api-Key header should be [REDACTED], got: %s", v)
			}
		}
	}
}

// Test 12: sensitive response headers are redacted
func TestSensitiveResponseHeadersAreRedacted(t *testing.T) {
	resetGlobalCounter()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "session=abc123; HttpOnly")
		w.Header().Set("Proxy-Authorization", "Basic secret")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(Config{
		UpstreamURL:    upstream.URL,
		CaptureHeaders: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proxy := httptest.NewServer(p.Handler())
	defer proxy.Close()

	_, _ = http.Get(proxy.URL)

	records := p.Records()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got: %d", len(records))
	}

	headers := records[0].ResponseHeaders

	if val, ok := headers["Set-Cookie"]; ok {
		for _, v := range val {
			if v != "[REDACTED]" {
				t.Errorf("Set-Cookie header should be [REDACTED], got: %s", v)
			}
		}
	}

	if val, ok := headers["Proxy-Authorization"]; ok {
		for _, v := range val {
			if v != "[REDACTED]" {
				t.Errorf("Proxy-Authorization header should be [REDACTED], got: %s", v)
			}
		}
	}
}
