// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

func TestDetectIPVersion(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected hrobot.IPVersion
	}{
		{
			name:     "IPv4 address",
			ip:       "192.168.1.1",
			expected: hrobot.IPv4,
		},
		{
			name:     "IPv4 CIDR",
			ip:       "192.168.1.0/24",
			expected: hrobot.IPv4,
		},
		{
			name:     "IPv6 address",
			ip:       "2001:db8::1",
			expected: hrobot.IPv6,
		},
		{
			name:     "IPv6 CIDR",
			ip:       "2001:db8::/32",
			expected: hrobot.IPv6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectIPVersion(tt.ip)
			if result != tt.expected {
				t.Errorf("detectIPVersion(%s) = %s, expected %s", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsAutoAddedMailRule(t *testing.T) {
	tests := []struct {
		name     string
		rule     hrobot.FirewallRule
		expected bool
	}{
		{
			name: "auto-added mail rule",
			rule: hrobot.FirewallRule{
				Name:     "Block mail ports",
				Action:   hrobot.ActionDiscard,
				Protocol: hrobot.ProtocolTCP,
				DestPort: "25,465",
			},
			expected: true,
		},
		{
			name: "regular rule",
			rule: hrobot.FirewallRule{
				Name:     "Allow SSH",
				Action:   hrobot.ActionAccept,
				Protocol: hrobot.ProtocolTCP,
				DestPort: "22",
			},
			expected: false,
		},
		{
			name: "different mail port",
			rule: hrobot.FirewallRule{
				Name:     "Block mail ports",
				Action:   hrobot.ActionDiscard,
				Protocol: hrobot.ProtocolTCP,
				DestPort: "25",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAutoAddedMailRule(tt.rule)
			if result != tt.expected {
				t.Errorf("isAutoAddedMailRule() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFilterAutoAddedRules(t *testing.T) {
	rules := []hrobot.FirewallRule{
		{
			Name:     "Allow SSH",
			Action:   hrobot.ActionAccept,
			Protocol: hrobot.ProtocolTCP,
			DestPort: "22",
		},
		{
			Name:     "Block mail ports",
			Action:   hrobot.ActionDiscard,
			Protocol: hrobot.ProtocolTCP,
			DestPort: "25,465",
		},
		{
			Name:     "Allow HTTPS",
			Action:   hrobot.ActionAccept,
			Protocol: hrobot.ProtocolTCP,
			DestPort: "443",
		},
	}

	filtered := filterAutoAddedRules(rules)

	if len(filtered) != 2 {
		t.Errorf("expected 2 rules after filtering, got %d", len(filtered))
	}

	for _, rule := range filtered {
		if rule.Name == "Block mail ports" {
			t.Error("mail blocking rule should have been filtered out")
		}
	}
}

func TestRuleExists(t *testing.T) {
	existingRules := []hrobot.FirewallRule{
		{
			Name:      "Allow SSH",
			IPVersion: hrobot.IPv4,
			Action:    hrobot.ActionAccept,
			Protocol:  hrobot.ProtocolTCP,
			SourceIP:  "1.2.3.4/32",
			DestPort:  "22",
		},
	}

	tests := []struct {
		name     string
		newRule  hrobot.FirewallRule
		expected bool
	}{
		{
			name: "duplicate by name",
			newRule: hrobot.FirewallRule{
				Name:      "Allow SSH",
				IPVersion: hrobot.IPv4,
				Action:    hrobot.ActionAccept,
				Protocol:  hrobot.ProtocolTCP,
				SourceIP:  "5.6.7.8/32",
				DestPort:  "22",
			},
			expected: true,
		},
		{
			name: "duplicate by function",
			newRule: hrobot.FirewallRule{
				Name:      "SSH rule",
				IPVersion: hrobot.IPv4,
				Action:    hrobot.ActionAccept,
				Protocol:  hrobot.ProtocolTCP,
				SourceIP:  "1.2.3.4/32",
				DestPort:  "22",
			},
			expected: true,
		},
		{
			name: "unique rule",
			newRule: hrobot.FirewallRule{
				Name:      "Allow HTTPS",
				IPVersion: hrobot.IPv4,
				Action:    hrobot.ActionAccept,
				Protocol:  hrobot.ProtocolTCP,
				SourceIP:  "1.2.3.4/32",
				DestPort:  "443",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ruleExists(existingRules, tt.newRule)
			if result != tt.expected {
				t.Errorf("ruleExists() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestAddFirewallRules_WaitsForPendingChanges(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			// First GET returns "in process", second returns "active"
			status := "in process"
			if callCount > 0 {
				status = "active"
			}
			callCount++

			response := map[string]interface{}{
				"firewall": map[string]interface{}{
					"server_ip":     "123.123.123.123",
					"server_number": 321,
					"status":        status,
					"whitelist_hos": true,
					"filter_ipv6":   false,
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
		case "POST":
			// Return success for POST (update)
			response := map[string]interface{}{
				"firewall": map[string]interface{}{
					"server_ip":     "123.123.123.123",
					"server_number": 321,
					"status":        "active",
					"whitelist_hos": true,
					"filter_ipv6":   false,
					"port":          "main",
					"rules": map[string]interface{}{
						"input": []map[string]interface{}{
							{
								"name":       "Allow SSH 1.2.3.4",
								"ip_version": "ipv4",
								"action":     "accept",
								"protocol":   "tcp",
								"src_ip":     "1.2.3.4/32",
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
		}
	}))
	defer server.Close()

	client := hrobot.NewClient("test-user", "test-pass", hrobot.WithBaseURL(server.URL))
	ctx := context.Background()

	newRules := []hrobot.FirewallRule{
		{
			Name:      "Allow SSH 1.2.3.4",
			IPVersion: hrobot.IPv4,
			Action:    hrobot.ActionAccept,
			Protocol:  hrobot.ProtocolTCP,
			SourceIP:  "1.2.3.4/32",
			DestPort:  "22",
		},
	}

	info, err := addFirewallRules(ctx, client, hrobot.ServerID(321), newRules)
	if err != nil {
		t.Fatalf("addFirewallRules returned error: %v", err)
	}

	if info.Added != 1 {
		t.Errorf("expected 1 rule added, got %d", info.Added)
	}

	// Should have made 3 GET requests:
	// 1. Initial GET in addFirewallRules (returns "in process")
	// 2. GET in WaitForFirewallReady (returns "active")
	// 3. Re-fetch GET in addFirewallRules after waiting
	// Plus 1 POST to update
	if callCount != 3 {
		t.Errorf("expected 3 GET calls (initial + wait + re-fetch), got %d", callCount)
	}
}

func TestAddFirewallRules_InvalidInputError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			// Return firewall with no rules
			response := map[string]interface{}{
				"firewall": map[string]interface{}{
					"server_ip":     "123.123.123.123",
					"server_number": 321,
					"status":        "active",
					"whitelist_hos": true,
					"filter_ipv6":   false,
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
		case "POST":
			// Return INVALID_INPUT error
			w.WriteHeader(http.StatusBadRequest)
			response := map[string]interface{}{
				"error": map[string]interface{}{
					"status":  400,
					"code":    "INVALID_INPUT",
					"message": "invalid input",
				},
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Fatalf("failed to encode response: %v", err)
			}
		}
	}))
	defer server.Close()

	client := hrobot.NewClient("test-user", "test-pass", hrobot.WithBaseURL(server.URL))
	ctx := context.Background()

	newRules := []hrobot.FirewallRule{
		{
			Name:      "Test Rule",
			IPVersion: hrobot.IPv4,
			Action:    hrobot.ActionAccept,
			Protocol:  hrobot.ProtocolTCP,
			SourceIP:  "1.2.3.4/32",
			DestPort:  "22",
		},
	}

	_, err := addFirewallRules(ctx, client, hrobot.ServerID(321), newRules)
	if err == nil {
		t.Fatal("expected error for INVALID_INPUT, got nil")
	}

	// Check that error message contains helpful information
	errMsg := err.Error()
	if !strings.Contains(errMsg, "invalid input") {
		t.Errorf("error message should mention 'invalid input', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "Test Rule") {
		t.Errorf("error message should list the rule name, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "list-rules") {
		t.Errorf("error message should suggest listing rules, got: %s", errMsg)
	}
}
