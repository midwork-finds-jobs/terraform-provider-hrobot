// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/aquasecurity/table"
	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

// parseProductCPU extracts CPU information from description array.
// Format: "Intel® Core™ i5-13500 14 Core "Raptor Lake-S"".
func parseProductCPU(description []string) string {
	for _, line := range description {
		line = strings.TrimSpace(line)
		// Look for lines with Intel, AMD, or "Core" that don't contain RAM, SSD, HDD, bandwidth, and not GPU lines
		if (strings.Contains(line, "Intel") || strings.Contains(line, "AMD") || strings.Contains(line, "Core")) &&
			!strings.Contains(line, "RAM") &&
			!strings.Contains(line, "SSD") &&
			!strings.Contains(line, "HDD") &&
			!strings.Contains(line, "bandwidth") &&
			!strings.Contains(line, "RTX") &&
			!strings.Contains(line, "Radeon") &&
			!strings.Contains(line, "GeForce") {
			return line
		}
	}
	return "-"
}

// parseProductGPU extracts GPU information from description array.
// Format: "Nvidia RTX™ 4000 SFF Ada Generation".
func parseProductGPU(description []string) string {
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

// stripProductNamePrefix removes "Dedicated Server" or "Dedicated GPU-Server" prefix from product name.
func stripProductNamePrefix(name string) string {
	name = strings.TrimPrefix(name, "Dedicated GPU-Server ")
	name = strings.TrimPrefix(name, "Dedicated Server ")
	return name
}

// parseProductMemory extracts memory information from description array.
// Format: "128 GB DDR5 ECC RAM" or "64 GB DDR4 RAM" or "64 GB DDR5 UDIMM" or "128 GB DDR5 ECC reg. RAM".
func parseProductMemory(description []string) float64 {
	// Match DDR memory with optional ECC, optional reg., and optional RAM/UDIMM/RDIMM at end
	memRegex := regexp.MustCompile(`(\d+(?:[,.]\d+)?)\s*(GB|TB)\s+DDR\d+(?:\s+(?:ECC|UDIMM|RDIMM|DIMM))?(?:\s+reg\.)?(?:\s+(?:RAM|UDIMM|RDIMM|DIMM))?`)

	for _, line := range description {
		line = strings.TrimSpace(line)
		if matches := memRegex.FindStringSubmatch(line); len(matches) >= 3 {
			sizeStr := strings.ReplaceAll(matches[1], ",", ".")
			size, _ := strconv.ParseFloat(sizeStr, 64)
			if matches[2] == "TB" {
				size *= 1000
			}
			return size
		}
	}
	return 0
}

// parseProductMemoryType extracts memory type from description array.
// Format: "128 GB DDR5 ECC RAM" -> "DDR5 ECC" or "64 GB DDR5 UDIMM" -> "DDR5 UDIMM".
func parseProductMemoryType(description []string) string {
	// Match DDR type with optional ECC/reg and optional ending type (RAM/UDIMM/RDIMM)
	memTypeRegex := regexp.MustCompile(`\d+\s*(?:GB|TB)\s+(DDR\d+(?:\s+(?:ECC|reg\.|UDIMM|RDIMM|DIMM))*)`)

	for _, line := range description {
		line = strings.TrimSpace(line)
		if matches := memTypeRegex.FindStringSubmatch(line); len(matches) >= 2 {
			// Clean up the matched type (remove "reg." as it's just a detail)
			memType := strings.TrimSpace(matches[1])
			memType = strings.ReplaceAll(memType, " reg.", "")
			memType = strings.TrimSpace(memType)
			return memType
		}
	}
	return "-"
}

// parseProductDiskSpace calculates total disk space from description array.
// Format: "2 x 512 GB NVMe SSD (Gen 4, Software RAID 1)" or "2x 1 TB NVMe SSD M.2".
func parseProductDiskSpace(description []string) float64 {
	// Match with or without space after quantity: "2 x 512 GB" or "2x 1 TB"
	diskRegex := regexp.MustCompile(`(\d+)\s*x\s+(\d+(?:[,.]\d+)?)\s+(GB|TB)\s+(?:NVMe\s+)?(?:SATA\s+)?(?:SSD|HDD)`)
	var totalGB float64

	for _, line := range description {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "SSD") && !strings.Contains(line, "HDD") {
			continue
		}

		matches := diskRegex.FindStringSubmatch(line)
		if len(matches) >= 4 {
			quantity, _ := strconv.Atoi(matches[1])
			sizeStr := strings.ReplaceAll(matches[2], ",", ".")
			size, _ := strconv.ParseFloat(sizeStr, 64)
			if matches[3] == "TB" {
				size *= 1000
			}
			totalGB += float64(quantity) * size
		}
	}
	return totalGB
}

// parseProductDiskInfo formats disk information for display.
// Format: "2 x 512 GB NVMe SSD (Gen 4, Software RAID 1)" -> "2x512GB NVMe SSD" or "2x 1 TB NVMe SSD M.2" -> "2x1TB NVMe SSD".
func parseProductDiskInfo(description []string) string {
	// Match with or without space after quantity: "2 x 512 GB" or "2x 1 TB"
	diskRegex := regexp.MustCompile(`(\d+)\s*x\s+(\d+(?:[,.]\d+)?)\s+(GB|TB)\s+(NVMe|SATA)?\s*(SSD|HDD)`)

	for _, line := range description {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "SSD") && !strings.Contains(line, "HDD") {
			continue
		}

		matches := diskRegex.FindStringSubmatch(line)
		if len(matches) >= 4 {
			sizeStr := strings.ReplaceAll(matches[2], ",", ".")
			// Format: "2x512GB NVMe SSD"
			result := fmt.Sprintf("%sx%s%s", matches[1], sizeStr, matches[3])
			if len(matches) >= 5 && matches[4] != "" {
				result += " " + matches[4]
			}
			if len(matches) >= 6 {
				result += " " + matches[5]
			}
			return result
		}
	}
	return "-"
}

func listProducts(ctx context.Context, client *hrobot.Client, location string, memoryMin float64, cpu string, cpuBenchmarkMin uint32, diskSpaceMin float64, priceMax float64, gpuOnly bool) error {
	products, err := client.Ordering.ListProducts(ctx)
	if err != nil {
		return fmt.Errorf("failed to list products: %w", err)
	}

	// Apply filters
	var filteredProducts []hrobot.Product
	for _, product := range products {
		// Parse product specs from description
		cpuName := parseProductCPU(product.Description)
		memory := parseProductMemory(product.Description)
		diskSpace := parseProductDiskSpace(product.Description)

		// Filter by location
		if location != "" {
			hasLocation := false
			for _, loc := range product.Locations {
				if strings.Contains(strings.ToUpper(loc), strings.ToUpper(location)) {
					hasLocation = true
					break
				}
			}
			if !hasLocation {
				continue
			}
		}

		// Filter by minimum memory
		if memoryMin > 0 && memory < memoryMin {
			continue
		}

		// Filter by CPU vendor
		if cpu != "" {
			cpuLower := strings.ToLower(cpuName)
			cpuFilter := strings.ToLower(cpu)
			if !strings.Contains(cpuLower, cpuFilter) {
				continue
			}
		}

		// Note: CPU benchmark filtering not available for products (no benchmark data)

		// Filter by minimum disk space
		if diskSpaceMin > 0 && diskSpace < diskSpaceMin {
			continue
		}

		// Filter by maximum price (use lowest price across locations)
		if priceMax > 0 && len(product.Prices) > 0 {
			lowestPrice := product.Prices[0].Price.Net.Float64()
			for _, p := range product.Prices {
				if p.Price.Net.Float64() < lowestPrice {
					lowestPrice = p.Price.Net.Float64()
				}
			}
			if lowestPrice > priceMax {
				continue
			}
		}

		// Filter by GPU presence
		if gpuOnly {
			gpuInfo := parseProductGPU(product.Description)
			if gpuInfo == "-" {
				continue
			}
		}

		filteredProducts = append(filteredProducts, product)
	}

	fmt.Printf("Found %d product server(s)", len(filteredProducts))
	if location != "" || memoryMin > 0 || cpu != "" || cpuBenchmarkMin > 0 || diskSpaceMin > 0 || priceMax > 0 || gpuOnly {
		fmt.Printf(" (filtered from %d total)", len(products))
	}
	fmt.Println(":\n")

	// Create table
	t := table.New(os.Stdout)
	t.SetHeaders("Product ID", "CPU", "GPU", "Memory", "Mem Type", "Storage", "Price/mo", "Setup", "Locations")

	for _, product := range filteredProducts {
		locations := strings.Join(product.Locations, ", ")
		if locations == "" {
			locations = "-"
		}

		// Parse product specs
		cpuInfo := parseProductCPU(product.Description)
		gpuName := parseProductGPU(product.Description)
		memory := parseProductMemory(product.Description)
		memType := parseProductMemoryType(product.Description)
		diskInfo := parseProductDiskInfo(product.Description)

		// Format memory
		memoryStr := "-"
		if memory > 0 {
			memoryStr = fmt.Sprintf("%.0f GB", memory)
		}

		// Find lowest price
		var lowestPrice float64
		var lowestSetup float64
		if len(product.Prices) > 0 {
			lowestPrice = product.Prices[0].Price.Net.Float64()
			lowestSetup = product.Prices[0].PriceSetup.Net.Float64()
			for _, p := range product.Prices {
				if p.Price.Net.Float64() < lowestPrice {
					lowestPrice = p.Price.Net.Float64()
				}
				if p.PriceSetup.Net.Float64() < lowestSetup {
					lowestSetup = p.PriceSetup.Net.Float64()
				}
			}
		}

		priceStr := fmt.Sprintf("%.2f €", lowestPrice)
		setupStr := fmt.Sprintf("%.2f €", lowestSetup)

		t.AddRow(
			product.ID,
			cpuInfo,
			gpuName,
			memoryStr,
			memType,
			diskInfo,
			priceStr,
			setupStr,
			locations,
		)
	}

	t.Render()

	fmt.Printf("\nNote: Prices shown are the lowest available across all locations\n")
	fmt.Printf("      Use 'hrobot product describe <product-id>' for full details\n")
	fmt.Printf("      Use 'hrobot product order <product-id>' to order a server\n")

	return nil
}

func describeProduct(ctx context.Context, client *hrobot.Client, productID string) error {
	// Fetch the product list to find the product details
	products, err := client.Ordering.ListProducts(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch products: %w", err)
	}

	// Find the product with matching ID
	var product *hrobot.Product
	for i := range products {
		if products[i].ID == productID {
			product = &products[i]
			break
		}
	}

	if product == nil {
		return fmt.Errorf("product with ID %s not found", productID)
	}

	// Display server details
	fmt.Printf("Product Server Details:\n")
	fmt.Printf("  Product ID:  %s\n", product.ID)
	if product.Name != "" {
		fmt.Printf("  Name:        %s\n", stripProductNamePrefix(product.Name))
	}

	// Show description array which contains the actual specs
	if len(product.Description) > 0 {
		fmt.Printf("  Specifications:\n")
		for _, desc := range product.Description {
			fmt.Printf("    - %s\n", desc)
		}
	}

	if product.Traffic != "" {
		fmt.Printf("  Traffic:     %s\n", product.Traffic)
	}

	// Show available locations
	if len(product.Locations) > 0 {
		fmt.Printf("  Locations:   %s\n", strings.Join(product.Locations, ", "))
	}

	// Show pricing per location
	if len(product.Prices) > 0 {
		fmt.Printf("  Pricing by location:\n")
		for _, price := range product.Prices {
			fmt.Printf("    %s: %.2f €/month", price.Location, price.Price.Net.Float64())
			if price.PriceSetup.Net.Float64() > 0 {
				fmt.Printf(" (%.2f € setup)", price.PriceSetup.Net.Float64())
			}
			fmt.Println()
		}
	}

	return nil
}

func orderProductServer(ctx context.Context, client *hrobot.Client, productID string, location string, sshKeyFingerprints []string, testMode bool, skipConfirmation bool) error {
	// Fetch the product list to find the product details
	fmt.Printf("Fetching product details...\n\n")
	products, err := client.Ordering.ListProducts(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch products: %w", err)
	}

	// Find the product with matching ID
	var product *hrobot.Product
	for i := range products {
		if products[i].ID == productID {
			product = &products[i]
			break
		}
	}

	if product == nil {
		return fmt.Errorf("product with ID %s not found", productID)
	}

	// Display server details
	fmt.Printf("Product Server Details:\n")
	fmt.Printf("  Product ID:  %s\n", product.ID)
	if product.Name != "" {
		fmt.Printf("  Name:        %s\n", product.Name)
	}

	// Show description array which contains the actual specs
	if len(product.Description) > 0 {
		fmt.Printf("  Specifications:\n")
		for _, desc := range product.Description {
			fmt.Printf("    - %s\n", desc)
		}
	}

	if product.Traffic != "" {
		fmt.Printf("  Traffic:     %s\n", product.Traffic)
	}

	// Show pricing per location
	if len(product.Prices) > 0 {
		fmt.Printf("  Pricing by location:\n")
		for _, price := range product.Prices {
			fmt.Printf("    %s: %.2f €/month", price.Location, price.Price.Net.Float64())
			if price.PriceSetup.Net.Float64() > 0 {
				fmt.Printf(" (%.2f € setup)", price.PriceSetup.Net.Float64())
			}
			fmt.Println()
		}
	}
	fmt.Println()

	// If no location specified, auto-select the cheapest one
	autoSelectedLocation := false
	if location == "" && len(product.Prices) > 0 {
		var lowestPrice float64
		for i, price := range product.Prices {
			monthlyPrice := price.Price.Net.Float64()
			if i == 0 || monthlyPrice < lowestPrice {
				lowestPrice = monthlyPrice
				location = price.Location
			}
		}
		autoSelectedLocation = true
	}

	// Show order configuration
	fmt.Printf("Order Configuration:\n")
	if location != "" {
		fmt.Printf("  Location:    %s\n", location)
	} else {
		fmt.Printf("  Location:    (not specified - order may fail)\n")
	}
	if len(sshKeyFingerprints) == 1 {
		fmt.Printf("  SSH Key:     %s\n", sshKeyFingerprints[0])
	} else {
		fmt.Printf("  SSH Keys:    %d keys\n", len(sshKeyFingerprints))
	}
	if testMode {
		fmt.Printf("  Test Mode:   enabled (order will not be placed)\n")
	}
	fmt.Println()

	// Show info if location was auto-selected
	if autoSelectedLocation {
		fmt.Printf("Selecting location %s as it's cheapest\n\n", location)
	}

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
	order := hrobot.ProductOrder{
		ProductID: productID,
		Auth: hrobot.AuthorizationMethod{
			Keys: sshKeyFingerprints,
		},
		Location:     location,
		Distribution: "Rescue system",
		Language:     "en",
		Test:         testMode,
	}

	fmt.Printf("Placing order...\n")
	tx, err := client.Ordering.PlaceProductOrder(ctx, order)
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	fmt.Printf("\n✓ Order placed successfully!\n")
	fmt.Printf("  Transaction ID: %s\n", tx.ID)
	fmt.Printf("  Status:         %s\n", tx.Status)
	fmt.Printf("  Date:           %s\n", tx.Date.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Product:        %s\n", tx.Product.Name)
	if tx.ServerNumber != nil {
		fmt.Printf("  Server Number:  %d\n", *tx.ServerNumber)
	}
	if tx.ServerIP != nil {
		fmt.Printf("  Server IP:      %s\n", *tx.ServerIP)
	}

	return nil
}
