// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"

	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

func getFirewall(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID) error {
	fw, err := client.Firewall.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get firewall: %w", err)
	}

	fmt.Printf("Firewall Configuration for Server #%d:\n", fw.ServerNumber)
	fmt.Printf("  Status:          %s\n", fw.Status)
	fmt.Printf("  Whitelist HOS:   %v\n", fw.WhitelistHOS)
	fmt.Printf("  Port:            %s\n", fw.Port)

	if len(fw.Rules.Input) > 0 {
		fmt.Printf("\n  Input Rules (%d):\n", len(fw.Rules.Input))
		for i, rule := range fw.Rules.Input {
			fmt.Printf("    [%d] %s - %s", i, rule.Name, rule.Action)
			if rule.IPVersion != "" {
				fmt.Printf(" (%s)", rule.IPVersion)
			}
			if rule.SourceIP != "" {
				fmt.Printf(" from %s", rule.SourceIP)
			}
			if rule.DestIP != "" {
				fmt.Printf(" to %s", rule.DestIP)
			}
			if rule.Protocol != "" {
				fmt.Printf(" proto %s", rule.Protocol)
			}
			if rule.DestPort != "" {
				fmt.Printf(" port %s", rule.DestPort)
			}
			fmt.Println()
		}
	}

	if len(fw.Rules.Output) > 0 {
		fmt.Printf("\n  Output Rules (%d):\n", len(fw.Rules.Output))
		for i, rule := range fw.Rules.Output {
			fmt.Printf("    [%d] %s - %s", i, rule.Name, rule.Action)
			if rule.IPVersion != "" {
				fmt.Printf(" (%s)", rule.IPVersion)
			}
			if rule.SourceIP != "" {
				fmt.Printf(" from %s", rule.SourceIP)
			}
			if rule.DestIP != "" {
				fmt.Printf(" to %s", rule.DestIP)
			}
			if rule.Protocol != "" {
				fmt.Printf(" proto %s", rule.Protocol)
			}
			if rule.DestPort != "" {
				fmt.Printf(" port %s", rule.DestPort)
			}
			fmt.Println()
		}
	}

	return nil
}

func allowIP(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, ipAddr string) error {
	// First, get the current firewall configuration
	fw, err := client.Firewall.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get firewall: %w", err)
	}

	fmt.Printf("Current firewall status: %s\n", fw.Status)
	fmt.Printf("Adding IP %s to allow list...\n", ipAddr)

	// Create a new rule to allow all traffic from the IP
	newRule := hrobot.FirewallRule{
		Name:      fmt.Sprintf("Allow %s", ipAddr),
		IPVersion: "ipv4",
		Action:    "accept",
		SourceIP:  ipAddr,
	}

	// Add the new rule to the beginning of the input rules
	updatedRules := append([]hrobot.FirewallRule{newRule}, fw.Rules.Input...)

	// Update the firewall configuration
	updateConfig := hrobot.UpdateConfig{
		Status:       fw.Status,
		WhitelistHOS: fw.WhitelistHOS,
		Rules: hrobot.FirewallRules{
			Input:  updatedRules,
			Output: fw.Rules.Output,
		},
	}

	updated, err := client.Firewall.Update(ctx, serverID, updateConfig)
	if err != nil {
		return fmt.Errorf("failed to update firewall: %w", err)
	}

	fmt.Printf("✓ Successfully added IP %s to firewall\n", ipAddr)
	fmt.Printf("  Status: %s\n", updated.Status)
	fmt.Printf("  Total input rules: %d\n", len(updated.Rules.Input))
	fmt.Println("\nNote: Firewall changes may take 30-40 seconds to apply.")

	return nil
}
