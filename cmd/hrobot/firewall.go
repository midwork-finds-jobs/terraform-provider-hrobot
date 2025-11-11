// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aquasecurity/table"
	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

// Helper functions

// detectIPVersion detects if an IP address or CIDR is IPv4 or IPv6.
func detectIPVersion(ipStr string) hrobot.IPVersion {
	// Remove CIDR suffix if present
	ip := ipStr
	if strings.Contains(ipStr, "/") {
		ip = strings.Split(ipStr, "/")[0]
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return hrobot.IPv4 // default fallback
	}

	if parsed.To4() != nil {
		return hrobot.IPv4
	}
	return hrobot.IPv6
}

// isAutoAddedMailRule checks if a rule is one of Hetzner's automatically-added mail blocking rules.
func isAutoAddedMailRule(rule hrobot.FirewallRule) bool {
	// Hetzner auto-adds "Block mail ports" rules for ports 25,465
	// These appear in output rules
	if rule.Name == "Block mail ports" &&
		rule.Action == hrobot.ActionDiscard &&
		rule.Protocol == hrobot.ProtocolTCP &&
		rule.DestPort == "25,465" {
		return true
	}
	return false
}

// filterAutoAddedRules removes Hetzner's automatically-added rules from a rule list.
// These rules are added by Hetzner automatically, so we don't need to send them back.
func filterAutoAddedRules(rules []hrobot.FirewallRule) []hrobot.FirewallRule {
	filtered := make([]hrobot.FirewallRule, 0, len(rules))
	for _, rule := range rules {
		if !isAutoAddedMailRule(rule) {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

// ruleExists checks if a similar rule already exists in the rule list.
func ruleExists(rules []hrobot.FirewallRule, newRule hrobot.FirewallRule) bool {
	for _, rule := range rules {
		// Check if rules match on key fields
		if rule.Name == newRule.Name {
			return true
		}
		// Also check for functional duplicates (same action, protocol, IPs, port, flags)
		if rule.Action == newRule.Action &&
			rule.Protocol == newRule.Protocol &&
			rule.SourceIP == newRule.SourceIP &&
			rule.DestIP == newRule.DestIP &&
			rule.DestPort == newRule.DestPort &&
			rule.IPVersion == newRule.IPVersion &&
			rule.TCPFlags == newRule.TCPFlags {
			return true
		}
	}
	return false
}

// RulesAddedInfo contains information about the result of adding rules.
type RulesAddedInfo struct {
	Added   int
	Skipped int
}

// ensureFirewallReady checks if firewall is in "in process" state and waits for it to be ready.
// It returns the updated firewall config after waiting (if necessary).
func ensureFirewallReady(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, currentFw *hrobot.FirewallConfig) (*hrobot.FirewallConfig, error) {
	if currentFw.Status == "in process" {
		fmt.Println("⏳ firewall is processing previous changes, waiting for it to be ready...")
		if err := client.Firewall.WaitForFirewallReady(ctx, serverID); err != nil {
			return nil, fmt.Errorf("failed while waiting for firewall to be ready: %w", err)
		}
		// Re-fetch firewall config after waiting
		updatedFw, err := client.Firewall.Get(ctx, serverID)
		if err != nil {
			return nil, fmt.Errorf("failed to get firewall after waiting: %w", err)
		}
		fmt.Println("✓ firewall is ready, applying changes...")
		return updatedFw, nil
	}
	return currentFw, nil
}

// addFirewallRules is a helper that adds new input rules to the firewall.
// Returns information about how many rules were added/skipped.
func addFirewallRules(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, newRules []hrobot.FirewallRule) (*RulesAddedInfo, error) {
	fw, err := client.Firewall.Get(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get firewall: %w", err)
	}

	// Ensure firewall is ready before making changes
	fw, err = ensureFirewallReady(ctx, client, serverID, fw)
	if err != nil {
		return nil, err
	}

	// Filter out duplicate rules
	var rulesToAdd []hrobot.FirewallRule
	var skippedCount int
	for _, newRule := range newRules {
		if ruleExists(fw.Rules.Input, newRule) {
			skippedCount++
			fmt.Printf("⊘ skipping duplicate rule: %s\n", newRule.Name)
		} else {
			rulesToAdd = append(rulesToAdd, newRule)
		}
	}

	// If all rules were duplicates, nothing to do
	if len(rulesToAdd) == 0 {
		if skippedCount > 0 {
			fmt.Printf("\nℹ all %d rule(s) already exist, no changes made\n", skippedCount)
		}
		return &RulesAddedInfo{Added: 0, Skipped: skippedCount}, nil
	}

	// Filter out auto-added mail rules from existing rules before sending update
	filteredInput := filterAutoAddedRules(fw.Rules.Input)

	// Check if adding new rules would exceed the 10 rule limit
	const maxFirewallRules = 10
	totalRulesAfter := len(filteredInput) + len(rulesToAdd)
	if totalRulesAfter > maxFirewallRules {
		return nil, fmt.Errorf(`cannot add %d rule(s): would exceed firewall rule limit

Current rules: %d
Trying to add: %d
Total would be: %d
Maximum allowed: %d inbound rules

To resolve this:
  1. List existing rules: hrobot firewall list-rules %d
  2. Delete %d rule(s) you don't need: hrobot firewall delete-rule %d --index <N>
  3. Try adding your rules again

Note: Hetzner enforces a maximum of 10 inbound firewall rules per server`,
			len(rulesToAdd), len(filteredInput), len(rulesToAdd), totalRulesAfter, maxFirewallRules,
			serverID, totalRulesAfter-maxFirewallRules, serverID)
	}

	// Add new rules to the beginning of input rules
	updatedRules := append(rulesToAdd, filteredInput...)

	updateConfig := hrobot.UpdateConfig{
		Status:       fw.Status,
		WhitelistHOS: fw.WhitelistHOS,
		FilterIPv6:   fw.FilterIPv6,
		Rules: hrobot.FirewallRules{
			Input:  updatedRules,
			Output: filterAutoAddedRules(fw.Rules.Output),
		},
	}

	_, err = client.Firewall.Update(ctx, serverID, updateConfig)
	if err != nil {
		// Check if this is a rule limit error
		var hrobotErr *hrobot.Error
		if errors.As(err, &hrobotErr) && hrobot.IsFirewallRuleLimitExceededError(hrobotErr) {
			currentCount := len(fw.Rules.Input)
			return nil, fmt.Errorf(`firewall rule limit exceeded

Current rules: %d
Trying to add: %d
Maximum allowed: 10 inbound rules

To resolve this:
  1. List existing rules: hrobot firewall list-rules %d
  2. Delete rules you don't need: hrobot firewall delete-rule %d --index <N>
  3. Try adding your rules again

Note: Hetzner enforces a maximum of 10 inbound firewall rules per server`,
				currentCount, len(rulesToAdd), serverID, serverID)
		}

		// Check if this is an invalid input error (likely duplicate rules)
		if errors.As(err, &hrobotErr) && hrobot.IsInvalidInputError(hrobotErr) {
			// Build list of rules we tried to add
			var ruleNames []string
			for _, rule := range rulesToAdd {
				ruleNames = append(ruleNames, fmt.Sprintf("  - %s", rule.Name))
			}

			return nil, fmt.Errorf(`invalid input: the firewall rejected one or more rules

This usually means:
  • Similar rules already exist on the server (but weren't detected as duplicates)
  • Rules conflict with existing firewall configuration

Tried to add %d rule(s):
%s

To resolve this:
  1. Check existing rules: hrobot firewall list-rules %d
  2. Look for rules that might conflict with the ones above
  3. Delete conflicting rules if needed: hrobot firewall delete-rule %d --name "<rule-name>"
  4. Try adding your rules again

Original error: %v`,
				len(rulesToAdd),
				strings.Join(ruleNames, "\n"),
				serverID,
				serverID,
				err)
		}

		return nil, fmt.Errorf("failed to update firewall: %w", err)
	}

	return &RulesAddedInfo{Added: len(rulesToAdd), Skipped: skippedCount}, nil
}

// getMyIP attempts to get the user's current public IP.
func getMyIP() (string, error) {
	// Try to get IPv4 first
	resp, err := http.Get("https://ipinfo.io/ip")
	if err != nil {
		return "", fmt.Errorf("failed to get public IP: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read IP response: %w", err)
	}

	ip := strings.TrimSpace(string(body))
	if ip == "" {
		return "", fmt.Errorf("received empty IP address")
	}

	return ip, nil
}

// Phase 1: Essential convenience commands

func allowSSH(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, sourceIPs []string, myIP bool) error {
	ips := sourceIPs

	if myIP {
		ip, err := getMyIP()
		if err != nil {
			return err
		}
		fmt.Printf("detected your public IP: %s\n", ip)
		ips = []string{ip + "/32"}
	}

	if len(ips) == 0 {
		return fmt.Errorf("no source IPs specified")
	}

	var rules []hrobot.FirewallRule
	for _, ip := range ips {
		ipVersion := detectIPVersion(ip)
		// Extract just the IP for the name (without CIDR)
		nameIP := ip
		if strings.Contains(ip, "/") {
			nameIP = strings.Split(ip, "/")[0]
		}
		rule := hrobot.FirewallRule{
			Name:      fmt.Sprintf("Allow SSH %s", nameIP),
			IPVersion: ipVersion,
			Action:    hrobot.ActionAccept,
			Protocol:  hrobot.ProtocolTCP,
			SourceIP:  ip,
			DestPort:  "22",
		}
		rules = append(rules, rule)
	}

	info, err := addFirewallRules(ctx, client, serverID, rules)
	if err != nil {
		return err
	}

	// Only show success message if rules were actually added
	if info.Added > 0 {
		fmt.Printf("✓ successfully added %d SSH rule(s)\n", info.Added)
		for _, ip := range ips {
			fmt.Printf("  - allowed SSH from %s\n", ip)
		}
		fmt.Println("\nnote: firewall changes may take 30-40 seconds to apply")
	}

	return nil
}

func allowHTTPS(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, sourceIPs []string) error {
	if len(sourceIPs) == 0 {
		return fmt.Errorf("no source IPs specified")
	}

	var rules []hrobot.FirewallRule
	for _, ip := range sourceIPs {
		ipVersion := detectIPVersion(ip)
		// Extract just the IP for the name (without CIDR)
		nameIP := ip
		if strings.Contains(ip, "/") {
			nameIP = strings.Split(ip, "/")[0]
		}
		rule := hrobot.FirewallRule{
			Name:      fmt.Sprintf("Allow HTTPS %s", nameIP),
			IPVersion: ipVersion,
			Action:    hrobot.ActionAccept,
			Protocol:  hrobot.ProtocolTCP,
			SourceIP:  ip,
			DestPort:  "443",
		}
		rules = append(rules, rule)
	}

	info, err := addFirewallRules(ctx, client, serverID, rules)
	if err != nil {
		return err
	}

	if info.Added > 0 {
		fmt.Printf("✓ successfully added %d HTTPS rule(s)\n", info.Added)
		for _, ip := range sourceIPs {
			fmt.Printf("  - allowed HTTPS from %s (%s)\n", ip, detectIPVersion(ip))
		}
		fmt.Println("\nnote: firewall changes may take 30-40 seconds to apply")
	}

	return nil
}

func allowMOSH(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, sourceIPs []string, myIP bool) error {
	// Determine IPs
	ips := sourceIPs
	if myIP {
		ip, err := getMyIP()
		if err != nil {
			return err
		}
		fmt.Printf("detected your public IP: %s\n", ip)
		ips = []string{ip + "/32"}
	}

	if len(ips) == 0 {
		return fmt.Errorf("no source IPs specified")
	}

	// Build all MOSH rules (SSH TCP, MOSH UDP, TCP established)
	var rules []hrobot.FirewallRule

	// Add SSH rules
	for _, ip := range ips {
		ipVersion := detectIPVersion(ip)
		nameIP := ip
		if strings.Contains(ip, "/") {
			nameIP = strings.Split(ip, "/")[0]
		}

		// SSH rule (TCP port 22)
		sshRule := hrobot.FirewallRule{
			Name:      fmt.Sprintf("Allow SSH %s", nameIP),
			IPVersion: ipVersion,
			Action:    hrobot.ActionAccept,
			Protocol:  hrobot.ProtocolTCP,
			SourceIP:  ip,
			DestPort:  "22",
		}
		rules = append(rules, sshRule)

		// MOSH UDP rule (ports 60000-61000)
		moshRule := hrobot.FirewallRule{
			Name:      fmt.Sprintf("MOSH UDP %s", nameIP),
			IPVersion: ipVersion,
			Action:    hrobot.ActionAccept,
			Protocol:  hrobot.ProtocolUDP,
			SourceIP:  ip,
			DestPort:  "60000-61000",
		}
		rules = append(rules, moshRule)
	}

	// Add TCP established rule (allows return traffic for established connections)
	// Only add IPv4 version as it covers most use cases
	tcpEstablishedRule := hrobot.FirewallRule{
		Name:      "TCP established",
		IPVersion: hrobot.IPv4,
		Action:    hrobot.ActionAccept,
		Protocol:  hrobot.ProtocolTCP,
		DestPort:  "32768-65535",
		TCPFlags:  "ack",
	}
	rules = append(rules, tcpEstablishedRule)

	// Add all rules at once
	info, err := addFirewallRules(ctx, client, serverID, rules)
	if err != nil {
		return err
	}

	// Show summary of what was added
	if info.Added > 0 {
		fmt.Printf("\n✓ successfully configured MOSH access (%d rule(s) added)\n", info.Added)
		for _, ip := range ips {
			fmt.Printf("  - SSH from %s\n", ip)
			fmt.Printf("  - MOSH UDP (60000-61000) from %s\n", ip)
		}
		fmt.Printf("  - TCP established connections\n")
		fmt.Println("\nnote: firewall changes may take 30-40 seconds to apply")
	}

	if info.Skipped > 0 {
		fmt.Printf("\nℹ %d rule(s) already existed\n", info.Skipped)
	}

	if info.Added == 0 && info.Skipped > 0 {
		fmt.Println("\n✓ MOSH already configured for this IP")
	}

	return nil
}

func blockHTTP(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID) error {
	var rules []hrobot.FirewallRule

	// Block HTTP on both IPv4 and IPv6
	for _, ipVersion := range []hrobot.IPVersion{hrobot.IPv4, hrobot.IPv6} {
		rule := hrobot.FirewallRule{
			Name:      "block insecure HTTP",
			IPVersion: ipVersion,
			Action:    hrobot.ActionDiscard,
			Protocol:  hrobot.ProtocolTCP,
			DestPort:  "80",
		}
		rules = append(rules, rule)
	}

	info, err := addFirewallRules(ctx, client, serverID, rules)
	if err != nil {
		return err
	}

	if info.Added > 0 {
		fmt.Println("✓ successfully blocked insecure HTTP (port 80)")
		fmt.Println("note: firewall changes may take 30-40 seconds to apply")
	}

	return nil
}

func hardenFirewall(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, blockHTTPFlag bool) error {
	if !blockHTTPFlag {
		return fmt.Errorf("specify --block-http flag")
	}

	if err := blockHTTP(ctx, client, serverID); err != nil {
		return err
	}

	fmt.Println("\n✓ firewall hardening completed")
	return nil
}

// Phase 2: Granular rule management

func addRule(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, direction, protocol, action, name string, sourceIPs, destIPs []string, port string) error {
	if direction != "in" && direction != "out" {
		return fmt.Errorf("direction must be 'in' or 'out'")
	}

	if action == "" {
		action = "accept" // default
	}

	if action != "accept" && action != "discard" {
		return fmt.Errorf("action must be 'accept' or 'discard'")
	}

	// Validate protocol
	validProtocols := map[string]bool{"tcp": true, "udp": true, "icmp": true, "esp": true, "gre": true}
	if !validProtocols[protocol] {
		return fmt.Errorf("protocol must be one of: tcp, udp, icmp, esp, gre")
	}

	// Validate port requirements
	if (protocol == "tcp" || protocol == "udp") && port == "" && direction == "in" {
		return fmt.Errorf("port is required for TCP/UDP rules")
	}

	fw, err := client.Firewall.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get firewall: %w", err)
	}

	// Ensure firewall is ready before making changes
	fw, err = ensureFirewallReady(ctx, client, serverID, fw)
	if err != nil {
		return err
	}

	// Convert action string to typed constant
	var actionTyped hrobot.Action
	if action == "accept" {
		actionTyped = hrobot.ActionAccept
	} else {
		actionTyped = hrobot.ActionDiscard
	}

	// Convert protocol string to typed constant
	var protocolTyped hrobot.Protocol
	switch protocol {
	case "tcp":
		protocolTyped = hrobot.ProtocolTCP
	case "udp":
		protocolTyped = hrobot.ProtocolUDP
	case "icmp":
		protocolTyped = hrobot.ProtocolICMP
	case "esp":
		protocolTyped = hrobot.ProtocolESP
	case "gre":
		protocolTyped = hrobot.ProtocolGRE
	}

	var rules []hrobot.FirewallRule

	// Handle different IP combinations
	if direction == "in" {
		if len(sourceIPs) == 0 {
			sourceIPs = []string{"0.0.0.0/0"} // default to all
		}

		for _, sourceIP := range sourceIPs {
			ipVersion := detectIPVersion(sourceIP)
			rule := hrobot.FirewallRule{
				Name:      name,
				IPVersion: ipVersion,
				Action:    actionTyped,
				Protocol:  protocolTyped,
				SourceIP:  sourceIP,
				DestPort:  port,
			}
			rules = append(rules, rule)
		}
	} else {
		// direction == "out"
		if len(destIPs) == 0 {
			destIPs = []string{"0.0.0.0/0"}
		}

		for _, destIP := range destIPs {
			ipVersion := detectIPVersion(destIP)
			rule := hrobot.FirewallRule{
				Name:      name,
				IPVersion: ipVersion,
				Action:    actionTyped,
				Protocol:  protocolTyped,
				DestIP:    destIP,
				DestPort:  port,
			}
			rules = append(rules, rule)
		}
	}

	// Check for duplicates
	var rulesToAdd []hrobot.FirewallRule
	var skippedCount int
	existingRules := fw.Rules.Input
	if direction == "out" {
		existingRules = fw.Rules.Output
	}

	for _, newRule := range rules {
		if ruleExists(existingRules, newRule) {
			skippedCount++
			fmt.Printf("⊘ skipping duplicate rule: %s\n", newRule.Name)
		} else {
			rulesToAdd = append(rulesToAdd, newRule)
		}
	}

	if len(rulesToAdd) == 0 {
		if skippedCount > 0 {
			fmt.Printf("\nℹ all %d rule(s) already exist, no changes made\n", skippedCount)
		}
		return nil
	}

	// Add rules based on direction
	if direction == "in" {
		// Filter out auto-added mail rules from existing input rules
		filteredInput := filterAutoAddedRules(fw.Rules.Input)
		updatedRules := append(rulesToAdd, filteredInput...)
		updateConfig := hrobot.UpdateConfig{
			Status:       fw.Status,
			WhitelistHOS: fw.WhitelistHOS,
			FilterIPv6:   fw.FilterIPv6,
			Rules: hrobot.FirewallRules{
				Input:  updatedRules,
				Output: filterAutoAddedRules(fw.Rules.Output),
			},
		}
		_, err = client.Firewall.Update(ctx, serverID, updateConfig)
	} else {
		filteredOutput := filterAutoAddedRules(fw.Rules.Output)
		updatedRules := append(rulesToAdd, filteredOutput...)
		updateConfig := hrobot.UpdateConfig{
			Status:       fw.Status,
			WhitelistHOS: fw.WhitelistHOS,
			FilterIPv6:   fw.FilterIPv6,
			Rules: hrobot.FirewallRules{
				Input:  filterAutoAddedRules(fw.Rules.Input), // Filter here too
				Output: updatedRules,
			},
		}
		_, err = client.Firewall.Update(ctx, serverID, updateConfig)
	}

	if err != nil {
		return fmt.Errorf("failed to update firewall: %w", err)
	}

	fmt.Printf("✓ successfully added %d %s rule(s)\n", len(rulesToAdd), direction)
	if skippedCount > 0 {
		fmt.Printf("  (%d duplicate(s) skipped)\n", skippedCount)
	}
	fmt.Println("\nnote: firewall changes may take 30-40 seconds to apply")

	return nil
}

func deleteRule(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, name string, index int, direction string) error {
	fw, err := client.Firewall.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get firewall: %w", err)
	}

	// Ensure firewall is ready before making changes
	fw, err = ensureFirewallReady(ctx, client, serverID, fw)
	if err != nil {
		return err
	}

	if direction == "" {
		direction = "in" // default
	}

	var rules []hrobot.FirewallRule
	if direction == "in" {
		rules = fw.Rules.Input
	} else {
		rules = fw.Rules.Output
	}

	var updatedRules []hrobot.FirewallRule
	deleted := 0

	if name != "" {
		// Delete by name
		for _, rule := range rules {
			if rule.Name != name {
				updatedRules = append(updatedRules, rule)
			} else {
				deleted++
			}
		}
	} else if index >= 0 {
		// Delete by index
		if index >= len(rules) {
			return fmt.Errorf("index %d out of range (total rules: %d)", index, len(rules))
		}
		for i, rule := range rules {
			if i != index {
				updatedRules = append(updatedRules, rule)
			} else {
				deleted++
			}
		}
	} else {
		return fmt.Errorf("specify either --name or --index")
	}

	if deleted == 0 {
		return fmt.Errorf("no matching rules found")
	}

	// Update firewall
	updateConfig := hrobot.UpdateConfig{
		Status:       fw.Status,
		WhitelistHOS: fw.WhitelistHOS,
		FilterIPv6:   fw.FilterIPv6,
		Rules:        fw.Rules,
	}

	if direction == "in" {
		updateConfig.Rules.Input = updatedRules
		// Always filter auto-added mail rules from input
		updateConfig.Rules.Input = filterAutoAddedRules(updateConfig.Rules.Input)
		// Filter output rules too
		updateConfig.Rules.Output = filterAutoAddedRules(fw.Rules.Output)
	} else {
		updateConfig.Rules.Output = updatedRules
		// Filter auto-added mail rules from both input and output
		updateConfig.Rules.Input = filterAutoAddedRules(fw.Rules.Input)
		updateConfig.Rules.Output = filterAutoAddedRules(updateConfig.Rules.Output)
	}

	_, err = client.Firewall.Update(ctx, serverID, updateConfig)
	if err != nil {
		return fmt.Errorf("failed to update firewall: %w", err)
	}

	fmt.Printf("✓ successfully deleted %d rule(s) from %s rules\n", deleted, direction)
	fmt.Println("note: firewall changes may take 30-40 seconds to apply")

	return nil
}

func listRules(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, direction string, outputFormat string) error {
	fw, err := client.Firewall.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get firewall: %w", err)
	}

	if outputFormat == "json" {
		data, err := json.MarshalIndent(fw, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Show firewall status
	fmt.Printf("Firewall Configuration for Server #%d:\n", fw.ServerNumber)
	fmt.Printf("  Status:          %s\n", fw.Status)
	fmt.Printf("  Whitelist Hetzner Services: %v\n", fw.WhitelistHOS)
	fmt.Printf("  Filter IPv6:     %v\n", fw.FilterIPv6)

	// Show input rules
	if direction == "" || direction == "in" {
		if len(fw.Rules.Input) > 0 {
			fmt.Printf("\nInput Rules (%d):\n", len(fw.Rules.Input))
			t := table.New(os.Stdout)
			t.SetHeaders("#", "Name", "Action", "IP Ver", "Protocol", "Source IP", "Dest IP", "Port", "TCP Flags")

			for i, rule := range fw.Rules.Input {
				t.AddRow(
					strconv.Itoa(i),
					rule.Name,
					string(rule.Action),
					string(rule.IPVersion),
					string(rule.Protocol),
					rule.SourceIP,
					rule.DestIP,
					rule.DestPort,
					rule.TCPFlags,
				)
			}
			t.Render()
		} else {
			fmt.Println("\nNo input rules configured")
		}
	}

	// Show output rules
	if direction == "" || direction == "out" {
		if len(fw.Rules.Output) > 0 {
			fmt.Printf("\nOutput Rules (%d):\n", len(fw.Rules.Output))
			t := table.New(os.Stdout)
			t.SetHeaders("#", "Name", "Action", "IP Ver", "Protocol", "Source IP", "Dest IP", "Port", "TCP Flags")

			for i, rule := range fw.Rules.Output {
				t.AddRow(
					strconv.Itoa(i),
					rule.Name,
					string(rule.Action),
					string(rule.IPVersion),
					string(rule.Protocol),
					rule.SourceIP,
					rule.DestIP,
					rule.DestPort,
					rule.TCPFlags,
				)
			}
			t.Render()
		} else if direction == "out" {
			fmt.Println("\nNo output rules configured")
		}
	}

	return nil
}

// Phase 3: Template management

func listTemplates(ctx context.Context, client *hrobot.Client, outputFormat string) error {
	templates, err := client.Firewall.ListTemplates(ctx)
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	if outputFormat == "json" {
		data, err := json.MarshalIndent(templates, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	if len(templates) == 0 {
		fmt.Println("no firewall templates found")
		return nil
	}

	fmt.Printf("Firewall Templates (%d):\n", len(templates))
	t := table.New(os.Stdout)
	t.SetHeaders("ID", "Name", "Default", "Whitelist Hetzner Services", "Filter IPv6", "Input Rules", "Output Rules")

	for _, tmpl := range templates {
		t.AddRow(
			strconv.Itoa(tmpl.ID),
			tmpl.Name,
			strconv.FormatBool(tmpl.IsDefault),
			strconv.FormatBool(tmpl.WhitelistHOS),
			strconv.FormatBool(tmpl.FilterIPv6),
			strconv.Itoa(len(tmpl.Rules.Input)),
			strconv.Itoa(len(tmpl.Rules.Output)),
		)
	}
	t.Render()

	return nil
}

func describeTemplate(ctx context.Context, client *hrobot.Client, templateID int, outputFormat string) error {
	tmpl, err := client.Firewall.GetTemplate(ctx, strconv.Itoa(templateID))
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	if outputFormat == "json" {
		data, err := json.MarshalIndent(tmpl, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Firewall Template #%d:\n", tmpl.ID)
	fmt.Printf("  Name:            %s\n", tmpl.Name)
	fmt.Printf("  Is Default:      %v\n", tmpl.IsDefault)
	fmt.Printf("  Whitelist Hetzner Services: %v\n", tmpl.WhitelistHOS)
	fmt.Printf("  Filter IPv6:     %v\n", tmpl.FilterIPv6)

	if len(tmpl.Rules.Input) > 0 {
		fmt.Printf("\nInput Rules (%d):\n", len(tmpl.Rules.Input))
		t := table.New(os.Stdout)
		t.SetHeaders("#", "Name", "Action", "IP Ver", "Protocol", "Source IP", "Dest IP", "Port", "TCP Flags")

		for i, rule := range tmpl.Rules.Input {
			t.AddRow(
				strconv.Itoa(i),
				rule.Name,
				string(rule.Action),
				string(rule.IPVersion),
				string(rule.Protocol),
				rule.SourceIP,
				rule.DestIP,
				rule.DestPort,
				rule.TCPFlags,
			)
		}
		t.Render()
	} else {
		fmt.Println("\nNo input rules configured")
	}

	if len(tmpl.Rules.Output) > 0 {
		fmt.Printf("\nOutput Rules (%d):\n", len(tmpl.Rules.Output))
		t := table.New(os.Stdout)
		t.SetHeaders("#", "Name", "Action", "IP Ver", "Protocol", "Source IP", "Dest IP", "Port", "TCP Flags")

		for i, rule := range tmpl.Rules.Output {
			t.AddRow(
				strconv.Itoa(i),
				rule.Name,
				string(rule.Action),
				string(rule.IPVersion),
				string(rule.Protocol),
				rule.SourceIP,
				rule.DestIP,
				rule.DestPort,
				rule.TCPFlags,
			)
		}
		t.Render()
	} else {
		fmt.Println("\nNo output rules configured")
	}

	return nil
}

func applyTemplate(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, templateID int) error {
	fmt.Printf("applying template #%d to server #%d...\n", templateID, serverID)

	_, err := client.Firewall.ApplyTemplate(ctx, serverID, strconv.Itoa(templateID))
	if err != nil {
		return fmt.Errorf("failed to apply template: %w", err)
	}

	fmt.Printf("✓ successfully applied template #%d to server #%d\n", templateID, serverID)
	fmt.Println("note: firewall changes may take 30-40 seconds to apply")

	return nil
}

func createTemplate(ctx context.Context, client *hrobot.Client, name string, fromServerID hrobot.ServerID, rulesFile string, whitelistHOS bool, filterIPv6 bool) error {
	var config hrobot.TemplateConfig

	if fromServerID > 0 {
		// Create from existing server firewall
		fw, err := client.Firewall.Get(ctx, fromServerID)
		if err != nil {
			return fmt.Errorf("failed to get firewall from server #%d: %w", fromServerID, err)
		}

		config = hrobot.TemplateConfig{
			Name:         name,
			WhitelistHOS: fw.WhitelistHOS,
			Rules:        fw.Rules,
		}
	} else if rulesFile != "" {
		// Create from rules file
		var fileData []byte
		var err error

		if rulesFile == "-" {
			fileData, err = io.ReadAll(os.Stdin)
		} else {
			fileData, err = os.ReadFile(rulesFile)
		}

		if err != nil {
			return fmt.Errorf("failed to read rules file: %w", err)
		}

		if err := json.Unmarshal(fileData, &config); err != nil {
			return fmt.Errorf("failed to parse rules file: %w", err)
		}

		config.Name = name
	} else {
		// Create empty template
		config = hrobot.TemplateConfig{
			Name:         name,
			WhitelistHOS: whitelistHOS,
			FilterIPv6:   filterIPv6,
		}
	}

	tmpl, err := client.Firewall.CreateTemplate(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	fmt.Printf("✓ successfully created template #%d: %s\n", tmpl.ID, tmpl.Name)
	fmt.Printf("  input rules:  %d\n", len(tmpl.Rules.Input))
	fmt.Printf("  output rules: %d\n", len(tmpl.Rules.Output))

	return nil
}

func deleteTemplate(ctx context.Context, client *hrobot.Client, templateID int, confirm bool) error {
	if !confirm {
		return fmt.Errorf("template deletion requires --confirm flag")
	}

	err := client.Firewall.DeleteTemplate(ctx, strconv.Itoa(templateID))
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	fmt.Printf("✓ successfully deleted template #%d\n", templateID)

	return nil
}

// Phase 4: Status management

func enableFirewall(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, filterIPv6 *bool) error {
	fmt.Printf("enabling firewall for server #%d...\n", serverID)

	// Get current firewall config
	fw, err := client.Firewall.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get firewall: %w", err)
	}

	// Ensure firewall is ready before making changes
	fw, err = ensureFirewallReady(ctx, client, serverID, fw)
	if err != nil {
		return err
	}

	// Determine filter_ipv6 value
	ipv6Filter := fw.FilterIPv6
	if filterIPv6 != nil {
		ipv6Filter = *filterIPv6
	}

	// Update firewall with active status and optionally update IPv6 filtering
	updateConfig := hrobot.UpdateConfig{
		Status:       hrobot.FirewallStatusActive,
		WhitelistHOS: fw.WhitelistHOS,
		FilterIPv6:   ipv6Filter,
		Rules: hrobot.FirewallRules{
			Input:  filterAutoAddedRules(fw.Rules.Input),
			Output: filterAutoAddedRules(fw.Rules.Output),
		},
	}

	_, err = client.Firewall.Update(ctx, serverID, updateConfig)
	if err != nil {
		return fmt.Errorf("failed to enable firewall: %w", err)
	}

	fmt.Printf("✓ successfully enabled firewall for server #%d\n", serverID)
	if filterIPv6 != nil {
		if *filterIPv6 {
			fmt.Println("  IPv6 filtering: enabled")
		} else {
			fmt.Println("  IPv6 filtering: disabled")
		}
	}
	fmt.Println("\nnote: firewall changes may take 30-40 seconds to apply")

	return nil
}

func disableFirewall(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID) error {
	fmt.Printf("disabling firewall for server #%d...\n", serverID)

	// Get current firewall config
	fw, err := client.Firewall.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get firewall: %w", err)
	}

	// Ensure firewall is ready before making changes
	fw, err = ensureFirewallReady(ctx, client, serverID, fw)
	if err != nil {
		return err
	}

	// Update firewall with disabled status while preserving other settings
	updateConfig := hrobot.UpdateConfig{
		Status:       hrobot.FirewallStatusDisabled,
		WhitelistHOS: fw.WhitelistHOS,
		FilterIPv6:   fw.FilterIPv6,
		Rules: hrobot.FirewallRules{
			Input:  filterAutoAddedRules(fw.Rules.Input),
			Output: filterAutoAddedRules(fw.Rules.Output),
		},
	}

	_, err = client.Firewall.Update(ctx, serverID, updateConfig)
	if err != nil {
		return fmt.Errorf("failed to disable firewall: %w", err)
	}

	fmt.Printf("✓ successfully disabled firewall for server #%d\n", serverID)
	fmt.Println("note: firewall changes may take 30-40 seconds to apply")

	return nil
}

func getFirewallStatus(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID) error {
	fw, err := client.Firewall.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get firewall status: %w", err)
	}

	fmt.Printf("Firewall Status for Server #%d:\n", fw.ServerNumber)
	fmt.Printf("  Status:          %s\n", fw.Status)
	fmt.Printf("  Whitelist Hetzner Services: %v\n", fw.WhitelistHOS)
	fmt.Printf("  Filter IPv6:     %v\n", fw.FilterIPv6)
	fmt.Printf("  Input Rules:     %d\n", len(fw.Rules.Input))
	fmt.Printf("  Output Rules:    %d\n", len(fw.Rules.Output))

	return nil
}

func waitForFirewall(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID) error {
	fmt.Printf("waiting for firewall to be ready for server #%d...\n", serverID)

	err := client.Firewall.WaitForFirewallReady(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed while waiting for firewall: %w", err)
	}

	fmt.Printf("✓ firewall is ready for server #%d\n", serverID)

	return nil
}

func resetFirewall(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, confirm bool) error {
	if !confirm {
		return fmt.Errorf("firewall reset requires --confirm flag (this will delete all rules)")
	}

	fmt.Printf("resetting firewall for server #%d...\n", serverID)

	err := client.Firewall.Delete(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to reset firewall: %w", err)
	}

	fmt.Printf("✓ successfully reset firewall for server #%d (all rules deleted)\n", serverID)
	fmt.Println("note: firewall changes may take 30-40 seconds to apply")

	return nil
}
