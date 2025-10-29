package hrobot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestVSwitchService_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vswitch" {
			t.Errorf("expected path '/vswitch', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := []map[string]interface{}{
			{
				"id":        12345,
				"name":      "test-vswitch-1",
				"vlan":      4000,
				"cancelled": false,
			},
			{
				"id":        12346,
				"name":      "test-vswitch-2",
				"vlan":      4001,
				"cancelled": false,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	vswitches, err := client.VSwitch.List(ctx)
	if err != nil {
		t.Fatalf("VSwitch.List returned error: %v", err)
	}

	if len(vswitches) != 2 {
		t.Errorf("expected 2 vswitches, got %d", len(vswitches))
	}

	if vswitches[0].ID != 12345 {
		t.Errorf("expected ID 12345, got %d", vswitches[0].ID)
	}

	if vswitches[0].Name != "test-vswitch-1" {
		t.Errorf("expected name 'test-vswitch-1', got '%s'", vswitches[0].Name)
	}

	if vswitches[0].VLAN != 4000 {
		t.Errorf("expected VLAN 4000, got %d", vswitches[0].VLAN)
	}
}

func TestVSwitchService_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vswitch/12345" {
			t.Errorf("expected path '/vswitch/12345', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := map[string]interface{}{
			"vswitch": map[string]interface{}{
				"id":        12345,
				"name":      "test-vswitch",
				"vlan":      4000,
				"cancelled": false,
				"server": []map[string]interface{}{
					{
						"server_ip":     "123.123.123.123",
						"server_number": 321,
						"status":        "ready",
					},
				},
				"subnet": []map[string]interface{}{
					{
						"ip":      "10.0.0.0",
						"mask":    24,
						"gateway": "10.0.0.1",
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

	vswitch, err := client.VSwitch.Get(ctx, 12345)
	if err != nil {
		t.Fatalf("VSwitch.Get returned error: %v", err)
	}

	if vswitch.ID != 12345 {
		t.Errorf("expected ID 12345, got %d", vswitch.ID)
	}

	if vswitch.Name != "test-vswitch" {
		t.Errorf("expected name 'test-vswitch', got '%s'", vswitch.Name)
	}

	if len(vswitch.Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(vswitch.Servers))
	}

	if len(vswitch.Subnets) != 1 {
		t.Errorf("expected 1 subnet, got %d", len(vswitch.Subnets))
	}
}

func TestVSwitchService_Create(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vswitch" {
			t.Errorf("expected path '/vswitch', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("name") != "new-vswitch" {
			t.Errorf("expected name 'new-vswitch', got '%s'", r.FormValue("name"))
		}

		if r.FormValue("vlan") != "4000" {
			t.Errorf("expected vlan '4000', got '%s'", r.FormValue("vlan"))
		}

		response := map[string]interface{}{
			"vswitch": map[string]interface{}{
				"id":        12345,
				"name":      "new-vswitch",
				"vlan":      4000,
				"cancelled": false,
				"server":    []map[string]interface{}{},
				"subnet":    []map[string]interface{}{},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	vswitch, err := client.VSwitch.Create(ctx, "new-vswitch", 4000)
	if err != nil {
		t.Fatalf("VSwitch.Create returned error: %v", err)
	}

	if vswitch.Name != "new-vswitch" {
		t.Errorf("expected name 'new-vswitch', got '%s'", vswitch.Name)
	}

	if vswitch.VLAN != 4000 {
		t.Errorf("expected VLAN 4000, got %d", vswitch.VLAN)
	}
}

func TestVSwitchService_Update(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vswitch/12345" {
			t.Errorf("expected path '/vswitch/12345', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("name") != "updated-vswitch" {
			t.Errorf("expected name 'updated-vswitch', got '%s'", r.FormValue("name"))
		}

		if r.FormValue("vlan") != "4001" {
			t.Errorf("expected vlan '4001', got '%s'", r.FormValue("vlan"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.VSwitch.Update(ctx, 12345, "updated-vswitch", 4001)
	if err != nil {
		t.Fatalf("VSwitch.Update returned error: %v", err)
	}
}

func TestVSwitchService_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vswitch/12345" {
			t.Errorf("expected path '/vswitch/12345', got '%s'", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		// DeleteWithBody sends form data in body with DELETE method
		// Read body to parse form data
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}

		// Parse the form-encoded body manually
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("failed to parse query: %v", err)
		}

		cancDate := values.Get("cancellation_date")
		if cancDate != "2024-12-31" {
			t.Errorf("expected cancellation_date '2024-12-31', got '%s' (body: %s)", cancDate, string(body))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.VSwitch.Delete(ctx, 12345, "2024-12-31")
	if err != nil {
		t.Fatalf("VSwitch.Delete returned error: %v", err)
	}
}

func TestVSwitchService_AddServers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vswitch/12345/server" {
			t.Errorf("expected path '/vswitch/12345/server', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		servers := r.Form["server[]"]
		if len(servers) != 2 {
			t.Errorf("expected 2 servers, got %d", len(servers))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.VSwitch.AddServers(ctx, 12345, []string{"123.123.123.123", "124.124.124.124"})
	if err != nil {
		t.Fatalf("VSwitch.AddServers returned error: %v", err)
	}
}

func TestVSwitchService_RemoveServers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vswitch/12345/server" {
			t.Errorf("expected path '/vswitch/12345/server', got '%s'", r.URL.Path)
		}
		// RemoveServers uses PostRaw which sends as POST (see vswitch.go:177)
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		servers := r.Form["server[]"]
		if len(servers) != 1 {
			t.Errorf("expected 1 server, got %d", len(servers))
		}

		if len(servers) > 0 && servers[0] != "123.123.123.123" {
			t.Errorf("expected server '123.123.123.123', got '%s'", servers[0])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	err := client.VSwitch.RemoveServers(ctx, 12345, []string{"123.123.123.123"})
	if err != nil {
		t.Fatalf("VSwitch.RemoveServers returned error: %v", err)
	}
}

func TestVSwitchService_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		setupFunc  func(*Client, context.Context) error
	}{
		{
			name:       "List error",
			statusCode: http.StatusInternalServerError,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.VSwitch.List(ctx)
				return err
			},
		},
		{
			name:       "Get not found",
			statusCode: http.StatusNotFound,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.VSwitch.Get(ctx, 12345)
				return err
			},
		},
		{
			name:       "Create unauthorized",
			statusCode: http.StatusUnauthorized,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.VSwitch.Create(ctx, "test", 4000)
				return err
			},
		},
		{
			name:       "Update error",
			statusCode: http.StatusInternalServerError,
			setupFunc: func(c *Client, ctx context.Context) error {
				return c.VSwitch.Update(ctx, 12345, "test", 4000)
			},
		},
		{
			name:       "Delete error",
			statusCode: http.StatusInternalServerError,
			setupFunc: func(c *Client, ctx context.Context) error {
				return c.VSwitch.Delete(ctx, 12345, "2024-12-31")
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

func TestVSwitchService_IntegerConversion(t *testing.T) {
	// Test that integer parameters are properly converted to strings for form encoding
	tests := []struct {
		name string
		id   int
		vlan int
	}{
		{
			name: "small numbers",
			id:   123,
			vlan: 4000,
		},
		{
			name: "large numbers",
			id:   999999,
			vlan: 4095,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" && r.URL.Path == "/vswitch" {
					if err := r.ParseForm(); err != nil {
						t.Fatalf("failed to parse form: %v", err)
					}

					vlanStr := r.FormValue("vlan")
					if vlanStr != strconv.Itoa(tt.vlan) {
						t.Errorf("expected vlan '%d', got '%s'", tt.vlan, vlanStr)
					}

					response := map[string]interface{}{
						"vswitch": map[string]interface{}{
							"id":        tt.id,
							"name":      "test",
							"vlan":      tt.vlan,
							"cancelled": false,
						},
					}
					_ = json.NewEncoder(w).Encode(response)
				}
			}))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			vswitch, err := client.VSwitch.Create(ctx, "test", tt.vlan)
			if err != nil {
				t.Fatalf("VSwitch.Create returned error: %v", err)
			}

			if vswitch.VLAN != tt.vlan {
				t.Errorf("expected VLAN %d, got %d", tt.vlan, vswitch.VLAN)
			}
		})
	}
}
