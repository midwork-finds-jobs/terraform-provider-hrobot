package hrobot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFirewallService_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/321" {
			t.Errorf("expected path '/firewall/321', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := map[string]interface{}{
			"firewall": map[string]interface{}{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"status":        "active",
				"whitelist_hos": true,
				"port":          "main",
				"rules": map[string]interface{}{
					"input": []map[string]interface{}{
						{
							"name":       "allow ssh",
							"ip_version": "ipv4",
							"action":     "accept",
							"protocol":   "tcp",
							"dst_port":   "22",
						},
					},
					"output": []map[string]interface{}{},
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

	config, err := client.Firewall.Get(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Firewall.Get returned error: %v", err)
	}

	if config.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", config.ServerNumber)
	}

	if config.Status != FirewallStatusActive {
		t.Errorf("expected status 'active', got '%s'", config.Status)
	}

	if !config.WhitelistHOS {
		t.Error("expected whitelist_hos to be true")
	}

	if len(config.Rules.Input) != 1 {
		t.Errorf("expected 1 input rule, got %d", len(config.Rules.Input))
	}

	if config.Rules.Input[0].Name != "allow ssh" {
		t.Errorf("expected rule name 'allow ssh', got '%s'", config.Rules.Input[0].Name)
	}
}

func TestFirewallService_Activate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/321" {
			t.Errorf("expected path '/firewall/321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("status") != "active" {
			t.Errorf("expected status 'active', got '%s'", r.FormValue("status"))
		}

		response := map[string]interface{}{
			"firewall": map[string]interface{}{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"status":        "active",
				"whitelist_hos": false,
				"port":          "main",
				"rules": map[string]interface{}{
					"input":  []map[string]interface{}{},
					"output": []map[string]interface{}{},
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

	config, err := client.Firewall.Activate(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Firewall.Activate returned error: %v", err)
	}

	if config.Status != FirewallStatusActive {
		t.Errorf("expected status 'active', got '%s'", config.Status)
	}
}

func TestFirewallService_Disable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/321" {
			t.Errorf("expected path '/firewall/321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("status") != "disabled" {
			t.Errorf("expected status 'disabled', got '%s'", r.FormValue("status"))
		}

		response := map[string]interface{}{
			"firewall": map[string]interface{}{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"status":        "disabled",
				"whitelist_hos": false,
				"port":          "main",
				"rules": map[string]interface{}{
					"input":  []map[string]interface{}{},
					"output": []map[string]interface{}{},
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

	config, err := client.Firewall.Disable(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Firewall.Disable returned error: %v", err)
	}

	if config.Status != FirewallStatusDisabled {
		t.Errorf("expected status 'disabled', got '%s'", config.Status)
	}
}

func TestFirewallService_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/321" {
			t.Errorf("expected path '/firewall/321', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.Firewall.Delete(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Firewall.Delete returned error: %v", err)
	}
}

func TestFirewallService_Update(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/firewall/321" {
			t.Errorf("expected path '/firewall/321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		response := map[string]interface{}{
			"firewall": map[string]interface{}{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"status":        "active",
				"whitelist_hos": true,
				"port":          "main",
				"rules": map[string]interface{}{
					"input": []map[string]interface{}{
						{
							"name":       "allow http",
							"ip_version": "ipv4",
							"action":     "accept",
							"protocol":   "tcp",
							"dst_port":   "80",
						},
					},
					"output": []map[string]interface{}{},
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

	updateConfig := UpdateConfig{
		Status:       FirewallStatusActive,
		WhitelistHOS: true,
		Rules: FirewallRules{
			Input: []FirewallRule{
				{
					Name:      "allow http",
					IPVersion: IPv4,
					Action:    ActionAccept,
					Protocol:  ProtocolTCP,
					DestPort:  "80",
				},
			},
			Output: []FirewallRule{},
		},
	}

	config, err := client.Firewall.Update(ctx, ServerID(321), updateConfig)
	if err != nil {
		t.Fatalf("Firewall.Update returned error: %v", err)
	}

	if config.Status != FirewallStatusActive {
		t.Errorf("expected status 'active', got '%s'", config.Status)
	}

	if len(config.Rules.Input) != 1 {
		t.Errorf("expected 1 input rule, got %d", len(config.Rules.Input))
	}
}

func TestFirewallService_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		setupFunc  func(*Client, context.Context) error
	}{
		{
			name:       "Get not found",
			statusCode: http.StatusNotFound,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Firewall.Get(ctx, ServerID(321))
				return err
			},
		},
		{
			name:       "Activate unauthorized",
			statusCode: http.StatusUnauthorized,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Firewall.Activate(ctx, ServerID(321))
				return err
			},
		},
		{
			name:       "Delete error",
			statusCode: http.StatusInternalServerError,
			setupFunc: func(c *Client, ctx context.Context) error {
				return c.Firewall.Delete(ctx, ServerID(321))
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
