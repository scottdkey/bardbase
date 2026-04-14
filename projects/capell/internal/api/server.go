// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

// Package api implements the HTTP API for the bardbase Shakespeare database.
package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Server holds the database connection and HTTP handler.
type Server struct {
	db     *sql.DB
	mux    *http.ServeMux
	apiKey string
}

// NewServer creates a new API server with all routes registered.
// If apiKey is empty, authentication is disabled.
func NewServer(db *sql.DB, apiKey string) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), apiKey: apiKey}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
	s.mux.ServeHTTP(rw, r)
	log.Printf("%s %s %d %s", r.Method, r.URL.RequestURI(), rw.status, time.Since(start))
}

// responseWriter wraps http.ResponseWriter to capture the status code for logging.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// auth wraps a handler with API key validation.
// If no API key is configured, all requests pass through.
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.apiKey == "" {
			next(w, r)
			return
		}
		key := r.Header.Get("Authorization")
		if key == "Bearer "+s.apiKey {
			next(w, r)
			return
		}
		// Also accept X-API-Key header
		if r.Header.Get("X-API-Key") == s.apiKey {
			next(w, r)
			return
		}
		writeError(w, http.StatusUnauthorized, "invalid or missing API key")
	}
}

func (s *Server) routes() {
	// Health check + version — no auth required
	s.mux.HandleFunc("GET /health", cors(s.handleHealth))
	s.mux.HandleFunc("GET /api/version", cors(s.handleVersion))
	s.mux.HandleFunc("GET /api/stats", cors(s.auth(s.handleStats)))
	s.mux.HandleFunc("GET /api/attributions", cors(s.auth(s.handleAttributions)))
	s.mux.HandleFunc("GET /api/works", cors(s.auth(s.handleWorks)))
	s.mux.HandleFunc("GET /api/works/{id}/editions", cors(s.auth(s.handleEditions)))
	s.mux.HandleFunc("GET /api/works/{id}/toc", cors(s.auth(s.handleWorkTOC)))
	s.mux.HandleFunc("GET /api/search", cors(s.auth(s.handleSearch)))
	s.mux.HandleFunc("GET /api/lexicon/letters", cors(s.auth(s.handleLexiconLetters)))
	s.mux.HandleFunc("GET /api/lexicon/index", cors(s.auth(s.handleLexiconIndex)))
	s.mux.HandleFunc("GET /api/lexicon/entry/{id}", cors(s.auth(s.handleLexiconEntry)))
	s.mux.HandleFunc("GET /api/text/scene/{workId}/{act}/{scene}", cors(s.auth(s.handleScene)))
	s.mux.HandleFunc("GET /api/text/scene/{workId}/{act}/{scene}/references", cors(s.auth(s.handleSceneReferences)))
	s.mux.HandleFunc("GET /api/lexicon/keys", cors(s.auth(s.handleLexiconKeys)))
	s.mux.HandleFunc("GET /api/reference/entry/{id}", cors(s.auth(s.handleReferenceEntry)))
	s.mux.HandleFunc("GET /api/reference/index", cors(s.auth(s.handleReferenceIndex)))
	s.mux.HandleFunc("GET /api/reference/search", cors(s.auth(s.handleReferenceSearch)))
	s.mux.HandleFunc("GET /api/reference/sources", cors(s.auth(s.handleReferenceSources)))
	s.mux.HandleFunc("GET /api/resolve/{slug}", cors(s.auth(s.handleWorkBySlug)))
	s.mux.HandleFunc("GET /api/corrections", cors(s.auth(s.handleCorrections)))
}

// cors adds CORS headers to allow cross-origin requests.
func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// pathInt parses an integer from a path parameter.
func pathInt(r *http.Request, name string) (int, bool) {
	v := r.PathValue(name)
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

// resolveWorkID resolves a path parameter that can be either a numeric ID or a slug.
func (s *Server) resolveWorkID(r *http.Request, param string) (int, bool) {
	v := r.PathValue(param)
	// Try numeric first
	if n, err := strconv.Atoi(v); err == nil {
		return n, true
	}
	// Resolve as slug
	rows, err := s.db.Query(`SELECT id, title FROM works`)
	if err != nil {
		return 0, false
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			continue
		}
		if slugify(title) == v {
			return id, true
		}
	}
	return 0, false
}

// queryInt parses an optional integer query parameter with a default.
func queryInt(r *http.Request, name string, def int) int {
	v := r.URL.Query().Get(name)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// sanitizeFTS cleans a search query for FTS5 MATCH syntax.
func sanitizeFTS(q string) string {
	// Remove FTS operators that could cause syntax errors
	q = strings.TrimSpace(q)
	q = strings.Map(func(r rune) rune {
		if r == '"' || r == '*' || r == '-' || r == '+' {
			return -1
		}
		return r
	}, q)
	return q
}
