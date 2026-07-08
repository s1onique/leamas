package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test: ResponseHeaders are not captured when CaptureHeaders=false
func TestResponseHeadersNotCapturedWhenCaptureHeadersFalse(t *testing.T) {
	resetGlobalCounter()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Response-Header", "response-value")
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

	_, _ = http.Get(proxy.URL)

	records := p.Records()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got: %d", len(records))
	}

	if records[0].ResponseHeaders != nil {
		t.Error("expected nil ResponseHeaders when CaptureHeaders is false")
	}
}

// Test: CaptureHeaders=true captures both request and response headers
func TestCaptureHeadersCapturesBothRequestAndResponse(t *testing.T) {
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
	if records[0].ResponseHeaders["X-Response-Header"] == nil {
		t.Error("expected X-Response-Header in captured headers")
	}
}
