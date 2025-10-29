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

func TestFailoverService_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/failover" {
			t.Errorf("expected path '/failover', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		activeIP := "123.123.123.124"
		response := []map[string]interface{}{
			{
				"failover": map[string]interface{}{
					"ip":               "123.123.123.100",
					"netmask":          "255.255.255.255",
					"server_ip":        "123.123.123.123",
					"server_number":    321,
					"active_server_ip": activeIP,
				},
			},
			{
				"failover": map[string]interface{}{
					"ip":               "124.124.124.100",
					"netmask":          "255.255.255.255",
					"server_ip":        "124.124.124.124",
					"server_number":    456,
					"active_server_ip": nil,
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

	failovers, err := client.Failover.List(ctx)
	if err != nil {
		t.Fatalf("Failover.List returned error: %v", err)
	}

	if len(failovers) != 2 {
		t.Errorf("expected 2 failovers, got %d", len(failovers))
	}

	if failovers[0].IP != "123.123.123.100" {
		t.Errorf("expected IP '123.123.123.100', got '%s'", failovers[0].IP)
	}

	if failovers[0].ActiveServerIP == nil {
		t.Error("expected active_server_ip to be set")
	} else if *failovers[0].ActiveServerIP != "123.123.123.124" {
		t.Errorf("expected active_server_ip '123.123.123.124', got '%s'", *failovers[0].ActiveServerIP)
	}

	if failovers[1].ActiveServerIP != nil {
		t.Errorf("expected active_server_ip to be nil, got '%s'", *failovers[1].ActiveServerIP)
	}
}

func TestFailoverService_Get(t *testing.T) {
	tests := []struct {
		name           string
		ip             string
		activeServerIP *string
		wantPath       string
	}{
		{
			name:           "failover with active routing",
			ip:             "123.123.123.100",
			activeServerIP: stringPtr("123.123.123.124"),
			wantPath:       "/failover/123.123.123.100",
		},
		{
			name:           "failover without routing",
			ip:             "124.124.124.100",
			activeServerIP: nil,
			wantPath:       "/failover/124.124.124.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.wantPath {
					t.Errorf("expected path '%s', got '%s'", tt.wantPath, r.URL.Path)
				}

				if r.Method != "GET" {
					t.Errorf("expected GET request, got '%s'", r.Method)
				}

				response := map[string]interface{}{
					"failover": map[string]interface{}{
						"ip":               tt.ip,
						"netmask":          "255.255.255.255",
						"server_ip":        "123.123.123.123",
						"server_number":    321,
						"active_server_ip": tt.activeServerIP,
					},
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Fatalf("failed to encode response: %v", err)
				}
			}))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			failover, err := client.Failover.Get(ctx, tt.ip)
			if err != nil {
				t.Fatalf("Failover.Get returned error: %v", err)
			}

			if failover.IP != tt.ip {
				t.Errorf("expected IP '%s', got '%s'", tt.ip, failover.IP)
			}

			if tt.activeServerIP == nil && failover.ActiveServerIP != nil {
				t.Errorf("expected active_server_ip to be nil, got '%s'", *failover.ActiveServerIP)
			}

			if tt.activeServerIP != nil {
				if failover.ActiveServerIP == nil {
					t.Error("expected active_server_ip to be set")
				} else if *failover.ActiveServerIP != *tt.activeServerIP {
					t.Errorf("expected active_server_ip '%s', got '%s'", *tt.activeServerIP, *failover.ActiveServerIP)
				}
			}
		})
	}
}

func TestFailoverService_Update(t *testing.T) {
	tests := []struct {
		name           string
		ip             string
		activeServerIP string
	}{
		{
			name:           "route to primary server",
			ip:             "123.123.123.100",
			activeServerIP: "123.123.123.123",
		},
		{
			name:           "route to backup server",
			ip:             "123.123.123.100",
			activeServerIP: "123.123.123.124",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/failover/" + tt.ip
				if r.URL.Path != expectedPath {
					t.Errorf("expected path '%s', got '%s'", expectedPath, r.URL.Path)
				}

				if r.Method != "POST" {
					t.Errorf("expected POST request, got '%s'", r.Method)
				}

				if err := r.ParseForm(); err != nil {
					t.Fatalf("failed to parse form: %v", err)
				}

				if r.FormValue("active_server_ip") != tt.activeServerIP {
					t.Errorf("expected active_server_ip '%s', got '%s'", tt.activeServerIP, r.FormValue("active_server_ip"))
				}

				response := map[string]interface{}{
					"failover": map[string]interface{}{
						"ip":               tt.ip,
						"netmask":          "255.255.255.255",
						"server_ip":        "123.123.123.123",
						"server_number":    321,
						"active_server_ip": tt.activeServerIP,
					},
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Fatalf("failed to encode response: %v", err)
				}
			}))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			failover, err := client.Failover.Update(ctx, tt.ip, tt.activeServerIP)
			if err != nil {
				t.Fatalf("Failover.Update returned error: %v", err)
			}

			if failover.ActiveServerIP == nil {
				t.Error("expected active_server_ip to be set")
			} else if *failover.ActiveServerIP != tt.activeServerIP {
				t.Errorf("expected active_server_ip '%s', got '%s'", tt.activeServerIP, *failover.ActiveServerIP)
			}
		})
	}
}

func TestFailoverService_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/failover/123.123.123.100" {
			t.Errorf("expected path '/failover/123.123.123.100', got '%s'", r.URL.Path)
		}

		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.Failover.Delete(ctx, "123.123.123.100")
	if err != nil {
		t.Fatalf("Failover.Delete returned error: %v", err)
	}
}

func TestFailoverService_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		setupFunc  func(*Client, context.Context) error
	}{
		{
			name:       "List error",
			statusCode: http.StatusInternalServerError,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Failover.List(ctx)
				return err
			},
		},
		{
			name:       "Get not found",
			statusCode: http.StatusNotFound,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Failover.Get(ctx, "123.123.123.100")
				return err
			},
		},
		{
			name:       "Update unauthorized",
			statusCode: http.StatusUnauthorized,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Failover.Update(ctx, "123.123.123.100", "123.123.123.123")
				return err
			},
		},
		{
			name:       "Delete error",
			statusCode: http.StatusInternalServerError,
			setupFunc: func(c *Client, ctx context.Context) error {
				return c.Failover.Delete(ctx, "123.123.123.100")
			},
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

			err := tt.setupFunc(client, ctx)
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

// Helper function to create string pointer.
func stringPtr(s string) *string {
	return &s
}
