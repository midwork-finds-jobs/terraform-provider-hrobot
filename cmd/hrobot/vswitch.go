// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"

	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

func listVSwitches(ctx context.Context, client *hrobot.Client) error {
	vswitches, err := client.VSwitch.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list vSwitches: %w", err)
	}

	fmt.Printf("Found %d vSwitch(es):\n\n", len(vswitches))
	for i, vs := range vswitches {
		fmt.Printf("[%d] %s (ID: %d)\n", i+1, vs.Name, vs.ID)
		fmt.Printf("    VLAN:      %d\n", vs.VLAN)
		fmt.Printf("    Cancelled: %v\n", vs.Cancelled)
		fmt.Println()
	}

	return nil
}

func getVSwitch(ctx context.Context, client *hrobot.Client, id int) error {
	vs, err := client.VSwitch.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get vSwitch: %w", err)
	}

	fmt.Printf("VSwitch Details:\n")
	fmt.Printf("  ID:        %d\n", vs.ID)
	fmt.Printf("  Name:      %s\n", vs.Name)
	fmt.Printf("  VLAN:      %d\n", vs.VLAN)
	fmt.Printf("  Cancelled: %v\n", vs.Cancelled)

	if len(vs.Servers) > 0 {
		fmt.Printf("\n  Servers (%d):\n", len(vs.Servers))
		for i, server := range vs.Servers {
			fmt.Printf("    [%d] #%d - %s\n", i+1, server.ServerNumber, server.ServerIP)
			if server.ServerIPv6Net != "" {
				fmt.Printf("        IPv6: %s\n", server.ServerIPv6Net)
			}
			fmt.Printf("        Status: %s\n", server.Status)
		}
	}

	if len(vs.Subnets) > 0 {
		fmt.Printf("\n  Subnets (%d):\n", len(vs.Subnets))
		for i, subnet := range vs.Subnets {
			fmt.Printf("    [%d] %s/%d\n", i+1, subnet.IP, subnet.Mask)
			fmt.Printf("        Gateway: %s\n", subnet.Gateway)
		}
	}

	if len(vs.CloudNetwork) > 0 {
		fmt.Printf("\n  Cloud Networks (%d):\n", len(vs.CloudNetwork))
		for i, cn := range vs.CloudNetwork {
			fmt.Printf("    [%d] ID %d - %s/%d\n", i+1, cn.ID, cn.IP, cn.Mask)
			fmt.Printf("        Gateway: %s\n", cn.Gateway)
		}
	}

	return nil
}

func createVSwitch(ctx context.Context, client *hrobot.Client, name string, vlan int) error {
	fmt.Printf("Creating vSwitch...\n")
	fmt.Printf("  Name: %s\n", name)
	fmt.Printf("  VLAN: %d\n\n", vlan)

	vs, err := client.VSwitch.Create(ctx, name, vlan)
	if err != nil {
		return fmt.Errorf("failed to create vSwitch: %w", err)
	}

	fmt.Printf("✓ VSwitch created successfully!\n")
	fmt.Printf("  ID:   %d\n", vs.ID)
	fmt.Printf("  Name: %s\n", vs.Name)
	fmt.Printf("  VLAN: %d\n", vs.VLAN)

	return nil
}

func updateVSwitch(ctx context.Context, client *hrobot.Client, id int, name string, vlan int) error {
	fmt.Printf("Updating vSwitch #%d...\n", id)
	fmt.Printf("  New Name: %s\n", name)
	fmt.Printf("  New VLAN: %d\n\n", vlan)

	err := client.VSwitch.Update(ctx, id, name, vlan)
	if err != nil {
		return fmt.Errorf("failed to update vSwitch: %w", err)
	}

	fmt.Printf("✓ VSwitch updated successfully!\n")

	return nil
}

func deleteVSwitch(ctx context.Context, client *hrobot.Client, id int, immediate bool) error {
	cancellationDate := "end_of_month"
	if immediate {
		cancellationDate = "immediately"
	}

	fmt.Printf("Cancelling vSwitch #%d...\n", id)
	fmt.Printf("  Cancellation: %s\n\n", cancellationDate)

	err := client.VSwitch.Delete(ctx, id, cancellationDate)
	if err != nil {
		return fmt.Errorf("failed to cancel vSwitch: %w", err)
	}

	fmt.Printf("✓ VSwitch cancelled successfully!\n")
	if !immediate {
		fmt.Println("  The vSwitch will be cancelled at the end of the month.")
	}

	return nil
}

func addServersToVSwitch(ctx context.Context, client *hrobot.Client, id int, servers []string) error {
	fmt.Printf("Adding %d server(s) to vSwitch #%d...\n", len(servers), id)
	for _, server := range servers {
		fmt.Printf("  - %s\n", server)
	}
	fmt.Println()

	err := client.VSwitch.AddServers(ctx, id, servers)
	if err != nil {
		return fmt.Errorf("failed to add servers to vSwitch: %w", err)
	}

	fmt.Printf("✓ Server(s) added successfully!\n")
	fmt.Println("  Note: It may take a few moments for the servers to become ready.")

	return nil
}

func removeServersFromVSwitch(ctx context.Context, client *hrobot.Client, id int, servers []string) error {
	fmt.Printf("Removing %d server(s) from vSwitch #%d...\n", len(servers), id)
	for _, server := range servers {
		fmt.Printf("  - %s\n", server)
	}
	fmt.Println()

	err := client.VSwitch.RemoveServers(ctx, id, servers)
	if err != nil {
		return fmt.Errorf("failed to remove servers from vSwitch: %w", err)
	}

	fmt.Printf("✓ Server(s) removed successfully!\n")

	return nil
}
