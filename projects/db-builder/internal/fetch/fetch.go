// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

// Package fetch provides HTTP utilities with retry logic for downloading source data.
package fetch

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultRetries = 3
	defaultTimeout = 30 * time.Second
	retryDelay     = 2 * time.Second
	userAgent      = "Shakespeare-DB-Builder/2.0 (academic research)"
)

// URL fetches the given URL and returns the response body as a string.
// Retries up to 3 times on failure with a 2-second delay between attempts.
func URL(url string) (string, error) {
	return URLWithRetries(url, defaultRetries)
}

// URLWithRetries fetches a URL with a configurable number of retries.
func URLWithRetries(url string, retries int) (string, error) {
	client := &http.Client{Timeout: defaultTimeout}

	var lastErr error
	for attempt := 0; attempt < retries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return "", fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < retries-1 {
				time.Sleep(retryDelay)
			}
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = err
			if attempt < retries-1 {
				time.Sleep(retryDelay)
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
			if attempt < retries-1 {
				time.Sleep(retryDelay)
			}
			continue
		}

		return string(body), nil
	}

	return "", fmt.Errorf("failed after %d attempts: %w", retries, lastErr)
}
