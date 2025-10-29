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

func TestResetService_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reset/321" {
			t.Errorf("expected path '/reset/321', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := map[string]interface{}{
			"reset": map[string]interface{}{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"type":          []string{"sw", "hw", "power"},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	reset, err := client.Reset.Get(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Reset.Get returned error: %v", err)
	}

	if reset.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", reset.ServerNumber)
	}

	if len(reset.Type) != 3 {
		t.Errorf("expected 3 reset types, got %d", len(reset.Type))
	}

	expectedTypes := []ResetType{ResetTypeSoftware, ResetTypeHardware, ResetTypePower}
	for i, expectedType := range expectedTypes {
		if i >= len(reset.Type) || reset.Type[i] != expectedType {
			t.Errorf("expected reset type %s at index %d, got %s", expectedType, i, reset.Type[i])
		}
	}
}

func TestResetService_Execute(t *testing.T) {
	tests := []struct {
		name      string
		resetType ResetType
	}{
		{
			name:      "software reset",
			resetType: ResetTypeSoftware,
		},
		{
			name:      "hardware reset",
			resetType: ResetTypeHardware,
		},
		{
			name:      "power reset",
			resetType: ResetTypePower,
		},
		{
			name:      "power long reset",
			resetType: ResetTypePowerLong,
		},
		{
			name:      "manual reset",
			resetType: ResetTypeManual,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/reset/321" {
					t.Errorf("expected path '/reset/321', got '%s'", r.URL.Path)
				}
				if r.Method != "POST" {
					t.Errorf("expected POST request, got '%s'", r.Method)
				}

				if err := r.ParseForm(); err != nil {
					t.Fatalf("failed to parse form: %v", err)
				}

				if r.FormValue("type") != string(tt.resetType) {
					t.Errorf("expected type '%s', got '%s'", tt.resetType, r.FormValue("type"))
				}

				response := map[string]interface{}{
					"reset": map[string]interface{}{
						"server_ip":     "123.123.123.123",
						"server_number": 321,
						"type":          []string{string(tt.resetType)},
					},
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Fatalf("failed to encode response: %v", err)
				}
			}))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			reset, err := client.Reset.Execute(ctx, ServerID(321), tt.resetType)
			if err != nil {
				t.Fatalf("Reset.Execute returned error: %v", err)
			}

			if reset.ServerNumber != 321 {
				t.Errorf("expected server number 321, got %d", reset.ServerNumber)
			}
		})
	}
}

func TestResetService_ExecuteSoftware(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reset/321" {
			t.Errorf("expected path '/reset/321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("type") != "sw" {
			t.Errorf("expected type 'sw', got '%s'", r.FormValue("type"))
		}

		response := map[string]interface{}{
			"reset": map[string]interface{}{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"type":          []string{"sw"},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	reset, err := client.Reset.ExecuteSoftware(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Reset.ExecuteSoftware returned error: %v", err)
	}

	if reset.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", reset.ServerNumber)
	}
}

func TestResetService_ExecuteHardware(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reset/321" {
			t.Errorf("expected path '/reset/321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("type") != "hw" {
			t.Errorf("expected type 'hw', got '%s'", r.FormValue("type"))
		}

		response := map[string]interface{}{
			"reset": map[string]interface{}{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"type":          []string{"hw"},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	reset, err := client.Reset.ExecuteHardware(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Reset.ExecuteHardware returned error: %v", err)
	}

	if reset.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", reset.ServerNumber)
	}
}

func TestResetService_ExecutePower(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reset/321" {
			t.Errorf("expected path '/reset/321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("type") != "power" {
			t.Errorf("expected type 'power', got '%s'", r.FormValue("type"))
		}

		response := map[string]interface{}{
			"reset": map[string]interface{}{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"type":          []string{"power"},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	reset, err := client.Reset.ExecutePower(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Reset.ExecutePower returned error: %v", err)
	}

	if reset.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", reset.ServerNumber)
	}
}

func TestResetService_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		method     string
	}{
		{
			name:       "Get not found",
			statusCode: http.StatusNotFound,
			method:     "get",
		},
		{
			name:       "Execute unauthorized",
			statusCode: http.StatusUnauthorized,
			method:     "execute",
		},
		{
			name:       "ExecuteSoftware error",
			statusCode: http.StatusInternalServerError,
			method:     "executesoftware",
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
			case "get":
				_, err = client.Reset.Get(ctx, ServerID(321))
			case "execute":
				_, err = client.Reset.Execute(ctx, ServerID(321), ResetTypeSoftware)
			case "executesoftware":
				_, err = client.Reset.ExecuteSoftware(ctx, ServerID(321))
			}

			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}
