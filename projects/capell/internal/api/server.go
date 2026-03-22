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
	db  *sql.DB
	mux *http.ServeMux
}

// NewServer creates a new API server with all routes registered.
func NewServer(db *sql.DB) *Server {
	s := &Server{db: db, mux: http.NewServeMux()}
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

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", cors(s.handleHealth))
	s.mux.HandleFunc("GET /api/stats", cors(s.handleStats))
	s.mux.HandleFunc("GET /api/attributions", cors(s.handleAttributions))
	s.mux.HandleFunc("GET /api/works", cors(s.handleWorks))
	s.mux.HandleFunc("GET /api/works/{id}/editions", cors(s.handleEditions))
	s.mux.HandleFunc("GET /api/works/{id}/toc", cors(s.handleWorkTOC))
	s.mux.HandleFunc("GET /api/search", cors(s.handleSearch))
	s.mux.HandleFunc("GET /api/lexicon/letters", cors(s.handleLexiconLetters))
	s.mux.HandleFunc("GET /api/lexicon/entry/{id}", cors(s.handleLexiconEntry))
	s.mux.HandleFunc("GET /api/text/scene/{workId}/{act}/{scene}", cors(s.handleScene))
	s.mux.HandleFunc("GET /api/corrections", cors(s.handleCorrections))
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
