// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aquasecurity/table"
	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

func getServer(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID) error {
	server, err := client.Server.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	// Get reset info to retrieve operating status
	reset, err := client.Reset.Get(ctx, serverID)
	var operatingStatus string
	if err != nil {
		operatingStatus = "(unavailable)"
	} else {
		operatingStatus = reset.OperatingStatus
		if operatingStatus == "" {
			operatingStatus = "ready"
		}
	}

	// Pretty print the server details
	fmt.Printf("Server Details:\n")
	fmt.Printf("  Server Number:     %d\n", server.ServerNumber)
	fmt.Printf("  Server Name:       %s\n", server.ServerName)
	fmt.Printf("  Server IP:         %s\n", server.ServerIP.String())
	fmt.Printf("  Product:           %s\n", server.Product)
	fmt.Printf("  DC:                %s\n", server.DC)
	fmt.Printf("  Status:            %s\n", server.Status)
	fmt.Printf("  Operating Status:  %s\n", operatingStatus)
	fmt.Printf("  Traffic:           %s\n", server.Traffic.String())
	fmt.Printf("  Cancelled:         %v\n", server.Cancelled)
	fmt.Printf("  Paid Until:        %s\n", server.PaidUntil)

	if len(server.IP) > 0 {
		fmt.Printf("  IP Addresses:\n")
		for i, ip := range server.IP {
			fmt.Printf("    [%d] %s", i, ip.String())
			if i == 0 && ip.To4() != nil {
				fmt.Printf(" (primary IPv4)")
			}
			fmt.Println()
		}
	}

	if len(server.Subnet) > 0 {
		fmt.Printf("  Subnets:\n")
		for i, subnet := range server.Subnet {
			fmt.Printf("    [%d] %s/%s\n", i, subnet.IP.String(), subnet.Mask)
		}
	}

	return nil
}

func listServers(ctx context.Context, client *hrobot.Client) error {
	servers, err := client.Server.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}

	fmt.Printf("Found %d server(s):\n\n", len(servers))

	t := table.New(os.Stdout)
	t.SetHeaders("Server #", "Name", "IP", "Product", "DC", "Status")

	for _, server := range servers {
		t.AddRow(
			fmt.Sprintf("%d", server.ServerNumber),
			server.ServerName,
			server.ServerIP.String(),
			server.Product,
			server.DC,
			string(server.Status),
		)
	}

	t.Render()
	return nil
}

func executeReset(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, resetType string) error {
	// Validate reset type
	validTypes := map[string]string{
		"sw":         "software reset (CTRL+ALT+DEL)",
		"hw":         "hardware reset (reset button)",
		"power":      "power cycle",
		"power_long": "shutdown (long power button press)",
		"man":        "manual reset",
	}

	description, valid := validTypes[resetType]
	if !valid {
		return fmt.Errorf("invalid reset type: %s\nValid types: sw, hw, power, power_long, man", resetType)
	}

	fmt.Printf("Executing %s on server #%d...\n", description, serverID)

	reset, err := client.Reset.Execute(ctx, serverID, hrobot.ResetType(resetType))
	if err != nil {
		return fmt.Errorf("failed to execute reset: %w", err)
	}

	fmt.Printf("✓ Reset executed successfully!\n")
	fmt.Printf("  Server IP: %s\n", reset.ServerIP.String())
	fmt.Printf("  Type:      %s\n", reset.Type)

	return nil
}

func powerOnServer(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID) error {
	// Get reset options to check operating status
	reset, err := client.Reset.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get server status: %w", err)
	}

	// Check if server is already powered on
	status := strings.ToLower(reset.OperatingStatus)
	if status == "ready" || status == "running" || status == "" {
		fmt.Printf("Server #%d is already powered on\n", serverID)
		fmt.Printf("  Server IP:        %s\n", reset.ServerIP.String())
		fmt.Printf("  Operating Status: %s\n", reset.OperatingStatus)
		return nil
	}

	// Server is powered off, send power command to turn it on
	fmt.Printf("Powering on server #%d...\n", serverID)
	fmt.Printf("  Current status: %s\n\n", reset.OperatingStatus)

	resetResult, err := client.Reset.Execute(ctx, serverID, hrobot.ResetTypePower)
	if err != nil {
		return fmt.Errorf("failed to power on server: %w", err)
	}

	fmt.Printf("✓ Power command sent successfully!\n")
	fmt.Printf("  Server IP: %s\n", resetResult.ServerIP.String())
	fmt.Printf("  Type:      %s\n", resetResult.Type)

	return nil
}

func powerOffServer(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID) error {
	// Get reset options to check operating status
	reset, err := client.Reset.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get server status: %w", err)
	}

	// Check if server is already powered off
	status := strings.ToLower(reset.OperatingStatus)
	if status == "off" || status == "powered off" || status == "shutdown" {
		fmt.Printf("Server #%d is already powered off\n", serverID)
		fmt.Printf("  Server IP:        %s\n", reset.ServerIP.String())
		fmt.Printf("  Operating Status: %s\n", reset.OperatingStatus)
		return nil
	}

	// Server is powered on, send power command to turn it off
	fmt.Printf("Powering off server #%d...\n", serverID)
	fmt.Printf("  Current status: %s\n\n", reset.OperatingStatus)

	resetResult, err := client.Reset.Execute(ctx, serverID, hrobot.ResetTypePower)
	if err != nil {
		return fmt.Errorf("failed to power off server: %w", err)
	}

	fmt.Printf("✓ Power command sent successfully!\n")
	fmt.Printf("  Server IP: %s\n", resetResult.ServerIP.String())
	fmt.Printf("  Type:      %s\n", resetResult.Type)

	return nil
}

func showTraffic(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, args []string) error {
	// Parse flags
	days := 30
	var fromDate, toDate string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--days":
			if i+1 < len(args) {
				d, err := strconv.Atoi(args[i+1])
				if err != nil {
					return fmt.Errorf("invalid --days value: %s", args[i+1])
				}
				days = d
				i++
			}
		case "--from":
			if i+1 < len(args) {
				fromDate = args[i+1]
				i++
			}
		case "--to":
			if i+1 < len(args) {
				toDate = args[i+1]
				i++
			}
		}
	}

	// Get server details to find IP address
	server, err := client.Server.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	serverIP := server.ServerIP.String()

	// Calculate date range if not provided
	if fromDate == "" || toDate == "" {
		now := time.Now()
		toDate = now.Format("2006-01-02")
		fromDate = now.AddDate(0, 0, -days+1).Format("2006-01-02")
	}

	// Fetch traffic data
	fmt.Printf("Fetching traffic data for server #%d (%s)...\n", serverID, serverIP)
	fmt.Printf("  Period: %s to %s\n\n", fromDate, toDate)

	// Use month type with single_values=true for proper format
	params := hrobot.TrafficGetParams{
		Type:         hrobot.TrafficTypeMonth,
		From:         fromDate,
		To:           toDate,
		IP:           serverIP,
		SingleValues: true,
	}

	trafficData, err := client.Traffic.Get(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to get traffic data: %w", err)
	}

	// Extract data for the server IP
	ipData, ok := trafficData.Data[serverIP]
	if !ok || len(ipData) == 0 {
		fmt.Println("No traffic data available for this period.")
		return nil
	}

	// Sort dates and find max traffic for scaling
	var dates []string
	maxTraffic := 0.0
	for date := range ipData {
		dates = append(dates, date)
		if ipData[date].Sum > maxTraffic {
			maxTraffic = ipData[date].Sum
		}
	}
	sort.Strings(dates)

	// Display traffic graph
	fmt.Printf("Traffic Statistics (GB)\n\n")

	// Determine scale for bar chart
	barWidth := 50
	scale := maxTraffic / float64(barWidth)
	if scale == 0 {
		scale = 1
	}

	totalIn := 0.0
	totalOut := 0.0
	totalSum := 0.0

	// Parse the from date to get year and month context
	fromTime, _ := time.Parse("2006-01-02", fromDate)

	// Create table
	t := table.New(os.Stdout)
	t.SetHeaders("Date", "Download", "Upload", "Graph")

	for _, date := range dates {
		traffic := ipData[date]
		totalIn += traffic.In
		totalOut += traffic.Out
		totalSum += traffic.Sum

		// Create bar
		barLength := int(traffic.Sum / scale)
		if barLength > barWidth {
			barLength = barWidth
		}
		bar := strings.Repeat("█", barLength)

		// Format date properly
		displayDate := date
		if len(date) == 2 {
			// Day of month only - construct full date
			dayNum, err := strconv.Atoi(date)
			if err == nil {
				// Use the year and month from the request period
				fullDate := time.Date(fromTime.Year(), fromTime.Month(), dayNum, 0, 0, 0, 0, time.UTC)
				displayDate = fullDate.Format("2006-01-02")
			}
		}

		// Add row
		t.AddRow(
			displayDate,
			fmt.Sprintf("%.2f GB", traffic.In),
			fmt.Sprintf("%.2f GB", traffic.Out),
			bar,
		)
	}

	t.Render()
	fmt.Printf("\nTotal Traffic: %.2f GB (↓%.2f GB in, ↑%.2f GB out)\n", totalSum, totalIn, totalOut)
	fmt.Printf("Average per day: %.2f GB\n", totalSum/float64(len(dates)))

	return nil
}
