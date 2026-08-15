package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func Test_requestLogger(t *testing.T) {
	logBuffer := &bytes.Buffer{}

	logger := slog.New(slog.NewJSONHandler(logBuffer, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Time(slog.TimeKey, time.Date(2023, 10, 1, 12, 34, 57, 0, time.UTC))
			}
			return a
		},
	}))

	requestLoggerMiddleware := requestLogger(logger)
	s := &server{}
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		if _, err := io.WriteString(w, "response"); err != nil {
			t.Fatalf("failed to write response body: %v", err)
		}
	})
	loggedHandler := requestLoggerMiddleware(s.authMiddleware(dummyHandler))

	req := httptest.NewRequest("POST", "http://lin.ko/api/stats", strings.NewReader("request"))
	req.SetBasicAuth("frodo", "ofTheNineFingers")
	rr := httptest.NewRecorder()
	loggedHandler.ServeHTTP(rr, req)

	const expectedStatusCode = http.StatusCreated
	if rr.Code != expectedStatusCode {
		t.Errorf("expected status code %d, got %d", expectedStatusCode, rr.Code)
	}

	var logEntry struct {
		Message           string `json:"msg"`
		Method            string `json:"method"`
		Path              string `json:"path"`
		ClientIP          string `json:"client_ip"`
		User              string `json:"user"`
		Duration          *int64 `json:"duration"`
		RequestBodyBytes  int    `json:"request_body_bytes"`
		ResponseStatus    int    `json:"response_status"`
		ResponseBodyBytes int    `json:"response_body_bytes"`
	}
	if err := json.Unmarshal(logBuffer.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to decode log entry: %v", err)
	}

	if logEntry.Message != "Served request" {
		t.Errorf("expected message %q, got %q", "Served request", logEntry.Message)
	}
	if logEntry.Method != http.MethodPost {
		t.Errorf("expected method %q, got %q", http.MethodPost, logEntry.Method)
	}
	if logEntry.Path != "/api/stats" {
		t.Errorf("expected path %q, got %q", "/api/stats", logEntry.Path)
	}
	if logEntry.ClientIP != "192.0.2.1:1234" {
		t.Errorf("expected client IP %q, got %q", "192.0.2.1:1234", logEntry.ClientIP)
	}
	if logEntry.User != "frodo" {
		t.Errorf("expected user %q, got %q", "frodo", logEntry.User)
	}
	if logEntry.Duration == nil || *logEntry.Duration < 0 {
		t.Errorf("expected a non-negative duration, got %v", logEntry.Duration)
	}
	if logEntry.RequestBodyBytes != len("request") {
		t.Errorf("expected %d request body bytes, got %d", len("request"), logEntry.RequestBodyBytes)
	}
	if logEntry.ResponseStatus != http.StatusCreated {
		t.Errorf("expected response status %d, got %d", http.StatusCreated, logEntry.ResponseStatus)
	}
	if logEntry.ResponseBodyBytes != len("response") {
		t.Errorf("expected %d response body bytes, got %d", len("response"), logEntry.ResponseBodyBytes)
	}
}
