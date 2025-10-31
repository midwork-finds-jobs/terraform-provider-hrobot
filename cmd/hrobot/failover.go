// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

func listFailovers(ctx context.Context, client *hrobot.Client) error {
	failovers, err := client.Failover.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list failover IPs: %w", err)
	}

	fmt.Printf("Found %d failover IP(s):\n\n", len(failovers))

	for i, fo := range failovers {
		fmt.Printf("[%d] %s/%s\n", i+1, fo.IP, fo.Netmask)
		fmt.Printf("    Server:        #%d (%s)\n", fo.ServerNumber, fo.ServerIP)
		if fo.ServerIPv6Net != "" {
			fmt.Printf("    Server IPv6:   %s\n", fo.ServerIPv6Net)
		}
		if fo.ActiveServerIP != nil {
			fmt.Printf("    Routed to:     %s\n", *fo.ActiveServerIP)
		} else {
			fmt.Printf("    Routed to:     (not routed)\n")
		}
		fmt.Println()
	}

	return nil
}

func getFailover(ctx context.Context, client *hrobot.Client, ip string) error {
	failover, err := client.Failover.Get(ctx, ip)
	if err != nil {
		return fmt.Errorf("failed to get failover IP: %w", err)
	}

	fmt.Printf("Failover IP Details:\n")
	fmt.Printf("  IP:            %s\n", failover.IP)
	fmt.Printf("  Netmask:       %s\n", failover.Netmask)
	fmt.Printf("  Server:        #%d\n", failover.ServerNumber)
	fmt.Printf("  Server IP:     %s\n", failover.ServerIP)
	if failover.ServerIPv6Net != "" {
		fmt.Printf("  Server IPv6:   %s\n", failover.ServerIPv6Net)
	}
	if failover.ActiveServerIP != nil {
		fmt.Printf("  Active Server: %s\n", *failover.ActiveServerIP)
	} else {
		fmt.Printf("  Active Server: (not routed)\n")
	}

	// Also output as JSON for easy parsing
	fmt.Println("\nJSON Output:")
	data, err := json.MarshalIndent(failover, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))

	return nil
}

func setFailover(ctx context.Context, client *hrobot.Client, ip, destIP string) error {
	failover, err := client.Failover.Update(ctx, ip, destIP)
	if err != nil {
		return fmt.Errorf("failed to route failover IP: %w", err)
	}

	fmt.Printf("✓ Successfully routed failover IP\n")
	fmt.Printf("  Failover IP:   %s\n", failover.IP)
	if failover.ActiveServerIP != nil {
		fmt.Printf("  Now routed to: %s\n", *failover.ActiveServerIP)
	}

	return nil
}

func deleteFailover(ctx context.Context, client *hrobot.Client, ip string) error {
	err := client.Failover.Delete(ctx, ip)
	if err != nil {
		return fmt.Errorf("failed to unroute failover IP: %w", err)
	}

	fmt.Printf("✓ Successfully unrouted failover IP %s\n", ip)

	return nil
}
