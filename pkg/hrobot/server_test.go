// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hrobot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerService_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/server" {
			t.Errorf("expected path '/server', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := []map[string]interface{}{
			{
				"server": map[string]interface{}{
					"server_ip":     "123.123.123.123",
					"server_number": 321,
					"server_name":   "server1",
					"product":       "EX41",
					"dc":            "FSN1-DC5",
					"traffic":       "unlimited",
					"status":        "ready",
					"cancelled":     false,
					"paid_until":    "2024-12-31",
					"ip":            []string{"123.123.123.123"},
					"subnet":        []map[string]interface{}{},
				},
			},
			{
				"server": map[string]interface{}{
					"server_ip":     "124.124.124.124",
					"server_number": 456,
					"server_name":   "server2",
					"product":       "AX41",
					"dc":            "NBG1-DC3",
					"traffic":       5368709120,
					"status":        "ready",
					"cancelled":     false,
					"paid_until":    "2024-11-30",
					"ip":            []string{"124.124.124.124"},
					"subnet":        []map[string]interface{}{},
				},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	servers, err := client.Server.List(ctx)
	if err != nil {
		t.Fatalf("Server.List returned error: %v", err)
	}

	if len(servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(servers))
	}

	if servers[0].ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", servers[0].ServerNumber)
	}

	if servers[0].ServerName != "server1" {
		t.Errorf("expected server name 'server1', got '%s'", servers[0].ServerName)
	}

	if servers[0].Product != "EX41" {
		t.Errorf("expected product 'EX41', got '%s'", servers[0].Product)
	}

	if !servers[0].Traffic.Unlimited {
		t.Errorf("expected unlimited traffic, got limited")
	}

	if servers[1].ServerNumber != 456 {
		t.Errorf("expected server number 456, got %d", servers[1].ServerNumber)
	}
}

func TestServerService_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/server/321" {
			t.Errorf("expected path '/server/321', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := map[string]interface{}{
			"server": map[string]interface{}{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"server_name":   "test-server",
				"product":       "EX41",
				"dc":            "FSN1-DC5",
				"traffic":       "unlimited",
				"status":        "ready",
				"cancelled":     false,
				"paid_until":    "2024-12-31",
				"ip":            []string{"123.123.123.123"},
				"subnet": []map[string]interface{}{
					{
						"ip":   "123.123.123.128",
						"mask": "255.255.255.192",
					},
				},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	srv, err := client.Server.Get(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Server.Get returned error: %v", err)
	}

	if srv.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", srv.ServerNumber)
	}

	if srv.ServerName != "test-server" {
		t.Errorf("expected server name 'test-server', got '%s'", srv.ServerName)
	}

	if srv.DC != "FSN1-DC5" {
		t.Errorf("expected DC 'FSN1-DC5', got '%s'", srv.DC)
	}

	if len(srv.Subnet) != 1 {
		t.Errorf("expected 1 subnet, got %d", len(srv.Subnet))
	}
}

func TestServerService_SetName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/server/321" {
			t.Errorf("expected path '/server/321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("server_name") != "new-name" {
			t.Errorf("expected server_name 'new-name', got '%s'", r.FormValue("server_name"))
		}

		response := map[string]interface{}{
			"server": map[string]interface{}{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"server_name":   "new-name",
				"product":       "EX41",
				"dc":            "FSN1-DC5",
				"traffic":       "unlimited",
				"status":        "ready",
				"cancelled":     false,
				"paid_until":    "2024-12-31",
				"ip":            []string{"123.123.123.123"},
				"subnet":        []map[string]interface{}{},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	srv, err := client.Server.SetName(ctx, ServerID(321), "new-name")
	if err != nil {
		t.Fatalf("Server.SetName returned error: %v", err)
	}

	if srv.ServerName != "new-name" {
		t.Errorf("expected server name 'new-name', got '%s'", srv.ServerName)
	}
}

func TestServerService_RequestCancellation(t *testing.T) {
	tests := []struct {
		name               string
		cancellation       Cancellation
		expectedDate       string
		expectedReason     string
		expectedReasonSent bool
	}{
		{
			name: "with reason",
			cancellation: Cancellation{
				ServerID:           ServerID(321),
				CancellationDate:   "2024-12-31",
				CancellationReason: "no longer needed",
			},
			expectedDate:       "2024-12-31",
			expectedReason:     "no longer needed",
			expectedReasonSent: true,
		},
		{
			name: "without reason",
			cancellation: Cancellation{
				ServerID:         ServerID(321),
				CancellationDate: "2024-12-31",
			},
			expectedDate:       "2024-12-31",
			expectedReasonSent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/server/321/cancellation" {
					t.Errorf("expected path '/server/321/cancellation', got '%s'", r.URL.Path)
				}
				if r.Method != "POST" {
					t.Errorf("expected POST request, got '%s'", r.Method)
				}

				if err := r.ParseForm(); err != nil {
					t.Fatalf("failed to parse form: %v", err)
				}

				if r.FormValue("cancellation_date") != tt.expectedDate {
					t.Errorf("expected cancellation_date '%s', got '%s'", tt.expectedDate, r.FormValue("cancellation_date"))
				}

				if tt.expectedReasonSent {
					if r.FormValue("cancellation_reason") != tt.expectedReason {
						t.Errorf("expected cancellation_reason '%s', got '%s'", tt.expectedReason, r.FormValue("cancellation_reason"))
					}
				}

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			err := client.Server.RequestCancellation(ctx, tt.cancellation)
			if err != nil {
				t.Fatalf("Server.RequestCancellation returned error: %v", err)
			}
		})
	}
}

func TestServerService_WithdrawCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/server/321/cancellation" {
			t.Errorf("expected path '/server/321/cancellation', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.Server.WithdrawCancellation(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Server.WithdrawCancellation returned error: %v", err)
	}
}

func TestServerService_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		method     string
	}{
		{
			name:       "List error",
			statusCode: http.StatusInternalServerError,
			method:     "list",
		},
		{
			name:       "Get not found",
			statusCode: http.StatusNotFound,
			method:     "get",
		},
		{
			name:       "SetName unauthorized",
			statusCode: http.StatusUnauthorized,
			method:     "setname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"status":  tt.statusCode,
						"code":    "ERROR",
						"message": "test error",
					},
				})
			}))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			var err error
			switch tt.method {
			case "list":
				_, err = client.Server.List(ctx)
			case "get":
				_, err = client.Server.Get(ctx, ServerID(321))
			case "setname":
				_, err = client.Server.SetName(ctx, ServerID(321), "new-name")
			}

			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}
