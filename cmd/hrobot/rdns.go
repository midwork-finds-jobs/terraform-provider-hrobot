// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

func listRDNS(ctx context.Context, client *hrobot.Client, serverIP string) error {
	entries, err := client.RDNS.List(ctx, serverIP)
	if err != nil {
		return fmt.Errorf("failed to list reverse DNS entries: %w", err)
	}

	if serverIP != "" {
		fmt.Printf("Reverse DNS entries for server %s:\n\n", serverIP)
	} else {
		fmt.Printf("Found %d reverse DNS entry/entries:\n\n", len(entries))
	}

	for i, entry := range entries {
		fmt.Printf("[%d] %s\n", i+1, entry.IP)
		fmt.Printf("    PTR: %s\n\n", entry.PTR)
	}

	return nil
}

func getRDNS(ctx context.Context, client *hrobot.Client, ip string) error {
	entry, err := client.RDNS.Get(ctx, ip)
	if err != nil {
		return fmt.Errorf("failed to get reverse DNS entry: %w", err)
	}

	fmt.Printf("Reverse DNS entry for %s:\n", entry.IP)
	fmt.Printf("  PTR: %s\n", entry.PTR)

	// Also output as JSON for easy parsing
	fmt.Println("\nJSON Output:")
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))

	return nil
}

func setRDNS(ctx context.Context, client *hrobot.Client, ip, ptr string) error {
	// Try to update first (works for both create and update)
	entry, err := client.RDNS.Update(ctx, ip, ptr)
	if err != nil {
		return fmt.Errorf("failed to set reverse DNS entry: %w", err)
	}

	fmt.Printf("✓ Successfully set reverse DNS entry\n")
	fmt.Printf("  IP:  %s\n", entry.IP)
	fmt.Printf("  PTR: %s\n", entry.PTR)

	return nil
}

func deleteRDNS(ctx context.Context, client *hrobot.Client, ip string) error {
	err := client.RDNS.Delete(ctx, ip)
	if err != nil {
		return fmt.Errorf("failed to delete reverse DNS entry: %w", err)
	}

	fmt.Printf("✓ Successfully deleted reverse DNS entry for %s\n", ip)

	return nil
}
