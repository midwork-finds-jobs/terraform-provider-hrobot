// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/aquasecurity/table"
	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

// diskInfo represents parsed disk information.
type diskInfo struct {
	quantity int
	size     float64 // normalized to GB for sorting
	sizeStr  string  // original size string for display
	unit     string  // GB or TB
	diskType string  // SSD or HDD
	tech     string  // NVMe, SATA, or empty
}

// parseAuctionGPU extracts GPU information from server description array.
// Format: "Nvidia RTX™ 4000 SFF Ada Generation".
func parseAuctionGPU(description []string) string {
	for _, line := range description {
		line = strings.TrimSpace(line)
		// Look for GPU lines (Nvidia, AMD Radeon, GeForce)
		if strings.Contains(line, "RTX") ||
			strings.Contains(line, "Radeon") ||
			strings.Contains(line, "GeForce") ||
			(strings.Contains(line, "Nvidia") && !strings.Contains(line, "bandwidth")) {
			return line
		}
	}
	return "-"
}

// parseAuctionMemoryType extracts memory type from server description array.
// Format: "2x RAM 16384 MB DDR4" -> "DDR4".
func parseAuctionMemoryType(description []string) string {
	memTypeRegex := regexp.MustCompile(`RAM\s+\d+\s+(?:MB|GB)\s+(DDR\d+(?:\s+ECC)?)`)

	for _, line := range description {
		line = strings.TrimSpace(line)
		if matches := memTypeRegex.FindStringSubmatch(line); len(matches) >= 2 {
			return matches[1]
		}
	}
	return "-"
}

// parseDiskDescription extracts disk information from server description array
// and formats it as grouped and sorted output.
func parseDiskDescription(description []string) string {
	if len(description) == 0 {
		return "-"
	}

	// Regex to match disk lines like "1x SSD U.2 NVMe 960 GB Datacenter"
	diskRegex := regexp.MustCompile(`^(\d+)x\s+(?:SSD|HDD).*?(\d+(?:[,.]\d+)?)\s+(GB|TB)`)

	var ssdNVMe []diskInfo
	var ssdSATA []diskInfo
	var hddSATA []diskInfo
	var hddOther []diskInfo

	for _, line := range description {
		line = strings.TrimSpace(line)

		// Check if line contains disk information
		if !strings.Contains(line, "SSD") && !strings.Contains(line, "HDD") {
			continue
		}

		matches := diskRegex.FindStringSubmatch(line)
		if len(matches) < 4 {
			continue
		}

		disk := diskInfo{
			sizeStr: matches[2],
			unit:    matches[3],
		}

		// Parse quantity
		disk.quantity, _ = strconv.Atoi(matches[1])

		// Parse size for sorting (normalize to GB)
		sizeStr := strings.ReplaceAll(disk.sizeStr, ",", ".")
		disk.size, _ = strconv.ParseFloat(sizeStr, 64)
		if disk.unit == "TB" {
			disk.size *= 1000
		}

		// Determine disk type
		if strings.Contains(line, "SSD") {
			disk.diskType = "SSD"
		} else if strings.Contains(line, "HDD") {
			disk.diskType = "HDD"
		}

		// Determine technology
		if strings.Contains(line, "NVMe") {
			disk.tech = "NVMe"
		} else if strings.Contains(line, "SATA") {
			disk.tech = "SATA"
		}

		// Group by type and technology
		if disk.diskType == "SSD" && disk.tech == "NVMe" {
			ssdNVMe = append(ssdNVMe, disk)
		} else if disk.diskType == "SSD" && disk.tech == "SATA" {
			ssdSATA = append(ssdSATA, disk)
		} else if disk.diskType == "HDD" && disk.tech == "SATA" {
			hddSATA = append(hddSATA, disk)
		} else if disk.diskType == "HDD" {
			hddOther = append(hddOther, disk)
		}
	}

	// Sort each group by size
	sortBySize := func(disks []diskInfo) {
		sort.Slice(disks, func(i, j int) bool {
			return disks[i].size < disks[j].size
		})
	}

	sortBySize(ssdSATA)
	sortBySize(ssdNVMe)
	sortBySize(hddSATA)
	sortBySize(hddOther)

	// Format output
	var groups []string

	formatGroup := func(disks []diskInfo, prefix string) string {
		if len(disks) == 0 {
			return ""
		}
		var parts []string
		for _, disk := range disks {
			sizeStr := strings.ReplaceAll(disk.sizeStr, ",", ".")
			parts = append(parts, fmt.Sprintf("%dx %s%s", disk.quantity, sizeStr, disk.unit))
		}
		return fmt.Sprintf("%s: %s", prefix, strings.Join(parts, " + "))
	}

	if group := formatGroup(ssdSATA, "SSD SATA"); group != "" {
		groups = append(groups, group)
	}
	if group := formatGroup(ssdNVMe, "SSD NVMe"); group != "" {
		groups = append(groups, group)
	}
	if group := formatGroup(hddSATA, "HDD SATA"); group != "" {
		groups = append(groups, group)
	}
	if group := formatGroup(hddOther, "HDD"); group != "" {
		groups = append(groups, group)
	}

	if len(groups) == 0 {
		return "-"
	}

	return strings.Join(groups, " | ")
}

func listAuctionServers(ctx context.Context, client *hrobot.Client, location string, memoryMin float64, cpu string, cpuBenchmarkMin uint32, diskSpaceMin float64, priceMax float64, gpuOnly bool) error {
	servers, err := client.Auction.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list auction servers: %w", err)
	}

	// Apply filters
	var filteredServers []hrobot.AuctionServer
	for _, server := range servers {
		// Filter by location
		if location != "" {
			if server.Datacenter == nil || !strings.Contains(strings.ToUpper(*server.Datacenter), strings.ToUpper(location)) {
				continue
			}
		}

		// Filter by minimum memory
		if memoryMin > 0 && server.MemorySize < memoryMin {
			continue
		}

		// Filter by CPU vendor
		if cpu != "" {
			cpuLower := strings.ToLower(server.CPU)
			cpuFilter := strings.ToLower(cpu)
			if !strings.Contains(cpuLower, cpuFilter) {
				continue
			}
		}

		// Filter by minimum CPU benchmark score
		if cpuBenchmarkMin > 0 && server.CPUBenchmark < cpuBenchmarkMin {
			continue
		}

		// Filter by minimum disk space
		if diskSpaceMin > 0 && server.HDDSize < diskSpaceMin {
			continue
		}

		// Filter by maximum price
		if priceMax > 0 && server.Price.Float64() > priceMax {
			continue
		}

		// Filter by GPU presence
		if gpuOnly {
			gpuInfo := parseAuctionGPU(server.Description)
			if gpuInfo == "-" {
				continue
			}
		}

		filteredServers = append(filteredServers, server)
	}

	fmt.Printf("Found %d auction server(s)", len(filteredServers))
	if location != "" || memoryMin > 0 || cpu != "" || cpuBenchmarkMin > 0 || diskSpaceMin > 0 || priceMax > 0 || gpuOnly {
		fmt.Printf(" (filtered from %d total)", len(servers))
	}
	fmt.Println(":\n")

	// Create table
	t := table.New(os.Stdout)
	t.SetHeaders("ID", "CPU", "GPU", "Memory", "Mem Type", "Storage", "Price/mo", "Setup", "Location", "Next cut")

	for _, server := range filteredServers {
		location := "-"
		if server.Datacenter != nil {
			location = *server.Datacenter
		}

		cpuInfo := fmt.Sprintf("%s (Benchmark: %d)", server.CPU, server.CPUBenchmark)
		gpuInfo := parseAuctionGPU(server.Description)
		memory := fmt.Sprintf("%.0f GB", server.MemorySize)
		memType := parseAuctionMemoryType(server.Description)
		price := fmt.Sprintf("%.2f €", server.Price.Float64())
		setup := fmt.Sprintf("%.2f €", server.PriceSetup.Float64())

		nextCut := "Auction"
		if server.FixedPrice {
			nextCut = "Fixed price"
		} else if server.NextReduce > 0 {
			hours := server.NextReduce / 3600
			minutes := (server.NextReduce % 3600) / 60
			nextCut = fmt.Sprintf("%dh %dm", hours, minutes)
		}

		t.AddRow(
			fmt.Sprintf("%d", server.ID),
			cpuInfo,
			gpuInfo,
			memory,
			memType,
			parseDiskDescription(server.Description),
			price,
			setup,
			location,
			nextCut,
		)
	}

	t.Render()
	return nil
}

func describeAuctionServer(ctx context.Context, client *hrobot.Client, serverID uint32) error {
	// Fetch all auction servers and find the one with matching ID
	servers, err := client.Auction.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list auction servers: %w", err)
	}

	// Find the server with matching ID
	var server *hrobot.AuctionServer
	for i := range servers {
		if servers[i].ID == serverID {
			server = &servers[i]
			break
		}
	}

	if server == nil {
		return fmt.Errorf("auction server with ID %d not found", serverID)
	}

	// Display server details
	fmt.Printf("Auction Server Details:\n")
	fmt.Printf("  Server ID:   %d\n", server.ID)
	fmt.Printf("  Name:        %s\n", server.Name)

	// Display specifications
	fmt.Printf("  Specifications:\n")
	fmt.Printf("    CPU:       %s (Benchmark: %d)\n", server.CPU, server.CPUBenchmark)

	// Parse and display GPU if available
	gpuInfo := parseAuctionGPU(server.Description)
	if gpuInfo != "-" {
		fmt.Printf("    GPU:       %s\n", gpuInfo)
	}

	fmt.Printf("    Memory:    %.0f GB\n", server.MemorySize)
	fmt.Printf("    Storage:   %s\n", parseDiskDescription(server.Description))

	// Show detailed description if available
	if len(server.Description) > 0 {
		fmt.Printf("  Detailed Description:\n")
		for _, desc := range server.Description {
			fmt.Printf("    - %s\n", desc)
		}
	}

	// Show location
	if server.Datacenter != nil {
		fmt.Printf("  Location:    %s\n", *server.Datacenter)
	}

	// Show pricing
	fmt.Printf("  Pricing:\n")
	fmt.Printf("    Monthly:   %.2f €\n", server.Price.Float64())
	fmt.Printf("    Setup:     %.2f €\n", server.PriceSetup.Float64())

	// Show status
	if server.FixedPrice {
		fmt.Printf("  Status:      Fixed price\n")
	} else if server.NextReduce > 0 {
		hours := server.NextReduce / 3600
		minutes := (server.NextReduce % 3600) / 60
		fmt.Printf("  Status:      Auction (next price reduction in %dh %dm)\n", hours, minutes)
	} else {
		fmt.Printf("  Status:      Auction\n")
	}

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
