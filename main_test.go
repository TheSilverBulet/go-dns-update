package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

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

func TestGetPublicIP_Success(t *testing.T) {
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

func TestGetPublicIP_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	_, err := GetPublicIP(ts.URL)
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	// Verify it's specifically a 500 error
	if !strings.Contains(err.Error(), "server returned status: 500") {
		t.Errorf("Expected server error, got: %v", err)
	}
}

func TestGetPublicIP_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(6 * time.Second) // Longer than 5s timeout
		w.Write([]byte("203.0.113.42"))
	}))
	defer ts.Close()

	_, err := GetPublicIP(ts.URL)
	if err == nil {
		t.Error("Expected timeout error but got none")
	}
	// Check that it's specifically a timeout error
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestGetPublicIP_InvalidURL(t *testing.T) {
	_, err := GetPublicIP("http://invalid.url")
	if err == nil {
		t.Error("Expected error for invalid URL but got none")
	}
}
