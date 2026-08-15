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

	"boot.dev/linko/internal/store"
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
	loggedHandler := requestIDMiddleware(requestLoggerMiddleware(s.authMiddleware(dummyHandler)))

	req := httptest.NewRequest("POST", "http://lin.ko/api/stats", strings.NewReader("request"))
	req.SetBasicAuth("frodo", "ofTheNineFingers")
	req.Header.Set(requestIDHeader, "test-request-id")
	rr := httptest.NewRecorder()
	loggedHandler.ServeHTTP(rr, req)

	const expectedStatusCode = http.StatusCreated
	if rr.Code != expectedStatusCode {
		t.Errorf("expected status code %d, got %d", expectedStatusCode, rr.Code)
	}
	if requestID := rr.Header().Get(requestIDHeader); requestID != "test-request-id" {
		t.Errorf("expected response request ID %q, got %q", "test-request-id", requestID)
	}

	var logEntry struct {
		Message           string `json:"msg"`
		Method            string `json:"method"`
		Path              string `json:"path"`
		ClientIP          string `json:"client_ip"`
		RequestID         string `json:"request_id"`
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
	if logEntry.RequestID != "test-request-id" {
		t.Errorf("expected request ID %q, got %q", "test-request-id", logEntry.RequestID)
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

func Test_requestLoggerLogsHTTPErrors(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		path         string
		username     string
		password     string
		status       int
		errorMessage string
		wantStack    bool
	}{
		{
			name:         "not found",
			method:       http.MethodGet,
			path:         "/not-real",
			status:       http.StatusNotFound,
			errorMessage: "not found",
		},
		{
			name:      "password validation error",
			method:    http.MethodPost,
			path:      "/api/login",
			username:  "saruman",
			password:  "invalidPassword",
			status:    http.StatusInternalServerError,
			wantStack: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuffer := &bytes.Buffer{}
			logger := slog.New(slog.NewJSONHandler(logBuffer, &slog.HandlerOptions{
				ReplaceAttr: replaceAttr,
			}))
			st, err := store.New(t.TempDir(), logger)
			if err != nil {
				t.Fatalf("failed to create store: %v", err)
			}
			s := newServer(*st, 0, func() {}, logger)

			req := httptest.NewRequest(tt.method, "http://lin.ko"+tt.path, nil)
			if tt.username != "" {
				req.SetBasicAuth(tt.username, tt.password)
			}
			rr := httptest.NewRecorder()
			s.httpServer.Handler.ServeHTTP(rr, req)

			if rr.Code != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, rr.Code)
			}

			var logEntry struct {
				Message        string `json:"msg"`
				Path           string `json:"path"`
				RequestID      string `json:"request_id"`
				ResponseStatus int    `json:"response_status"`
				Error          *struct {
					Message    string `json:"message"`
					StackTrace string `json:"stack_trace"`
				} `json:"error"`
			}
			if err := json.Unmarshal(logBuffer.Bytes(), &logEntry); err != nil {
				t.Fatalf("failed to decode log entry: %v", err)
			}
			if logEntry.Message != "Served request" {
				t.Errorf("expected request log, got %q", logEntry.Message)
			}
			if logEntry.Path != tt.path {
				t.Errorf("expected path %q, got %q", tt.path, logEntry.Path)
			}
			if logEntry.RequestID == "" {
				t.Error("expected a generated request ID in the request log")
			}
			if logEntry.ResponseStatus != tt.status {
				t.Errorf("expected logged status %d, got %d", tt.status, logEntry.ResponseStatus)
			}
			if logEntry.Error == nil {
				t.Fatal("expected structured error in request log")
			}
			if tt.errorMessage != "" && logEntry.Error.Message != tt.errorMessage {
				t.Errorf("expected error message %q, got %q", tt.errorMessage, logEntry.Error.Message)
			}
			if gotStack := logEntry.Error.StackTrace != ""; gotStack != tt.wantStack {
				t.Errorf("expected stack trace presence %t, got %t", tt.wantStack, gotStack)
			}
		})
	}
}
