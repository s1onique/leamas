package cockpit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNew_DefaultsListenAddr(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got := c.ListenAddr(); got != DefaultListenAddr {
		t.Errorf("ListenAddr() = %q, want %q", got, DefaultListenAddr)
	}
}

func TestNew_CustomListenAddr(t *testing.T) {
	const want = "127.0.0.1:8080"
	c, err := New(Config{ListenAddr: want})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got := c.ListenAddr(); got != want {
		t.Errorf("ListenAddr() = %q, want %q", got, want)
	}
}

func TestHandler_GetRoot(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	c.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET / status = %d, want %d", rec.Code, http.StatusOK)
	}

	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("GET / Content-Type = %q, want text/html", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Leamas Cockpit") {
		t.Error("GET / response does not contain 'Leamas Cockpit'")
	}
}

func TestHandler_GetAPIStatus(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	c.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /api/status status = %d, want %d", rec.Code, http.StatusOK)
	}

	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("GET /api/status Content-Type = %q, want application/json", ct)
	}

	var status StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("GET /api/status response is not valid JSON: %v", err)
	}

	if status.Mode != "local-only" {
		t.Errorf("status.Mode = %q, want %q", status.Mode, "local-only")
	}

	if !status.ReadOnly {
		t.Error("status.ReadOnly = false, want true")
	}

	if status.Storage != "none" {
		t.Errorf("status.Storage = %q, want %q", status.Storage, "none")
	}

	if status.Auth != "none" {
		t.Errorf("status.Auth = %q, want %q", status.Auth, "none")
	}
}

func TestHandler_GetAPIComponents(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/components", nil)
	rec := httptest.NewRecorder()

	c.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /api/components status = %d, want %d", rec.Code, http.StatusOK)
	}

	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("GET /api/components Content-Type = %q, want application/json", ct)
	}

	var resp ComponentsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("GET /api/components response is not valid JSON: %v", err)
	}

	if len(resp.Components) != 3 {
		t.Fatalf("len(resp.Components) = %d, want 3", len(resp.Components))
	}

	found := map[string]bool{
		"factory": false,
		"hulk":    false,
		"witness": false,
	}
	for _, comp := range resp.Components {
		found[comp.Name] = true
	}
	for name, ok := range found {
		if !ok {
			t.Errorf("missing component: %s", name)
		}
	}
}

func TestHandler_UnknownRoute(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()

	c.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /unknown status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandler_NonGETAPIRoutes(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/status"},
		{http.MethodPut, "/api/status"},
		{http.MethodDelete, "/api/status"},
		{http.MethodPost, "/api/components"},
		{http.MethodPut, "/api/components"},
		{http.MethodDelete, "/api/components"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			c.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s status = %d, want %d", tt.method, tt.path, rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestHandler_NoSetCookie(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	c.Handler().ServeHTTP(rec, req)

	if cookie := rec.Header().Get("Set-Cookie"); cookie != "" {
		t.Errorf("GET / has Set-Cookie header: %q", cookie)
	}
}

func TestHandler_StaticAssets(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/assets/style.css", nil)
	rec := httptest.NewRecorder()

	c.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /assets/style.css status = %d, want %d", rec.Code, http.StatusOK)
	}

	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/css") {
		t.Errorf("GET /assets/style.css Content-Type = %q, want text/css", ct)
	}
}

func TestHandler_StaticAssetsEmbedded(t *testing.T) {
	// Verify that static assets work without filesystem at runtime
	// by reading directly from the embedded FS
	data, err := staticFiles.ReadFile("static/style.css")
	if err != nil {
		t.Fatalf("embedded static/style.css not found: %v", err)
	}

	if len(data) == 0 {
		t.Error("embedded static/style.css is empty")
	}
}

func TestHandler_IndexEmbedded(t *testing.T) {
	// Verify that index.html is embedded correctly
	data, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		t.Fatalf("embedded static/index.html not found: %v", err)
	}

	body := string(data)
	if !strings.Contains(body, "Leamas Cockpit") {
		t.Error("embedded static/index.html does not contain 'Leamas Cockpit'")
	}
}

func TestHandler_NonRootPaths404(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []string{
		"/home",
		"/admin",
		"/api",
		"/status",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			c.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("GET %s status = %d, want %d", path, rec.Code, http.StatusNotFound)
			}
		})
	}
}
