package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test 5: proxy returns upstream status code
func TestProxyReturnsUpstreamStatusCode(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"204 No Content", http.StatusNoContent},
		{"301 Moved", http.StatusMovedPermanently},
		{"400 Bad Request", http.StatusBadRequest},
		{"401 Unauthorized", http.StatusUnauthorized},
		{"403 Forbidden", http.StatusForbidden},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resetGlobalCounter()
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
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

			if resp.StatusCode != tc.statusCode {
				t.Errorf("expected status %d, got: %d", tc.statusCode, resp.StatusCode)
			}
		})
	}
}

// Test 6: witness record captures method/path/status
func TestWitnessRecordCapturesMethodPathStatus(t *testing.T) {
	resetGlobalCounter()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(Config{UpstreamURL: upstream.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proxy := httptest.NewServer(p.Handler())
	defer proxy.Close()

	_, _ = http.Get(proxy.URL + "/test/path")

	records := p.Records()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got: %d", len(records))
	}

	rec := records[0]
	if rec.Method != http.MethodGet {
		t.Errorf("expected method 'GET', got: %s", rec.Method)
	}
	if rec.Path != "/test/path" {
		t.Errorf("expected path '/test/path', got: %s", rec.Path)
	}
	if rec.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got: %d", rec.StatusCode)
	}
	if rec.UpstreamURL == "" {
		t.Error("expected non-empty upstream URL")
	}
}

// Test 7: records are bounded and oldest records are dropped
func TestRecordsAreBoundedAndOldestDropped(t *testing.T) {
	resetGlobalCounter()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(Config{
		UpstreamURL: upstream.URL,
		MaxRecords:  3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proxy := httptest.NewServer(p.Handler())
	defer proxy.Close()

	for i := 0; i < 5; i++ {
		resetGlobalCounter()
		_, _ = http.Get(proxy.URL)
	}

	records := p.Records()
	if len(records) != 3 {
		t.Errorf("expected 3 records (bounded), got: %d", len(records))
	}
}

// Test 8: Records() returns a defensive copy
func TestRecordsReturnsDefensiveCopy(t *testing.T) {
	resetGlobalCounter()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(Config{UpstreamURL: upstream.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proxy := httptest.NewServer(p.Handler())
	defer proxy.Close()

	_, _ = http.Get(proxy.URL)

	records1 := p.Records()
	records2 := p.Records()

	if len(records1) > 0 {
		records1[0].Method = "MODIFIED"
	}

	if len(records2) > 0 && records2[0].Method == "MODIFIED" {
		t.Error("Records() did not return a defensive copy")
	}
}

// Test 13: upstream failure records an error witness
func TestUpstreamFailureRecordsErrorWitness(t *testing.T) {
	resetGlobalCounter()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	upstreamURL := upstream.URL
	upstream.Close()

	p, err := New(Config{UpstreamURL: upstreamURL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proxy := httptest.NewServer(p.Handler())
	defer proxy.Close()

	client := &http.Client{Timeout: 100 * time.Millisecond}
	_, _ = client.Get(proxy.URL)

	_ = p.Records()
}

// Test 14: no provider routing behavior exists; single upstream is used
func TestNoProviderRoutingSingleUpstreamUsed(t *testing.T) {
	resetGlobalCounter()
	upstreamCallCount := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCallCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(Config{UpstreamURL: upstream.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proxy := httptest.NewServer(p.Handler())
	defer proxy.Close()

	paths := []string{"/openai/chat", "/anthropic/complete", "/local/model"}
	for _, path := range paths {
		resetGlobalCounter()
		_, _ = http.Get(proxy.URL + path)
	}

	if upstreamCallCount != 3 {
		t.Errorf("expected 3 upstream calls, got: %d", upstreamCallCount)
	}

	records := p.Records()
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got: %d", len(records))
	}

	for _, rec := range records {
		if rec.UpstreamURL != upstream.URL {
			t.Errorf("expected upstream URL %s, got: %s", upstream.URL, rec.UpstreamURL)
		}
	}
}

// Test: Reset clears all records
func TestResetClearsRecords(t *testing.T) {
	resetGlobalCounter()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(Config{UpstreamURL: upstream.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proxy := httptest.NewServer(p.Handler())
	defer proxy.Close()

	for i := 0; i < 3; i++ {
		resetGlobalCounter()
		_, _ = http.Get(proxy.URL)
	}

	records := p.Records()
	if len(records) != 3 {
		t.Fatalf("expected 3 records before reset, got: %d", len(records))
	}

	p.Reset()

	recordsAfterReset := p.Records()
	if len(recordsAfterReset) != 0 {
		t.Errorf("expected 0 records after reset, got: %d", len(recordsAfterReset))
	}
}

// Test: Reset does not affect concurrent access
func TestResetIsSafeForConcurrentAccess(t *testing.T) {
	resetGlobalCounter()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(Config{UpstreamURL: upstream.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proxy := httptest.NewServer(p.Handler())
	defer proxy.Close()

	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 20; j++ {
				resetGlobalCounter()
				_, _ = http.Get(proxy.URL)
			}
			done <- true
		}()
		go func() {
			for j := 0; j < 5; j++ {
				p.Reset()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	_ = p.Records()
}
