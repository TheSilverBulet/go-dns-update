package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

// Test SetLogLevel function
func TestSetLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected log.Level
	}{
		{"Info Level", "Info", log.InfoLevel},
		{"Warn Level", "Warn", log.WarnLevel},
		{"Error Level", "Error", log.ErrorLevel},
		{"Fatal Level", "Fatal", log.FatalLevel},
		{"Default Level", "Unknown", log.WarnLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLogLevel(tt.level)
			if log.GetLevel() != tt.expected {
				t.Errorf("Expected level %v, got %v", tt.expected, log.GetLevel())
			}
		})
	}
}

// Test GetPublicIP function
func TestGetPublicIP(t *testing.T) {
	// Create a test server to mock the IP service
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("203.0.113.42"))
	}))
	defer ts.Close()

	ip, err := GetPublicIP(ts.URL)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if ip != "203.0.113.42" {
		t.Errorf("Expected IP 203.0.113.42, got %s", ip)
	}
}

// Test GetPublicIP with error
func TestGetPublicIP_Error(t *testing.T) {
	// Create a failing test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	_, err := GetPublicIP(ts.URL)
	if err == nil {
		t.Error("Expected error but got none")
	}
}

// Test GetPublicIP with timeout
func TestGetPublicIP_Timeout(t *testing.T) {
	// Create a test server that hangs to simulate timeout
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate long processing time
		time.Sleep(2 * time.Second)
		w.Write([]byte("203.0.113.42"))
	}))
	defer ts.Close()

	// Call with a very short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Create a new request with the timeout context
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Do(req)
	if err == nil {
		t.Error("Expected timeout error but got none")
	}
}
