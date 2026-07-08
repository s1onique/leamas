package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test: Records() deep copies header maps - slice immutability
func TestRecordsDeepCopySliceImmutability(t *testing.T) {
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
		t.Error("Records() did not return a defensive copy of the slice")
	}
}

// Test: Records() deep copies header maps - request headers immutability
func TestRecordsDeepCopyRequestHeadersImmutability(t *testing.T) {
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
	req.Header.Set("X-Test-Header", "original-value")
	_, _ = http.DefaultClient.Do(req)

	records1 := p.Records()
	records2 := p.Records()

	// Modify header values in first copy.
	if len(records1) > 0 && records1[0].RequestHeaders != nil {
		records1[0].RequestHeaders["X-Test-Header"][0] = "MUTATED"
	}

	// Second copy should be unaffected.
	if len(records2) > 0 && records2[0].RequestHeaders != nil {
		if records2[0].RequestHeaders["X-Test-Header"] != nil {
			if len(records2[0].RequestHeaders["X-Test-Header"]) > 0 {
				if records2[0].RequestHeaders["X-Test-Header"][0] == "MUTATED" {
					t.Error("Records() did not deep copy RequestHeaders map")
				}
			}
		}
	}
}

// Test: Records() deep copies header maps - response headers immutability
func TestRecordsDeepCopyResponseHeadersImmutability(t *testing.T) {
	resetGlobalCounter()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Response-Test", "response-value")
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

	records1 := p.Records()
	records2 := p.Records()

	// Modify response header values in first copy.
	if len(records1) > 0 && records1[0].ResponseHeaders != nil {
		records1[0].ResponseHeaders["X-Response-Test"] = []string{"MUTATED"}
	}

	// Second copy should be unaffected.
	if len(records2) > 0 && records2[0].ResponseHeaders != nil {
		if records2[0].ResponseHeaders["X-Response-Test"] != nil {
			if len(records2[0].ResponseHeaders["X-Response-Test"]) > 0 {
				if records2[0].ResponseHeaders["X-Response-Test"][0] == "MUTATED" {
					t.Error("Records() did not deep copy ResponseHeaders map")
				}
			}
		}
	}
}
