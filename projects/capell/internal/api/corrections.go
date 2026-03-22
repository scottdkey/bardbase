package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	githubRepo  = "scottdkey/bardbase"
	githubLabel = "correction"
)

type ghIssue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	HTMLURL   string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Labels    []ghLabel `json:"labels"`
	Body      string    `json:"body"`
}

type ghLabel struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type correctionIssue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Labels    []string  `json:"labels"`
	Body      string    `json:"body"`
}

func (s *Server) handleCorrections(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		state = "all"
	}

	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/issues?labels=%s&state=%s&per_page=100&sort=created&direction=desc",
		githubRepo, githubLabel, state,
	)

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build request")
		return
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "bardbase-api")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to reach GitHub")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		writeError(w, http.StatusBadGateway, fmt.Sprintf("GitHub returned %d: %s", resp.StatusCode, string(body)))
		return
	}

	var issues []ghIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse GitHub response")
		return
	}

	result := make([]correctionIssue, 0, len(issues))
	for _, i := range issues {
		labels := make([]string, 0, len(i.Labels))
		for _, l := range i.Labels {
			labels = append(labels, l.Name)
		}
		result = append(result, correctionIssue{
			Number:    i.Number,
			Title:     i.Title,
			State:     i.State,
			URL:       i.HTMLURL,
			CreatedAt: i.CreatedAt,
			UpdatedAt: i.UpdatedAt,
			Labels:    labels,
			Body:      i.Body,
		})
	}

	writeJSON(w, http.StatusOK, result)
}
