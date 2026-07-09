// Package cockpit provides a local-only, read-only web cockpit for reviewing
// Leamas status and static/demo evidence.
//
// This is not an enterprise admin UI, auth system, database app, live witness-proxy
// runtime, provider gateway, or model control plane.
//
// Security Notice: This cockpit is local-only and has no authentication.
// Do not expose it beyond loopback (127.0.0.1).
package cockpit

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

// Config holds the configuration for the cockpit server.
type Config struct {
	// ListenAddr is the address to listen on. Defaults to 127.0.0.1:0.
	ListenAddr string
}

// DefaultListenAddr is the default listen address.
const DefaultListenAddr = "127.0.0.1:0"

// Cockpit provides the web cockpit server.
type Cockpit struct {
	listenAddr string
}

// New creates a new Cockpit instance.
func New(config Config) (*Cockpit, error) {
	listenAddr := config.ListenAddr
	if listenAddr == "" {
		listenAddr = DefaultListenAddr
	}

	return &Cockpit{
		listenAddr: listenAddr,
	}, nil
}

// ListenAddr returns the configured listen address.
func (c *Cockpit) ListenAddr() string {
	return c.listenAddr
}

// StatusResponse represents the JSON response for /api/status.
type StatusResponse struct {
	Name     string `json:"name"`
	Mode     string `json:"mode"`
	ReadOnly bool   `json:"read_only"`
	Storage  string `json:"storage"`
	Auth     string `json:"auth"`
}

// ComponentStatus represents a component's status.
type ComponentStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

// ComponentsResponse represents the JSON response for /api/components.
type ComponentsResponse struct {
	Components []ComponentStatus `json:"components"`
}

// Handler returns an http.Handler for the cockpit server.
func (c *Cockpit) Handler() http.Handler {
	return http.HandlerFunc(c.serveHTTP)
}

// serveHTTP is the main request handler that routes based on path.
func (c *Cockpit) serveHTTP(w http.ResponseWriter, r *http.Request) {
	// Only allow GET for API routes
	switch r.URL.Path {
	case "/":
		c.handleIndex(w, r)
	case "/api/status":
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		c.handleStatus(w, r)
	case "/api/components":
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		c.handleComponents(w, r)
	default:
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			c.handleAssets(w, r)
		} else {
			http.NotFound(w, r)
		}
	}
}

// handleIndex serves the embedded HTML page.
func (c *Cockpit) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleStatus returns the system status as JSON.
func (c *Cockpit) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := StatusResponse{
		Name:     "Leamas Cockpit",
		Mode:     "local-only",
		ReadOnly: true,
		Storage:  "none",
		Auth:     "none",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(status); err != nil {
		// Already wrote status OK, cannot change now
		fmt.Fprint(w, "{}")
	}
}

// handleComponents returns the component status as JSON.
func (c *Cockpit) handleComponents(w http.ResponseWriter, r *http.Request) {
	components := ComponentsResponse{
		Components: []ComponentStatus{
			{
				Name:    "factory",
				Status:  "available",
				Summary: "Factory gates and digest workflow",
			},
			{
				Name:    "hulk",
				Status:  "seeded",
				Summary: "Typed run-bundle and claim/evidence cores",
			},
			{
				Name:    "witness",
				Status:  "seeded",
				Summary: "Local witness proxy package; runtime not started by cockpit",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(components); err != nil {
		// Already wrote status OK, cannot change now
		fmt.Fprint(w, "{}")
	}
}

// handleAssets serves static assets from the embedded filesystem.
func (c *Cockpit) handleAssets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/assets/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	data, err := staticFiles.ReadFile("static/" + path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Determine content type
	var contentType string
	switch {
	case strings.HasSuffix(path, ".css"):
		contentType = "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		contentType = "application/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".html"):
		contentType = "text/html; charset=utf-8"
	default:
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
