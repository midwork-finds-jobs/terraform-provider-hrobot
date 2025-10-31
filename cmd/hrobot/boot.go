// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aquasecurity/table"
	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

func getBootConfig(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID) error {
	config, err := client.Boot.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get boot configuration: %w", err)
	}

	// Describe server number from any available config
	var serverNumber int
	var serverIP string
	if config.Rescue != nil {
		serverNumber = config.Rescue.ServerNumber
		serverIP = config.Rescue.ServerIP
	} else if config.Linux != nil {
		serverNumber = config.Linux.ServerNumber
		serverIP = config.Linux.ServerIP
	} else if config.VNC != nil {
		serverNumber = config.VNC.ServerNumber
		serverIP = config.VNC.ServerIP
	} else if config.Windows != nil {
		serverNumber = config.Windows.ServerNumber
		serverIP = config.Windows.ServerIP
	}

	if serverNumber > 0 {
		fmt.Printf("Boot Configuration for Server #%d (%s)\n\n", serverNumber, serverIP)
	} else {
		fmt.Printf("Boot Configuration\n\n")
	}

	// Create table
	t := table.New(nil)
	t.SetHeaders("Installation Type", "Active", "Distribution/OS", "Languages")

	// Rescue System
	if config.Rescue != nil {
		activeStatus := "No"
		if config.Rescue.Active {
			activeStatus = "Yes"
		}

		var osStrings []string
		if config.Rescue.OS != nil {
			if osSlice, ok := config.Rescue.OS.([]interface{}); ok {
				for _, os := range osSlice {
					if osStr, ok := os.(string); ok {
						osStrings = append(osStrings, osStr)
					}
				}
			}
		}

		// Special info for active rescue
		extraInfo := ""
		if config.Rescue.Active && config.Rescue.Password != nil && *config.Rescue.Password != "" {
			extraInfo = fmt.Sprintf(" (Password: %s)", *config.Rescue.Password)
		}
		if len(config.Rescue.AuthorizedKeys) > 0 {
			if extraInfo != "" {
				extraInfo += " "
			}
			extraInfo += fmt.Sprintf("(%d SSH key(s))", len(config.Rescue.AuthorizedKeys))
		}

		if len(osStrings) > 0 {
			for i, os := range osStrings {
				installType := ""
				status := ""
				if i == 0 {
					installType = "Rescue System"
					status = activeStatus
				}
				lang := ""
				if i == 0 && extraInfo != "" {
					lang = extraInfo
				}
				t.AddRow(installType, status, os, lang)
			}
		} else {
			t.AddRow("Rescue System", activeStatus, "(no OS options)", "")
		}
	}

	// Linux Installation
	if config.Linux != nil {
		activeStatus := "No"
		if config.Linux.Active {
			activeStatus = "Yes"
		}

		var distStrings []string
		var langStrings []string

		if config.Linux.Dist != nil {
			if distSlice, ok := config.Linux.Dist.([]interface{}); ok {
				for _, dist := range distSlice {
					if distStr, ok := dist.(string); ok {
						distStrings = append(distStrings, distStr)
					}
				}
			}
		}

		if config.Linux.Lang != nil {
			if langSlice, ok := config.Linux.Lang.([]interface{}); ok {
				for _, lang := range langSlice {
					if langStr, ok := lang.(string); ok {
						langStrings = append(langStrings, langStr)
					}
				}
			}
		}

		languages := strings.Join(langStrings, ", ")

		if len(distStrings) > 0 {
			for i, dist := range distStrings {
				installType := ""
				status := ""
				lang := ""
				if i == 0 {
					installType = "Linux Install"
					status = activeStatus
					lang = languages
					if config.Linux.Hostname != "" {
						lang += fmt.Sprintf(" | Hostname: %s", config.Linux.Hostname)
					}
				}
				t.AddRow(installType, status, dist, lang)
			}
		} else {
			t.AddRow("Linux Install", activeStatus, "(no distributions)", languages)
		}
	}

	// VNC Installation
	if config.VNC != nil {
		activeStatus := "No"
		if config.VNC.Active {
			activeStatus = "Yes"
		}

		var distStrings []string
		var langStrings []string

		if config.VNC.Dist != nil {
			if distSlice, ok := config.VNC.Dist.([]interface{}); ok {
				for _, dist := range distSlice {
					if distStr, ok := dist.(string); ok {
						distStrings = append(distStrings, distStr)
					}
				}
			}
		}

		if config.VNC.Lang != nil {
			if langSlice, ok := config.VNC.Lang.([]interface{}); ok {
				for _, lang := range langSlice {
					if langStr, ok := lang.(string); ok {
						langStrings = append(langStrings, langStr)
					}
				}
			}
		}

		languages := strings.Join(langStrings, ", ")

		if len(distStrings) > 0 {
			for i, dist := range distStrings {
				installType := ""
				status := ""
				lang := ""
				if i == 0 {
					installType = "VNC Install"
					status = activeStatus
					lang = languages
				}
				t.AddRow(installType, status, dist, lang)
			}
		} else {
			t.AddRow("VNC Install", activeStatus, "(no distributions)", languages)
		}
	}

	// Windows Installation
	if config.Windows != nil {
		activeStatus := "No"
		if config.Windows.Active {
			activeStatus = "Yes"
		}

		var osStrings []string
		var langStrings []string

		if config.Windows.OS != nil {
			if osSlice, ok := config.Windows.OS.([]interface{}); ok {
				for _, os := range osSlice {
					if osStr, ok := os.(string); ok {
						osStrings = append(osStrings, osStr)
					}
				}
			}
		}

		if config.Windows.Lang != nil {
			if langSlice, ok := config.Windows.Lang.([]interface{}); ok {
				for _, lang := range langSlice {
					if langStr, ok := lang.(string); ok {
						langStrings = append(langStrings, langStr)
					}
				}
			}
		}

		languages := strings.Join(langStrings, ", ")

		if len(osStrings) > 0 {
			for i, os := range osStrings {
				installType := ""
				status := ""
				lang := ""
				if i == 0 {
					installType = "Windows Install"
					status = activeStatus
					lang = languages
				}
				t.AddRow(installType, status, os, lang)
			}
		} else {
			t.AddRow("Windows Install", activeStatus, "(no OS options)", languages)
		}
	}

	t.Render()
	return nil
}

func activateRescue(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, os string, usePassword bool) error {
	fmt.Printf("Activating rescue system for server #%d...\n", serverID)
	fmt.Printf("  OS: %s\n", os)

	var authorizedKeys []string
	if !usePassword {
		// Query all SSH keys from the API
		keys, err := client.Key.List(ctx)
		if err != nil {
			fmt.Printf("  Warning: Failed to query SSH keys: %v\n", err)
			fmt.Println("  Falling back to password-based authentication")
		} else if len(keys) == 0 {
			fmt.Println("  No SSH keys found in your account")
			fmt.Println("  Using password-based authentication")
		} else {
			// Collect all fingerprints
			for _, key := range keys {
				authorizedKeys = append(authorizedKeys, key.Fingerprint)
			}
			fmt.Printf("  Adding %d SSH key(s) for authentication\n", len(authorizedKeys))
		}
	} else {
		fmt.Println("  Using password-based authentication")
	}
	fmt.Println()

	// Use default architecture 64-bit
	rescue, err := client.Boot.ActivateRescue(ctx, serverID, os, 64, authorizedKeys)
	if err != nil {
		return fmt.Errorf("failed to activate rescue system: %w", err)
	}

	fmt.Printf("✓ Rescue system activated successfully!\n")
	fmt.Printf("  OS:       %s\n", rescue.OS)
	fmt.Printf("  Active:   %v\n", rescue.Active)
	if len(authorizedKeys) > 0 {
		fmt.Printf("  SSH Keys: %d authorized\n", len(authorizedKeys))
	}
	if rescue.Password != nil && *rescue.Password != "" {
		fmt.Printf("  Password: %s\n", *rescue.Password)
		fmt.Println("\nIMPORTANT: Save the password above - it will not be shown again!")
	}
	fmt.Println("You need to reboot the server for the rescue system to become active.")

	return nil
}

func deactivateRescue(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID) error {
	err := client.Boot.DeactivateRescue(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to deactivate rescue system: %w", err)
	}

	fmt.Printf("✓ Successfully deactivated rescue system for server #%d\n", serverID)
	fmt.Println("You need to reboot the server for the change to take effect.")

	return nil
}

func installOS(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, args []string) error {
	// Parse flags
	var linuxDist, vncDist, lang string
	skipConfirmation := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--linux=") {
			linuxDist = strings.TrimPrefix(arg, "--linux=")
		} else if strings.HasPrefix(arg, "--vnc=") {
			vncDist = strings.TrimPrefix(arg, "--vnc=")
		} else if strings.HasPrefix(arg, "--lang=") {
			lang = strings.TrimPrefix(arg, "--lang=")
		} else if arg == "--yes" {
			skipConfirmation = true
		}
	}

	// Validate that exactly one of --linux or --vnc is specified
	if linuxDist == "" && vncDist == "" {
		return fmt.Errorf("must specify either --linux=<distribution> or --vnc=<distribution>")
	}
	if linuxDist != "" && vncDist != "" {
		return fmt.Errorf("cannot specify both --linux and --vnc, choose one")
	}

	// Get boot configuration to see available distributions
	config, err := client.Boot.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get boot configuration: %w", err)
	}

	if linuxDist != "" {
		return installLinux(ctx, client, serverID, config, linuxDist, lang, skipConfirmation)
	} else {
		return installVNC(ctx, client, serverID, config, vncDist, lang, skipConfirmation)
	}
}

func installLinux(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, config *hrobot.BootConfig, searchTerm string, lang string, skipConfirmation bool) error {
	if config.Linux == nil {
		return fmt.Errorf("linux installation not available for this server")
	}

	// Extract available distributions
	var availableDists []string
	if config.Linux.Dist != nil {
		if distSlice, ok := config.Linux.Dist.([]interface{}); ok {
			for _, dist := range distSlice {
				if distStr, ok := dist.(string); ok {
					availableDists = append(availableDists, distStr)
				}
			}
		}
	}

	if len(availableDists) == 0 {
		return fmt.Errorf("no linux distributions available")
	}

	// Find matching distribution (case-insensitive, pick newest)
	searchLower := strings.ToLower(searchTerm)
	var matches []string
	for _, dist := range availableDists {
		if strings.Contains(strings.ToLower(dist), searchLower) {
			matches = append(matches, dist)
		}
	}

	if len(matches) == 0 {
		fmt.Printf("No distributions found matching '%s'\n\n", searchTerm)
		fmt.Println("Available distributions:")
		for _, dist := range availableDists {
			fmt.Printf("  - %s\n", dist)
		}
		return fmt.Errorf("no matching distribution found")
	}

	// Sort matches and pick the last one (likely the newest)
	sort.Strings(matches)
	selectedDist := matches[len(matches)-1]

	// Set default language if not specified
	if lang == "" {
		lang = "en"
	}

	// Get SSH keys for authorization
	keys, err := client.Key.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to query SSH keys: %w", err)
	}

	var keyFingerprints []string
	for _, key := range keys {
		keyFingerprints = append(keyFingerprints, key.Fingerprint)
	}

	// Show installation details
	fmt.Printf("Installation Details:\n")
	fmt.Printf("  Server:       #%d\n", serverID)
	fmt.Printf("  Distribution: %s\n", selectedDist)
	fmt.Printf("  Language:     %s\n", lang)
	fmt.Printf("  SSH Keys:     %d key(s) will be authorized\n", len(keyFingerprints))
	fmt.Println()

	if len(matches) > 1 {
		fmt.Printf("Note: Multiple matches found, selected the newest: %s\n", selectedDist)
		fmt.Println("      Other matches:")
		for _, match := range matches[:len(matches)-1] {
			fmt.Printf("        - %s\n", match)
		}
		fmt.Println()
	}

	// Warning
	fmt.Printf("⚠️  WARNING: This will format ALL drives on server #%d!\n", serverID)
	fmt.Printf("⚠️  WARNING: All existing data will be permanently lost!\n")
	fmt.Printf("⚠️  WARNING: The server will be rebooted automatically!\n\n")

	// Confirmation
	if !skipConfirmation {
		fmt.Printf("Are you sure you want to install %s? (yes/no): ", selectedDist)
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			response = ""
		}
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "yes" {
			fmt.Println("Installation cancelled.")
			return nil
		}
		fmt.Println()
	}

	// Perform installation
	fmt.Printf("Installing %s...\n", selectedDist)
	result, err := client.Boot.ActivateLinux(ctx, serverID, selectedDist, 64, lang, keyFingerprints)
	if err != nil {
		return fmt.Errorf("failed to activate linux installation: %w", err)
	}

	fmt.Printf("\n✓ Linux installation activated successfully!\n")
	fmt.Printf("  Distribution: %s\n", selectedDist)
	fmt.Printf("  Active:       %v\n", result.Active)
	if len(result.AuthorizedKeys) > 0 {
		fmt.Printf("  SSH Keys:     %d authorized\n", len(result.AuthorizedKeys))
	}
	if result.Password != nil && *result.Password != "" {
		fmt.Printf("  Password:     %s\n", *result.Password)
	}
	fmt.Println("\nThe server will boot into the installer on next reboot.")
	fmt.Printf("You can reboot the server using: ./hrobot server reboot %d\n", serverID)

	return nil
}

func installVNC(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, config *hrobot.BootConfig, searchTerm string, lang string, skipConfirmation bool) error {
	if config.VNC == nil {
		return fmt.Errorf("VNC installation not available for this server")
	}

	// Extract available distributions
	var availableDists []string
	if config.VNC.Dist != nil {
		if distSlice, ok := config.VNC.Dist.([]interface{}); ok {
			for _, dist := range distSlice {
				if distStr, ok := dist.(string); ok {
					availableDists = append(availableDists, distStr)
				}
			}
		}
	}

	if len(availableDists) == 0 {
		return fmt.Errorf("no VNC distributions available")
	}

	// Find matching distribution (case-insensitive, pick newest)
	searchLower := strings.ToLower(searchTerm)
	var matches []string
	for _, dist := range availableDists {
		if strings.Contains(strings.ToLower(dist), searchLower) {
			matches = append(matches, dist)
		}
	}

	if len(matches) == 0 {
		fmt.Printf("No distributions found matching '%s'\n\n", searchTerm)
		fmt.Println("Available VNC distributions:")
		for _, dist := range availableDists {
			fmt.Printf("  - %s\n", dist)
		}
		return fmt.Errorf("no matching distribution found")
	}

	// Sort matches and pick the last one (likely the newest)
	sort.Strings(matches)
	selectedDist := matches[len(matches)-1]

	// Set default language if not specified
	if lang == "" {
		lang = "en_US"
	}

	// Show installation details
	fmt.Printf("VNC Installation Details:\n")
	fmt.Printf("  Server:       #%d\n", serverID)
	fmt.Printf("  Distribution: %s\n", selectedDist)
	fmt.Printf("  Language:     %s\n", lang)
	fmt.Println()

	if len(matches) > 1 {
		fmt.Printf("Note: Multiple matches found, selected the newest: %s\n", selectedDist)
		fmt.Println("      Other matches:")
		for _, match := range matches[:len(matches)-1] {
			fmt.Printf("        - %s\n", match)
		}
		fmt.Println()
	}

	// Warning
	fmt.Printf("⚠️  WARNING: This will format ALL drives on server #%d!\n", serverID)
	fmt.Printf("⚠️  WARNING: All existing data will be permanently lost!\n")
	fmt.Printf("⚠️  WARNING: The server will be rebooted automatically!\n\n")

	// Confirmation
	if !skipConfirmation {
		fmt.Printf("Are you sure you want to install %s via VNC? (yes/no): ", selectedDist)
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			response = ""
		}
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "yes" {
			fmt.Println("Installation cancelled.")
			return nil
		}
		fmt.Println()
	}

	// Perform installation
	fmt.Printf("Installing %s via VNC...\n", selectedDist)
	result, err := client.Boot.ActivateVNC(ctx, serverID, selectedDist, 64, lang)
	if err != nil {
		return fmt.Errorf("failed to activate VNC installation: %w", err)
	}

	fmt.Printf("\n✓ VNC installation activated successfully!\n")
	fmt.Printf("  Distribution: %s\n", selectedDist)
	fmt.Printf("  Active:       %v\n", result.Active)
	if result.Password != nil && *result.Password != "" {
		fmt.Printf("  VNC Password: %s\n", *result.Password)
		fmt.Println("\nIMPORTANT: Save the VNC password above - you'll need it to access the installer!")
	}
	fmt.Println("\nThe server will boot into the VNC installer on next reboot.")
	fmt.Printf("You can reboot the server using: ./hrobot server reboot %d\n", serverID)

	return nil
}
