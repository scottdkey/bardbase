package fetch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestURL_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("expected User-Agent %q, got %q", userAgent, r.Header.Get("User-Agent"))
		}
		w.Write([]byte("hello shakespeare"))
	}))
	defer server.Close()

	body, err := URL(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != "hello shakespeare" {
		t.Errorf("expected 'hello shakespeare', got %q", body)
	}
}

func TestURL_Retries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("success"))
	}))
	defer server.Close()

	body, err := URLWithRetries(server.URL, 3)
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}
	if body != "success" {
		t.Errorf("expected 'success', got %q", body)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestURL_AllRetriesFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := URLWithRetries(server.URL, 2)
	if err == nil {
		t.Error("expected error when all retries fail")
	}
}
