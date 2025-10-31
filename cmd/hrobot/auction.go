// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aquasecurity/table"
	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

func listAuctionServers(ctx context.Context, client *hrobot.Client) error {
	servers, err := client.Auction.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list auction servers: %w", err)
	}

	fmt.Printf("Found %d auction server(s):\n\n", len(servers))

	// Create table
	t := table.New(os.Stdout)
	t.SetHeaders("ID", "Name", "CPU", "Memory", "Storage", "Price/mo", "Setup", "Location", "Status")

	for _, server := range servers {
		location := "-"
		if server.Datacenter != nil {
			location = *server.Datacenter
		}

		cpuInfo := fmt.Sprintf("%s (Benchmark: %d)", server.CPU, server.CPUBenchmark)
		memory := fmt.Sprintf("%.0f GB", server.MemorySize)
		price := fmt.Sprintf("%.2f €", server.Price.Float64())
		setup := fmt.Sprintf("%.2f €", server.PriceSetup.Float64())

		status := "Auction"
		if server.FixedPrice {
			status = "Fixed price"
		} else if server.NextReduce > 0 {
			hours := server.NextReduce / 3600
			minutes := (server.NextReduce % 3600) / 60
			status = fmt.Sprintf("Next cut: %dh %dm", hours, minutes)
		}

		t.AddRow(
			fmt.Sprintf("%d", server.ID),
			server.Name,
			cpuInfo,
			memory,
			server.HDDText,
			price,
			setup,
			location,
			status,
		)
	}

	t.Render()
	return nil
}

func orderMarketServer(ctx context.Context, client *hrobot.Client, productID uint32, sshKeyFingerprints []string, testMode bool, skipConfirmation bool) error {
	// First, fetch the auction server details to show the user what they're ordering
	fmt.Printf("Fetching server details...\n\n")
	servers, err := client.Auction.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch auction servers: %w", err)
	}

	// Find the server with matching product ID
	var server *hrobot.AuctionServer
	for i := range servers {
		if servers[i].ID == productID {
			server = &servers[i]
			break
		}
	}

	if server == nil {
		return fmt.Errorf("server with product ID %d not found in auction list", productID)
	}

	// Display server details
	fmt.Printf("Server Details:\n")
	fmt.Printf("  Product ID:  %d\n", server.ID)
	fmt.Printf("  Name:        %s\n", server.Name)
	fmt.Printf("  CPU:         %s (Benchmark: %d)\n", server.CPU, server.CPUBenchmark)
	fmt.Printf("  Memory:      %.0f GB\n", server.MemorySize)
	fmt.Printf("  Storage:     %s\n", server.HDDText)
	fmt.Printf("  Traffic:     %s\n", server.Traffic)
	if server.Datacenter != nil {
		fmt.Printf("  Location:    %s\n", *server.Datacenter)
	}
	fmt.Printf("  Price:       %.2f €/month (%.2f € incl. VAT)\n", server.Price.Float64(), server.PriceVAT.Float64())
	fmt.Printf("  Setup:       %.2f € (%.2f € incl. VAT)\n", server.PriceSetup.Float64(), server.PriceSetupVAT.Float64())
	if server.FixedPrice {
		fmt.Printf("  Status:      Fixed price (lowest price reached)\n")
	} else if server.NextReduce > 0 {
		hours := server.NextReduce / 3600
		minutes := (server.NextReduce % 3600) / 60
		fmt.Printf("  Next cut:    in %dh %dm (%s)\n", hours, minutes, server.NextReduceDate)
	}
	fmt.Println()

	// Show order configuration
	fmt.Printf("Order Configuration:\n")
	if len(sshKeyFingerprints) == 1 {
		fmt.Printf("  SSH Key:     %s\n", sshKeyFingerprints[0])
	} else {
		fmt.Printf("  SSH Keys:    %d keys\n", len(sshKeyFingerprints))
	}
	if testMode {
		fmt.Printf("  Test Mode:   enabled (order will not be placed)\n")
	}
	fmt.Println()

	// Ask for confirmation unless --yes flag was used
	if !skipConfirmation {
		fmt.Printf("Do you want to proceed with this order? (y/N): ")
		var response string
		// Read response, treating any error (e.g., EOF) as empty input
		if _, err := fmt.Scanln(&response); err != nil {
			response = ""
		}
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Order cancelled.")
			return nil
		}
		fmt.Println()
	}

	// Proceed with the order
	order := hrobot.MarketProductOrder{
		ProductID: productID,
		Auth: hrobot.AuthorizationMethod{
			Keys: sshKeyFingerprints,
		},
		Distribution: "Rescue system",
		Language:     "en",
		Test:         testMode,
	}

	fmt.Printf("Placing order...\n")
	tx, err := client.Ordering.PlaceMarketOrder(ctx, order)
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	fmt.Printf("\n✓ Order placed successfully!\n")
	fmt.Printf("  Transaction ID: %s\n", tx.ID)
	fmt.Printf("  Status:         %s\n", tx.Status)
	fmt.Printf("  Date:           %s\n", tx.Date.Format("2006-01-02 15:04:05"))
	if tx.ServerNumber != nil {
		fmt.Printf("  Server Number:  %d\n", *tx.ServerNumber)
	}
	if tx.ServerIP != nil {
		fmt.Printf("  Server IP:      %s\n", *tx.ServerIP)
	}

	return nil
}
