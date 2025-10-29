// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hrobot

import (
	"encoding/json"
	"testing"
)

func TestUnwrapResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "array response",
			input:    `[{"id":1},{"id":2}]`,
			expected: `[{"id":1},{"id":2}]`,
		},
		{
			name:     "wrapped in server key",
			input:    `{"server":{"id":123,"name":"test"}}`,
			expected: `{"id":123,"name":"test"}`,
		},
		{
			name:     "wrapped in data key",
			input:    `{"data":{"id":123}}`,
			expected: `{"id":123}`,
		},
		{
			name:     "wrapped in firewall key",
			input:    `{"firewall":{"status":"active"}}`,
			expected: `{"status":"active"}`,
		},
		{
			name:     "no wrapper",
			input:    `{"id":123}`,
			expected: `{"id":123}`,
		},
		{
			name:     "empty object",
			input:    `{}`,
			expected: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := unwrapResponse([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Compare as normalized JSON to ignore whitespace differences
			var resultJSON, expectedJSON interface{}
			if err := json.Unmarshal(result, &resultJSON); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.expected), &expectedJSON); err != nil {
				t.Fatalf("failed to unmarshal expected: %v", err)
			}

			resultBytes, _ := json.Marshal(resultJSON)
			expectedBytes, _ := json.Marshal(expectedJSON)

			if string(resultBytes) != string(expectedBytes) {
				t.Errorf("unwrapResponse() = %s, want %s", string(resultBytes), string(expectedBytes))
			}
		})
	}
}

func TestUnwrapArrayResponse(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wrapperKey string
		expected   string
		wantErr    bool
	}{
		{
			name:       "wrapped servers",
			input:      `[{"server":{"id":1,"name":"s1"}},{"server":{"id":2,"name":"s2"}}]`,
			wrapperKey: "server",
			expected:   `[{"id":1,"name":"s1"},{"id":2,"name":"s2"}]`,
			wantErr:    false,
		},
		{
			name:       "wrapped ips",
			input:      `[{"ip":{"address":"1.2.3.4"}},{"ip":{"address":"5.6.7.8"}}]`,
			wrapperKey: "ip",
			expected:   `[{"address":"1.2.3.4"},{"address":"5.6.7.8"}]`,
			wantErr:    false,
		},
		{
			name:       "empty array",
			input:      `[]`,
			wrapperKey: "server",
			expected:   `[]`,
			wantErr:    false,
		},
		{
			name:       "single item",
			input:      `[{"server":{"id":1}}]`,
			wrapperKey: "server",
			expected:   `[{"id":1}]`,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := unwrapArrayResponse([]byte(tt.input), tt.wrapperKey)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unwrapArrayResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			// Compare as normalized JSON
			var resultJSON, expectedJSON interface{}
			if err := json.Unmarshal(result, &resultJSON); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.expected), &expectedJSON); err != nil {
				t.Fatalf("failed to unmarshal expected: %v", err)
			}

			resultBytes, _ := json.Marshal(resultJSON)
			expectedBytes, _ := json.Marshal(expectedJSON)

			if string(resultBytes) != string(expectedBytes) {
				t.Errorf("unwrapArrayResponse() = %s, want %s", string(resultBytes), string(expectedBytes))
			}
		})
	}
}

func TestClientOptions(t *testing.T) {
	t.Run("WithBaseURL", func(t *testing.T) {
		client := NewClient("user", "pass", WithBaseURL("https://custom.example.com/"))
		expected := "https://custom.example.com"
		if client.baseURL != expected {
			t.Errorf("baseURL = %s, want %s", client.baseURL, expected)
		}
	})

	t.Run("WithUserAgent", func(t *testing.T) {
		customUA := "custom-agent/1.0"
		client := NewClient("user", "pass", WithUserAgent(customUA))
		if client.userAgent != customUA {
			t.Errorf("userAgent = %s, want %s", client.userAgent, customUA)
		}
	})

	t.Run("default values", func(t *testing.T) {
		client := NewClient("user", "pass")
		if client.baseURL != DefaultBaseURL {
			t.Errorf("baseURL = %s, want %s", client.baseURL, DefaultBaseURL)
		}
		if client.userAgent != UserAgent {
			t.Errorf("userAgent = %s, want %s", client.userAgent, UserAgent)
		}
		if client.username != "user" {
			t.Errorf("username = %s, want user", client.username)
		}
		if client.password != "pass" {
			t.Errorf("password = %s, want pass", client.password)
		}
	})

	t.Run("services initialized", func(t *testing.T) {
		client := NewClient("user", "pass")
		if client.Server == nil {
			t.Error("Server service not initialized")
		}
		if client.Firewall == nil {
			t.Error("Firewall service not initialized")
		}
		if client.IP == nil {
			t.Error("IP service not initialized")
		}
		if client.Boot == nil {
			t.Error("Boot service not initialized")
		}
		if client.Reset == nil {
			t.Error("Reset service not initialized")
		}
	})

	t.Run("New alias works", func(t *testing.T) {
		client := New("user", "pass")
		if client == nil {
			t.Fatal("New() returned nil")
		}
		if client.username != "user" {
			t.Errorf("username = %s, want user", client.username)
		}
	})
}
