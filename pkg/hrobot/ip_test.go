package hrobot

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIPService_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ip" {
			t.Errorf("expected path '/ip', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := []map[string]interface{}{
			{
				"ip":               "123.123.123.123",
				"server_ip":        "123.123.123.123",
				"server_number":    321,
				"locked":           false,
				"traffic_warnings": true,
				"traffic_hourly":   1000,
				"traffic_daily":    50000,
				"traffic_monthly":  1000000,
			},
			{
				"ip":               "124.124.124.124",
				"server_ip":        "124.124.124.124",
				"server_number":    456,
				"locked":           false,
				"traffic_warnings": false,
				"traffic_hourly":   2000,
				"traffic_daily":    60000,
				"traffic_monthly":  1500000,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	ips, err := client.IP.List(ctx)
	if err != nil {
		t.Fatalf("IP.List returned error: %v", err)
	}

	if len(ips) != 2 {
		t.Errorf("expected 2 IPs, got %d", len(ips))
	}

	if ips[0].ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", ips[0].ServerNumber)
	}

	if !ips[0].TrafficWarnings {
		t.Error("expected traffic warnings to be enabled")
	}

	if ips[1].TrafficWarnings {
		t.Error("expected traffic warnings to be disabled")
	}
}

func TestIPService_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ip/123.123.123.123" {
			t.Errorf("expected path '/ip/123.123.123.123', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := map[string]interface{}{
			"ip": map[string]interface{}{
				"ip":               "123.123.123.123",
				"server_ip":        "123.123.123.123",
				"server_number":    321,
				"locked":           false,
				"separate_mac":     "00:50:56:00:00:01",
				"traffic_warnings": true,
				"traffic_hourly":   1000,
				"traffic_daily":    50000,
				"traffic_monthly":  1000000,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	ip := net.ParseIP("123.123.123.123")
	ipAddr, err := client.IP.Get(ctx, ip)
	if err != nil {
		t.Fatalf("IP.Get returned error: %v", err)
	}

	if ipAddr.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", ipAddr.ServerNumber)
	}

	if ipAddr.SeparateMac != "00:50:56:00:00:01" {
		t.Errorf("expected separate_mac '00:50:56:00:00:01', got '%s'", ipAddr.SeparateMac)
	}

	if !ipAddr.TrafficWarnings {
		t.Error("expected traffic warnings to be enabled")
	}
}

func TestIPService_SetTrafficWarnings(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "enable traffic warnings",
			enabled: true,
		},
		{
			name:    "disable traffic warnings",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/ip/123.123.123.123" {
					t.Errorf("expected path '/ip/123.123.123.123', got '%s'", r.URL.Path)
				}

				if r.Method != "POST" {
					t.Errorf("expected POST request, got '%s'", r.Method)
				}

				if err := r.ParseForm(); err != nil {
					t.Fatalf("failed to parse form: %v", err)
				}

				expectedValue := "false"
				if tt.enabled {
					expectedValue = "true"
				}

				if r.FormValue("traffic_warnings") != expectedValue {
					t.Errorf("expected traffic_warnings '%s', got '%s'", expectedValue, r.FormValue("traffic_warnings"))
				}

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			ip := net.ParseIP("123.123.123.123")
			err := client.IP.SetTrafficWarnings(ctx, ip, tt.enabled)
			if err != nil {
				t.Fatalf("IP.SetTrafficWarnings returned error: %v", err)
			}
		})
	}
}

func TestIPService_GetTraffic(t *testing.T) {
	tests := []struct {
		name        string
		trafficType string
		from        string
		to          string
		wantQuery   string
	}{
		{
			name:        "daily traffic",
			trafficType: "day",
			from:        "2024-01-01",
			to:          "2024-01-31",
			wantQuery:   "from=2024-01-01&to=2024-01-31&type=day",
		},
		{
			name:        "monthly traffic without dates",
			trafficType: "month",
			from:        "",
			to:          "",
			wantQuery:   "type=month",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/traffic/123.123.123.123" {
					t.Errorf("expected path '/traffic/123.123.123.123', got '%s'", r.URL.Path)
				}

				if r.Method != "GET" {
					t.Errorf("expected GET request, got '%s'", r.Method)
				}

				if r.URL.RawQuery != tt.wantQuery {
					t.Errorf("expected query '%s', got '%s'", tt.wantQuery, r.URL.RawQuery)
				}

				response := map[string]interface{}{
					"traffic": map[string]interface{}{
						"type": tt.trafficType,
						"data": []map[string]interface{}{
							{
								"timestamp": "2024-01-01 00:00:00",
								"in":        1000000,
								"out":       500000,
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

			ip := net.ParseIP("123.123.123.123")
			traffic, err := client.IP.GetTraffic(ctx, ip, tt.trafficType, tt.from, tt.to)
			if err != nil {
				t.Fatalf("IP.GetTraffic returned error: %v", err)
			}

			if traffic.Type != tt.trafficType {
				t.Errorf("expected type '%s', got '%s'", tt.trafficType, traffic.Type)
			}

			if len(traffic.Data) != 1 {
				t.Errorf("expected 1 data point, got %d", len(traffic.Data))
			}
		})
	}
}

func TestIPService_CancelIP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ip/123.123.123.123/cancellation" {
			t.Errorf("expected path '/ip/123.123.123.123/cancellation', got '%s'", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("cancellation_date") != "2024-12-31" {
			t.Errorf("expected cancellation_date '2024-12-31', got '%s'", r.FormValue("cancellation_date"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	ip := net.ParseIP("123.123.123.123")
	err := client.IP.CancelIP(ctx, ip, "2024-12-31")
	if err != nil {
		t.Fatalf("IP.CancelIP returned error: %v", err)
	}
}

func TestIPService_WithdrawIPCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ip/123.123.123.123/cancellation" {
			t.Errorf("expected path '/ip/123.123.123.123/cancellation', got '%s'", r.URL.Path)
		}

		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	ip := net.ParseIP("123.123.123.123")
	err := client.IP.WithdrawIPCancellation(ctx, ip)
	if err != nil {
		t.Fatalf("IP.WithdrawIPCancellation returned error: %v", err)
	}
}

func TestIPService_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		setupFunc  func(*Client, context.Context) error
	}{
		{
			name:       "List error",
			statusCode: http.StatusInternalServerError,
			setupFunc: func(c *Client, ctx context.Context) error {
				_, err := c.IP.List(ctx)
				return err
			},
		},
		{
			name:       "Get not found",
			statusCode: http.StatusNotFound,
			setupFunc: func(c *Client, ctx context.Context) error {
				ip := net.ParseIP("123.123.123.123")
				_, err := c.IP.Get(ctx, ip)
				return err
			},
		},
		{
			name:       "SetTrafficWarnings unauthorized",
			statusCode: http.StatusUnauthorized,
			setupFunc: func(c *Client, ctx context.Context) error {
				ip := net.ParseIP("123.123.123.123")
				return c.IP.SetTrafficWarnings(ctx, ip, true)
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
