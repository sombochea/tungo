package introspect

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/**/*
var staticFS embed.FS

// Dashboard manages the introspection web interface
type Dashboard struct {
	addr      string
	templates *template.Template
	server    *http.Server
}

// NewDashboard creates a new dashboard server
func NewDashboard(port int) (*Dashboard, error) {
	addr := fmt.Sprintf("0.0.0.0:%d", port)

	// Parse templates with custom functions
	funcMap := template.FuncMap{
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	d := &Dashboard{
		addr:      addr,
		templates: tmpl,
	}

	// Setup HTTP server
	mux := http.NewServeMux()

	// Routes
	mux.HandleFunc("/", d.handleIndex)
	mux.HandleFunc("/detail/", d.handleDetail)
	mux.HandleFunc("/replay/", d.handleReplay)
	mux.HandleFunc("/api/requests", d.handleAPIRequests)
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))

	d.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return d, nil
}

// Start starts the dashboard server
func (d *Dashboard) Start() error {
	log.Info().Str("addr", d.addr).Msg("Starting introspection dashboard")
	fmt.Printf("\n📊 Dashboard: http://localhost%s\n\n", strings.TrimPrefix(d.addr, "0.0.0.0"))

	if err := d.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("dashboard server error: %w", err)
	}
	return nil
}

// Stop stops the dashboard server
func (d *Dashboard) Stop() error {
	if d.server != nil {
		return d.server.Close()
	}
	return nil
}

// handleIndex displays the list of requests
func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	requests := GetStore().GetAll()

	// Sort by completion time (most recent first)
	sort.Slice(requests, func(i, j int) bool {
		return requests[i].Completed.After(requests[j].Completed)
	})

	data := map[string]interface{}{
		"Requests": requests,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := d.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Error().Err(err).Msg("Failed to render index template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleDetail displays details of a specific request
func (d *Dashboard) handleDetail(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	id := strings.TrimPrefix(r.URL.Path, "/detail/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	req, ok := GetStore().Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"Request":  req,
		"Incoming": parseBodyData(req.BodyData),
		"Response": parseBodyData(req.ResponseData),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := d.templates.ExecuteTemplate(w, "detail.html", data); err != nil {
		log.Error().Err(err).Msg("Failed to render detail template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleReplay replays a captured request (placeholder for future implementation)
func (d *Dashboard) handleReplay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	id := strings.TrimPrefix(r.URL.Path, "/replay/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	req, ok := GetStore().Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	// TODO: Implement replay functionality
	// For now, just redirect back to home
	log.Info().Str("id", req.ID).Str("path", req.Path).Msg("Replay request (not yet implemented)")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleAPIRequests returns requests as JSON
func (d *Dashboard) handleAPIRequests(w http.ResponseWriter, r *http.Request) {
	requests := GetStore().GetAll()

	// Sort by completion time (most recent first)
	sort.Slice(requests, func(i, j int) bool {
		return requests[i].Completed.After(requests[j].Completed)
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(requests)
}

// BodyData represents parsed body data for display
type BodyData struct {
	DataType string
	Content  string
	Raw      string
}

// parseBodyData attempts to parse body data (JSON, etc.)
func parseBodyData(data []byte) BodyData {
	body := BodyData{
		DataType: "unknown",
		Raw:      string(data),
	}

	if len(data) == 0 {
		body.Raw = ""
		return body
	}

	// Try to parse as JSON
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err == nil {
		body.DataType = "json"
		if formatted, err := json.MarshalIndent(jsonData, "", "  "); err == nil {
			body.Content = string(formatted)
		}
	}

	return body
}
