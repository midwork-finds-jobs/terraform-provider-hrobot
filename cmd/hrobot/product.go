// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aquasecurity/table"
	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

func listProducts(ctx context.Context, client *hrobot.Client) error {
	products, err := client.Ordering.ListProducts(ctx)
	if err != nil {
		return fmt.Errorf("failed to list products: %w", err)
	}

	fmt.Printf("Available Product Servers (%d found)\n\n", len(products))

	// Create table
	t := table.New(nil)
	t.SetHeaders("Product ID", "Name", "Price from", "Setup Fee", "Locations")

	for _, product := range products {
		locations := strings.Join(product.Locations, ", ")
		if locations == "" {
			locations = "-"
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

		priceStr := fmt.Sprintf("%.2f €/month", lowestPrice)
		setupStr := fmt.Sprintf("%.2f €", lowestSetup)

		t.AddRow(
			product.ID,
			product.Name,
			priceStr,
			setupStr,
			locations,
		)
	}

	t.Render()

	fmt.Printf("\nNote: Prices shown are the lowest available across all locations\n")
	fmt.Printf("      Use 'hrobot product order <product-id>' for full details and location-specific pricing\n")

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
