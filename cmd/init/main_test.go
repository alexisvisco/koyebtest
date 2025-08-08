package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// Table-driven test for generateNginxConfig
func TestGenerateNginxConfig(t *testing.T) {
	tests := []struct {
		name        string
		isScript    bool
		expectedSub string
	}{
		{
			name:        "Script mode",
			isScript:    true,
			expectedSub: "fastcgi_param SCRIPT_FILENAME /app/wrapper.sh;",
		},
		{
			name:        "Static mode",
			isScript:    false,
			expectedSub: "try_files /output =404;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := generateNginxConfig(tt.isScript, &buf)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			content := buf.String()
			if !strings.Contains(content, tt.expectedSub) {
				t.Errorf("expected config to contain %q, got:\n%s", tt.expectedSub, content)
			}
		})
	}
}

// Table-driven test for downloadFromURL
func TestDownloadFromURL(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		expectError    bool
	}{
		{
			name:           "Valid response",
			serverResponse: "hello world",
			statusCode:     http.StatusOK,
			expectError:    false,
		},
		{
			name:           "404 response",
			serverResponse: "not found",
			statusCode:     http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				io.WriteString(w, tt.serverResponse)
			}))
			defer ts.Close()

			parsedURL, _ := url.Parse(ts.URL)
			var buf bytes.Buffer

			err := downloadFromURL(parsedURL, &buf)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectError && buf.String() != tt.serverResponse {
				t.Errorf("unexpected response body: got %q, want %q", buf.String(), tt.serverResponse)
			}
		})
	}
}

// Test createCGIWrapper creates the correct file with expected content
func TestCreateCGIWrapper(t *testing.T) {
	// Cleanup
	defer os.Remove("wrapper.sh")

	err := createCGIWrapper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile("wrapper.sh")
	if err != nil {
		t.Fatalf("failed to read wrapper.sh: %v", err)
	}

	expected := "#!/bin/sh\necho \"Content-Type: text/plain\"\necho \"\"\n/bin/sh /app/output 2>&1"
	if !strings.Contains(string(data), expected) {
		t.Errorf("wrapper.sh missing expected script call: got\n%s", string(data))
	}
}
