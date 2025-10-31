// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Parse command line arguments
	if len(os.Args) < 2 {
		printHelp()
		return fmt.Errorf("no command specified")
	}

	command := os.Args[1]

	// Handle help flag
	if command == "--help" || command == "-h" || command == "help" {
		printHelp()
		return nil
	}

	// Get credentials from environment
	username := os.Getenv("HROBOT_USERNAME")
	password := os.Getenv("HROBOT_PASSWORD")

	if username == "" || password == "" {
		return fmt.Errorf(`HROBOT_USERNAME and HROBOT_PASSWORD environment variables must be set

To get your credentials:
  1. Visit: https://robot.hetzner.com/preferences/index
  2. Navigate to the 'Webservice and app settings' section
  3. Retrieve username and set new password

Note: If you want to order servers via the API, you also need to:
  1. Go to 'Webservice and app settings' -> 'Ordering'
  2. Enable 'ordering over the webservice'
  3. Click 'confirm' to save the setting

To limit API access to certain IP-address:
  1. Go to 'Webservice and app settings' -> 'Webservice/app access'
  2. Add your IP and save

Example usage:
  export HROBOT_USERNAME='#ws+XXXXXXX'
  export HROBOT_PASSWORD='YYYYYY'`)
	}

	// Create client
	client := hrobot.New(username, password)
	ctx := context.Background()

	switch command {
	case "server":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s server <subcommand>\nSubcommands:\n  list              - List all servers\n  describe <id>     - Describe server details by ID\n  reboot <id>       - Reboot server (hardware reset)\n  shutdown <id>     - Shutdown server\n  poweron <id>      - Power on server\n  poweroff <id>     - Power off server\n  enable-rescue <id> - Enable rescue system\n  disable-rescue <id> - Disable rescue system\n  traffic <id>      - Show traffic statistics\n  images <id>       - Show boot/image configuration\n  install <id>      - Install operating system on server", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "list":
			return enhanceAuthError(listServers(ctx, client))

		case "describe":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s server describe <server-id>\n\n", os.Args[0])
				fmt.Println("Describe detailed information about a specific server.")
				fmt.Println("\nArguments:")
				fmt.Println("  <server-id>    The server number to describe")
				return nil
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			return enhanceAuthError(getServer(ctx, client, hrobot.ServerID(serverID)))

		case "reboot":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s server reboot <server-id>\n\n", os.Args[0])
				fmt.Println("Reboot a server using hardware reset.")
				fmt.Println("\nArguments:")
				fmt.Println("  <server-id>    The server number to reboot")
				return nil
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			// Reboot is an alias for hardware reset
			return enhanceAuthError(executeReset(ctx, client, hrobot.ServerID(serverID), "hw"))

		case "shutdown":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s server shutdown <server-id> [--order-manual-power-cycle-from-technician]\n\n", os.Args[0])
				fmt.Println("Shutdown a server using long power button press.")
				fmt.Println("\nArguments:")
				fmt.Println("  <server-id>                                  The server number to shutdown")
				fmt.Println("\nFlags:")
				fmt.Println("  --order-manual-power-cycle-from-technician   Emails datacenter technician to manually turn the server off and on")
				return nil
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			// Check if manual power cycle flag is present
			manualPowerCycle := false
			for _, arg := range os.Args[4:] {
				if arg == "--order-manual-power-cycle-from-technician" {
					manualPowerCycle = true
					break
				}
			}
			// If manual flag is set, use 'man' reset type, otherwise use 'power_long'
			resetType := "power_long"
			if manualPowerCycle {
				resetType = "man"
			}
			return enhanceAuthError(executeReset(ctx, client, hrobot.ServerID(serverID), resetType))

		case "poweron":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s server poweron <server-id>\n\n", os.Args[0])
				fmt.Println("Power on a server.")
				fmt.Println("\nArguments:")
				fmt.Println("  <server-id>    The server number to power on")
				return nil
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			return enhanceAuthError(powerOnServer(ctx, client, hrobot.ServerID(serverID)))

		case "poweroff":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s server poweroff <server-id>\n\n", os.Args[0])
				fmt.Println("Power off a server.")
				fmt.Println("\nArguments:")
				fmt.Println("  <server-id>    The server number to power off")
				return nil
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			return enhanceAuthError(powerOffServer(ctx, client, hrobot.ServerID(serverID)))

		case "enable-rescue":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s server enable-rescue <server-id> [--linux|--vkvm] [--password]\n\n", os.Args[0])
				fmt.Println("Enable rescue system for a server.")
				fmt.Println("\nArguments:")
				fmt.Println("  <server-id>    The server number")
				fmt.Println("\nFlags:")
				fmt.Println("  --linux        Use Linux rescue system (default)")
				fmt.Println("  --vkvm         Use VNC/KVM rescue system")
				fmt.Println("  --password     Use password-based authentication instead of SSH keys")
				return nil
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			// Default to 'linux' if no OS flag is provided
			osType := "linux"
			usePassword := false
			// Check remaining arguments for flags
			for _, arg := range os.Args[4:] {
				switch arg {
				case "--password":
					usePassword = true
				case "--linux":
					osType = "linux"
				case "--vkvm":
					osType = "vkvm"
				}
			}
			return enhanceAuthError(activateRescue(ctx, client, hrobot.ServerID(serverID), osType, usePassword))

		case "disable-rescue":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s server disable-rescue <server-id>\n\n", os.Args[0])
				fmt.Println("Disable rescue system for a server.")
				fmt.Println("\nArguments:")
				fmt.Println("  <server-id>    The server number")
				return nil
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			return enhanceAuthError(deactivateRescue(ctx, client, hrobot.ServerID(serverID)))

		case "traffic":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s server traffic <server-id> [--days <n>] [--from <date>] [--to <date>]\n\n", os.Args[0])
				fmt.Println("Show traffic statistics for a server.")
				fmt.Println("\nArguments:")
				fmt.Println("  <server-id>    The server number")
				fmt.Println("\nFlags:")
				fmt.Println("  --days <n>     Number of days to show (default: 30)")
				fmt.Println("  --from <date>  Start date in YYYY-MM-DD format")
				fmt.Println("  --to <date>    End date in YYYY-MM-DD format")
				fmt.Println("\nNote: If --from and --to are specified, --days is ignored.")
				return nil
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			return enhanceAuthError(showTraffic(ctx, client, hrobot.ServerID(serverID), os.Args[4:]))

		case "images":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s server images <server-id>\n\n", os.Args[0])
				fmt.Println("Show boot/image configuration for a server.")
				fmt.Println("\nArguments:")
				fmt.Println("  <server-id>    The server number")
				return nil
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			return enhanceAuthError(getBootConfig(ctx, client, hrobot.ServerID(serverID)))

		case "install":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s server install <server-id> --linux=<distribution> [--lang=<language>] [--yes]\n", os.Args[0])
				fmt.Printf("       %s server install <server-id> --vnc=<distribution> [--lang=<language>] [--yes]\n\n", os.Args[0])
				fmt.Println("Install an operating system on a server.")
				fmt.Println("\nArguments:")
				fmt.Println("  <server-id>         The server number")
				fmt.Println("\nFlags:")
				fmt.Println("  --linux=<dist>      Install Linux distribution (e.g., --linux=ubuntu, --linux=debian)")
				fmt.Println("  --vnc=<dist>        Install via VNC (e.g., --vnc=centos)")
				fmt.Println("  --lang=<language>   Language code (default: en for Linux, en_US for VNC)")
				fmt.Println("  --yes               Skip confirmation prompt")
				fmt.Println("\nNote: The distribution name will be matched to the newest available version.")
				fmt.Println("      WARNING: This will format all drives on the server!")
				return nil
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			return enhanceAuthError(installOS(ctx, client, hrobot.ServerID(serverID), os.Args[4:]))

		default:
			return fmt.Errorf("unknown server subcommand: %s\nSubcommands:\n  list              - List all servers\n  describe <id>     - Describe server details by ID\n  reboot <id>       - Reboot server (hardware reset)\n  shutdown <id>     - Shutdown server\n  poweron <id>      - Power on server\n  poweroff <id>     - Power off server\n  enable-rescue <id> - Enable rescue system\n  disable-rescue <id> - Disable rescue system\n  traffic <id>      - Show traffic statistics\n  images <id>       - Show boot/image configuration\n  install <id>      - Install operating system on server", subcommand)
		}

	case "firewall":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s firewall <subcommand>\nSubcommands:\n  describe <server-id>           - Describe firewall configuration\n  allow <server-id> <ip>    - Add IP to firewall allow list", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "describe":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s firewall describe <server-id>", os.Args[0])
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			return enhanceAuthError(getFirewall(ctx, client, hrobot.ServerID(serverID)))

		case "allow":
			if len(os.Args) < 5 {
				return fmt.Errorf("usage: %s firewall allow <server-id> <ip-address>", os.Args[0])
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			ipAddr := os.Args[4]
			return enhanceAuthError(allowIP(ctx, client, hrobot.ServerID(serverID), ipAddr))

		default:
			return fmt.Errorf("unknown firewall subcommand: %s\nSubcommands:\n  describe <server-id>           - Describe firewall configuration\n  allow <server-id> <ip>    - Add IP to firewall allow list", subcommand)
		}

	case "ssh-key":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s ssh-key <subcommand>\nSubcommands:\n  list                     - List all SSH keys\n  describe <name>               - Describe SSH key details\n  create <name> <file|->   - Create a new SSH key from file or stdin\n  rename <name> <new-name> - Rename an SSH key\n  delete <name>            - Delete an SSH key", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "list":
			return enhanceAuthError(listKeys(ctx, client))

		case "describe":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s ssh-key describe <name>\n\n", os.Args[0])
				fmt.Println("Describe detailed information about a specific SSH key.")
				fmt.Println("\nArguments:")
				fmt.Println("  <name>    The name of the SSH key to describe")
				return nil
			}
			name := os.Args[3]
			return enhanceAuthError(getKey(ctx, client, name))

		case "create":
			if isHelpRequested() || len(os.Args) < 5 {
				fmt.Printf("Usage: %s ssh-key create <name> <file|->\n\n", os.Args[0])
				fmt.Println("Create a new SSH key from a file or stdin.")
				fmt.Println("\nArguments:")
				fmt.Println("  <name>      The name for the new SSH key")
				fmt.Println("  <file|->    Path to the public key file, or '-' to read from stdin")
				return nil
			}
			name := os.Args[3]
			keyPath := os.Args[4]
			return enhanceAuthError(createKey(ctx, client, name, keyPath))

		case "rename":
			if isHelpRequested() || len(os.Args) < 5 {
				fmt.Printf("Usage: %s ssh-key rename <name> <new-name>\n\n", os.Args[0])
				fmt.Println("Rename an existing SSH key.")
				fmt.Println("\nArguments:")
				fmt.Println("  <name>        The current name of the SSH key")
				fmt.Println("  <new-name>    The new name for the SSH key")
				return nil
			}
			name := os.Args[3]
			newName := os.Args[4]
			return enhanceAuthError(renameKey(ctx, client, name, newName))

		case "delete":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s ssh-key delete <name>\n\n", os.Args[0])
				fmt.Println("Delete an SSH key.")
				fmt.Println("\nArguments:")
				fmt.Println("  <name>    The name of the SSH key to delete")
				return nil
			}
			name := os.Args[3]
			return enhanceAuthError(deleteKey(ctx, client, name))

		default:
			return fmt.Errorf("unknown ssh-key subcommand: %s\nSubcommands:\n  list                     - List all SSH keys\n  describe <name>               - Describe SSH key details\n  create <name> <file|->   - Create a new SSH key from file or stdin\n  rename <name> <new-name> - Rename an SSH key\n  delete <name>            - Delete an SSH key", subcommand)
		}

	case "auction":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s auction <subcommand>\nSubcommands:\n  list                - List available auction servers\n  order <product-id> - Order a server from auction", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "list":
			return enhanceOrderingAuthError(ctx, client, listAuctionServers(ctx, client))

		case "order":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s auction order <product-id> [<ssh-key-name>] [--yes] [--test]\n\n", os.Args[0])
				fmt.Println("Order a server from the auction marketplace.")
				fmt.Println("\nArguments:")
				fmt.Println("  <product-id>      The auction server product ID")
				fmt.Println("  <ssh-key-name>    Optional: specific SSH key to use (default: all keys)")
				fmt.Println("\nFlags:")
				fmt.Println("  --yes             Skip confirmation prompt")
				fmt.Println("  --test            Test mode - does not actually place the order")
				return nil
			}
			productID, err := strconv.ParseUint(os.Args[3], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid product ID: %s", os.Args[3])
			}

			var sshKeyFingerprints []string
			var sshKeyName string
			testMode := false
			skipConfirmation := false

			// Parse remaining arguments for SSH key name and flags
			for i := 4; i < len(os.Args); i++ {
				arg := os.Args[i]
				switch arg {
				case "--test":
					testMode = true
				case "--yes":
					skipConfirmation = true
				default:
					// Must be SSH key name if it's not a flag
					if !strings.HasPrefix(arg, "--") && sshKeyName == "" {
						sshKeyName = arg
					}
				}
			}

			// Resolve SSH keys
			if sshKeyName != "" {
				// Specific SSH key provided
				sshKeyFingerprint, err := findKeyFingerprintByName(ctx, client, sshKeyName)
				if err != nil {
					return enhanceAuthError(err)
				}
				sshKeyFingerprints = []string{sshKeyFingerprint}
			} else {
				// No specific SSH key provided, query all SSH keys
				keys, err := client.Key.List(ctx)
				if err != nil {
					return fmt.Errorf("failed to query SSH keys: %w", err)
				}
				if len(keys) == 0 {
					return fmt.Errorf("no SSH keys found in your account. Please add at least one SSH key or specify a key name")
				}
				// Collect all fingerprints
				for _, key := range keys {
					sshKeyFingerprints = append(sshKeyFingerprints, key.Fingerprint)
				}
			}

			return enhanceOrderingAuthError(ctx, client, orderMarketServer(ctx, client, uint32(productID), sshKeyFingerprints, testMode, skipConfirmation))

		default:
			return fmt.Errorf("unknown auction subcommand: %s\nSubcommands:\n  list                - List available auction servers\n  order <product-id> - Order a server from auction", subcommand)
		}

	case "product":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s product <subcommand>\nSubcommands:\n  list                - List available product servers\n  order <product-id> - Order a product server", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "list":
			return enhanceOrderingAuthError(ctx, client, listProducts(ctx, client))

		case "order":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s product order <product-id> [--location=<datacenter>] [<ssh-key-name>] [--yes] [--test]\n\n", os.Args[0])
				fmt.Println("Order a product server (e.g., AX41-NVMe, EX40).")
				fmt.Println("\nArguments:")
				fmt.Println("  <product-id>      The product server ID (e.g., AX41-NVMe)")
				fmt.Println("  <ssh-key-name>    Optional: specific SSH key to use (default: all keys)")
				fmt.Println("\nFlags:")
				fmt.Println("  --location=<dc>   Datacenter location (e.g., FSN1, HEL1, NBG1)")
				fmt.Println("  --yes             Skip confirmation prompt")
				fmt.Println("  --test            Test mode - does not actually place the order")
				return nil
			}
			productID := os.Args[3]

			var sshKeyFingerprints []string
			var sshKeyName string
			var location string
			testMode := false
			skipConfirmation := false

			// Parse remaining arguments for SSH key name and flags
			for i := 4; i < len(os.Args); i++ {
				arg := os.Args[i]
				if strings.HasPrefix(arg, "--location=") {
					location = strings.TrimPrefix(arg, "--location=")
				} else if arg == "--test" {
					testMode = true
				} else if arg == "--yes" {
					skipConfirmation = true
				} else if !strings.HasPrefix(arg, "--") && sshKeyName == "" {
					// Must be SSH key name if it's not a flag
					sshKeyName = arg
				}
			}

			// Resolve SSH keys
			if sshKeyName != "" {
				// Specific SSH key provided
				sshKeyFingerprint, err := findKeyFingerprintByName(ctx, client, sshKeyName)
				if err != nil {
					return enhanceAuthError(err)
				}
				sshKeyFingerprints = []string{sshKeyFingerprint}
			} else {
				// No specific SSH key provided, query all SSH keys
				keys, err := client.Key.List(ctx)
				if err != nil {
					return fmt.Errorf("failed to query SSH keys: %w", err)
				}
				if len(keys) == 0 {
					return fmt.Errorf("no SSH keys found in your account. Please add at least one SSH key or specify a key name")
				}
				// Collect all fingerprints
				for _, key := range keys {
					sshKeyFingerprints = append(sshKeyFingerprints, key.Fingerprint)
				}
			}

			return enhanceOrderingAuthError(ctx, client, orderProductServer(ctx, client, productID, location, sshKeyFingerprints, testMode, skipConfirmation))

		default:
			return fmt.Errorf("unknown product subcommand: %s\nSubcommands:\n  list                - List available product servers\n  order <product-id> - Order a product server", subcommand)
		}

	case "rdns":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s rdns <subcommand>\nSubcommands:\n  list          - List all reverse DNS entries\n  describe <ip> - Describe reverse DNS entry for an IP\n  set <ip> <ptr> - Set reverse DNS entry for an IP\n  reset <ip>    - Reset reverse DNS entry for an IP", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "list":
			if isHelpRequested() {
				fmt.Printf("Usage: %s rdns list [--server-ip <ip>]\n\n", os.Args[0])
				fmt.Println("List all reverse DNS entries.")
				fmt.Println("\nFlags:")
				fmt.Println("  --server-ip <ip>    Filter by server IP address")
				return nil
			}
			serverIP := ""
			// Parse flags
			for i := 3; i < len(os.Args); i++ {
				if os.Args[i] == "--server-ip" && i+1 < len(os.Args) {
					serverIP = os.Args[i+1]
					i++ // Skip the next argument since it's the value
				}
			}
			return enhanceAuthError(listRDNS(ctx, client, serverIP))

		case "describe":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s rdns describe <ip>", os.Args[0])
			}
			ip := os.Args[3]
			return enhanceAuthError(getRDNS(ctx, client, ip))

		case "set":
			if len(os.Args) < 5 {
				return fmt.Errorf("usage: %s rdns set <ip> <ptr>", os.Args[0])
			}
			ip := os.Args[3]
			ptr := os.Args[4]
			return enhanceAuthError(setRDNS(ctx, client, ip, ptr))

		case "reset":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s rdns reset <ip>", os.Args[0])
			}
			ip := os.Args[3]
			return enhanceAuthError(deleteRDNS(ctx, client, ip))

		default:
			return fmt.Errorf("unknown rdns subcommand: %s\nSubcommands:\n  list          - List all reverse DNS entries\n  describe <ip> - Describe reverse DNS entry for an IP\n  set <ip> <ptr> - Set reverse DNS entry for an IP\n  reset <ip>    - Reset reverse DNS entry for an IP", subcommand)
		}

	case "failover":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s failover <subcommand>\nSubcommands:\n  list            - List all failover IPs\n  describe <ip>        - Describe failover IP details\n  set <ip> <dst>  - Route failover IP to destination server\n  delete <ip>     - Unroute failover IP", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "list":
			return enhanceAuthError(listFailovers(ctx, client))

		case "describe":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s failover describe <ip>", os.Args[0])
			}
			ip := os.Args[3]
			return enhanceAuthError(getFailover(ctx, client, ip))

		case "set":
			if len(os.Args) < 5 {
				return fmt.Errorf("usage: %s failover set <ip> <destination-server-ip>", os.Args[0])
			}
			ip := os.Args[3]
			destIP := os.Args[4]
			return enhanceAuthError(setFailover(ctx, client, ip, destIP))

		case "delete":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s failover delete <ip>", os.Args[0])
			}
			ip := os.Args[3]
			return enhanceAuthError(deleteFailover(ctx, client, ip))

		default:
			return fmt.Errorf("unknown failover subcommand: %s\nSubcommands:\n  list            - List all failover IPs\n  describe <ip>        - Describe failover IP details\n  set <ip> <dst>  - Route failover IP to destination server\n  delete <ip>     - Unroute failover IP", subcommand)
		}

	case "vswitch":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s vswitch <subcommand>\nSubcommands:\n  list                    - List all vSwitches\n  describe <id>           - Describe vSwitch details\n  create <name> <vlan>    - Create a new vSwitch\n  update <id> <name> <vlan> - Update vSwitch name and VLAN\n  delete <id>             - Cancel a vSwitch\n  add-server <id> <ip>    - Add server to vSwitch\n  remove-server <id> <ip> - Remove server from vSwitch", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "list":
			return enhanceAuthError(listVSwitches(ctx, client))

		case "describe":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s vswitch describe <id>\n\n", os.Args[0])
				fmt.Println("Describe detailed information about a specific vSwitch.")
				fmt.Println("\nArguments:")
				fmt.Println("  <id>    The vSwitch ID")
				return nil
			}
			id, err := strconv.Atoi(os.Args[3])
			if err != nil {
				return fmt.Errorf("invalid vSwitch ID: %s", os.Args[3])
			}
			return enhanceAuthError(getVSwitch(ctx, client, id))

		case "create":
			if isHelpRequested() || len(os.Args) < 5 {
				fmt.Printf("Usage: %s vswitch create <name> <vlan>\n\n", os.Args[0])
				fmt.Println("Create a new vSwitch.")
				fmt.Println("\nArguments:")
				fmt.Println("  <name>    Name for the vSwitch")
				fmt.Println("  <vlan>    VLAN ID (4000-4091)")
				return nil
			}
			name := os.Args[3]
			vlan, err := strconv.Atoi(os.Args[4])
			if err != nil {
				return fmt.Errorf("invalid VLAN ID: %s", os.Args[4])
			}
			return enhanceAuthError(createVSwitch(ctx, client, name, vlan))

		case "update":
			if isHelpRequested() || len(os.Args) < 6 {
				fmt.Printf("Usage: %s vswitch update <id> <name> <vlan>\n\n", os.Args[0])
				fmt.Println("Update a vSwitch name and VLAN.")
				fmt.Println("\nArguments:")
				fmt.Println("  <id>      The vSwitch ID")
				fmt.Println("  <name>    New name for the vSwitch")
				fmt.Println("  <vlan>    New VLAN ID (4000-4091)")
				return nil
			}
			id, err := strconv.Atoi(os.Args[3])
			if err != nil {
				return fmt.Errorf("invalid vSwitch ID: %s", os.Args[3])
			}
			name := os.Args[4]
			vlan, err := strconv.Atoi(os.Args[5])
			if err != nil {
				return fmt.Errorf("invalid VLAN ID: %s", os.Args[5])
			}
			return enhanceAuthError(updateVSwitch(ctx, client, id, name, vlan))

		case "delete":
			if isHelpRequested() || len(os.Args) < 4 {
				fmt.Printf("Usage: %s vswitch delete <id> [--immediate]\n\n", os.Args[0])
				fmt.Println("Cancel a vSwitch.")
				fmt.Println("\nArguments:")
				fmt.Println("  <id>          The vSwitch ID")
				fmt.Println("\nFlags:")
				fmt.Println("  --immediate   Cancel immediately (default: end of month)")
				return nil
			}
			id, err := strconv.Atoi(os.Args[3])
			if err != nil {
				return fmt.Errorf("invalid vSwitch ID: %s", os.Args[3])
			}
			immediate := false
			for _, arg := range os.Args[4:] {
				if arg == "--immediate" {
					immediate = true
				}
			}
			return enhanceAuthError(deleteVSwitch(ctx, client, id, immediate))

		case "add-server":
			if isHelpRequested() || len(os.Args) < 5 {
				fmt.Printf("Usage: %s vswitch add-server <id> <server-ip> [<server-ip>...]\n\n", os.Args[0])
				fmt.Println("Add one or more servers to a vSwitch.")
				fmt.Println("\nArguments:")
				fmt.Println("  <id>          The vSwitch ID")
				fmt.Println("  <server-ip>   Server IP address (can specify multiple)")
				return nil
			}
			id, err := strconv.Atoi(os.Args[3])
			if err != nil {
				return fmt.Errorf("invalid vSwitch ID: %s", os.Args[3])
			}
			servers := os.Args[4:]
			return enhanceAuthError(addServersToVSwitch(ctx, client, id, servers))

		case "remove-server":
			if isHelpRequested() || len(os.Args) < 5 {
				fmt.Printf("Usage: %s vswitch remove-server <id> <server-ip> [<server-ip>...]\n\n", os.Args[0])
				fmt.Println("Remove one or more servers from a vSwitch.")
				fmt.Println("\nArguments:")
				fmt.Println("  <id>          The vSwitch ID")
				fmt.Println("  <server-ip>   Server IP address (can specify multiple)")
				return nil
			}
			id, err := strconv.Atoi(os.Args[3])
			if err != nil {
				return fmt.Errorf("invalid vSwitch ID: %s", os.Args[3])
			}
			servers := os.Args[4:]
			return enhanceAuthError(removeServersFromVSwitch(ctx, client, id, servers))

		default:
			return fmt.Errorf("unknown vswitch subcommand: %s\nSubcommands:\n  list                    - List all vSwitches\n  describe <id>           - Describe vSwitch details\n  create <name> <vlan>    - Create a new vSwitch\n  update <id> <name> <vlan> - Update vSwitch name and VLAN\n  delete <id>             - Cancel a vSwitch\n  add-server <id> <ip>    - Add server to vSwitch\n  remove-server <id> <ip> - Remove server from vSwitch", subcommand)
		}

	default:
		printHelp()
		return fmt.Errorf("unknown command: %s", command)
	}
}

// isHelpRequested checks if --help or -h flag is present in arguments.
func isHelpRequested() bool {
	for _, arg := range os.Args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

// verifyCredentials checks if the credentials are valid by attempting to list SSH keys.
// Returns true if credentials work, false otherwise.
func verifyCredentials(ctx context.Context, client *hrobot.Client) bool {
	_, err := client.Key.List(ctx)
	return err == nil
}

// enhanceAuthError checks if an error is an authentication error and adds helpful instructions.
func enhanceAuthError(err error) error {
	if err == nil {
		return nil
	}

	// Check if this is an unauthorized error (check wrapped errors too)
	var hrobotErr *hrobot.Error
	if errors.As(err, &hrobotErr) && hrobot.IsUnauthorizedError(hrobotErr) {
		return fmt.Errorf(`%w

Authentication failed. Please verify your credentials are correct.

To get or reset your credentials:
  1. Visit: https://robot.hetzner.com/preferences/index
  2. Navigate to the 'Webservice and app settings' section
  3. Retrieve username and set new password

Note: If you want to order servers via the API, you also need to:
  1. Go to 'Webservice and app settings' -> 'Ordering'
  2. Enable 'ordering over the webservice'
  3. Click 'confirm' to save the setting

To limit API access to certain IP-address:
  1. Go to 'Webservice and app settings' -> 'Webservice/app access'
  2. Add your IP and save

Example usage:
  export HROBOT_USERNAME='#ws+XXXXXXX'
  export HROBOT_PASSWORD='YYYYYY'`, err)
	}

	return err
}

// enhanceOrderingAuthError checks if an error is an authentication error for ordering operations.
// It verifies if credentials are valid but ordering permission is missing.
func enhanceOrderingAuthError(ctx context.Context, client *hrobot.Client, err error) error {
	if err == nil {
		return nil
	}

	// Check if this is an unauthorized error
	var hrobotErr *hrobot.Error
	if errors.As(err, &hrobotErr) && hrobot.IsUnauthorizedError(hrobotErr) {
		// Verify if credentials work by testing with SSH keys endpoint
		if verifyCredentials(ctx, client) {
			// Credentials work, but ordering is not enabled
			return fmt.Errorf(`%w

Your credentials are valid, but ordering via the API is not enabled.

To enable ordering via the API:
  1. Visit: https://robot.hetzner.com/preferences/index
  2. Go to 'Webservice and app settings' -> 'Ordering'
  3. Enable 'ordering over the webservice'
  4. Click 'confirm' to save the setting`, err)
		}

		// Credentials don't work at all, show full authentication error
		return enhanceAuthError(err)
	}

	return err
}

func printHelp() {
	fmt.Printf(`hrobot - Command-line tool for Hetzner Robot API

Usage:
  hrobot [command] [subcommand] [args]

Available Commands:
  help                                       Show this help message

  Server Commands:
    server list                              List all servers
    server describe <id>                     Describe server details by ID
    server reboot <id>                       Reboot server (hardware reset)
    server shutdown <id>                     Shutdown server
    server poweron <id>                      Power on server
    server poweroff <id>                     Power off server
    server enable-rescue <id>                Enable rescue system
    server disable-rescue <id>               Disable rescue system
    server traffic <id>                      Show traffic statistics
    server images <id>                       Show boot/image configuration
    server install <id>                      Install operating system on server

  Firewall Commands:
    firewall describe <server-id>            Describe firewall configuration
    firewall allow <server-id> <ip>          Add IP to firewall allow list

  SSH Key Commands:
    ssh-key list                             List all SSH keys
    ssh-key describe <name>                  Describe SSH key details
    ssh-key create <name> <file|->           Create a new SSH key from file or stdin -
    ssh-key rename <name> <new-name>         Rename an SSH key
    ssh-key delete <name>                    Delete an SSH key

  Auction Commands:
    auction list                             List available auction servers
    auction order <product-id>               Order a server from auction

  Product Commands:
    product list                             List available product servers
    product order <product-id>               Order a product server

  Reverse DNS Commands:
    rdns list                                List all reverse DNS entries
    rdns describe <ip>                       Describe reverse DNS entry for an IP
    rdns set <ip> <ptr>                      Set reverse DNS entry for an IP
    rdns reset <ip>                          Use default Hetzner reverse DNS entry for an IP

  Failover IP Commands:
    failover list                            List all failover IPs
    failover describe <ip>                   Describe failover IP details
    failover set <ip> <destination-ip>       Route failover IP to destination server
    failover delete <ip>                     Unroute failover IP

  VSwitch Commands:
    vswitch list                             List all vSwitches
    vswitch describe <id>                    Describe vSwitch details
    vswitch create <name> <vlan>             Create a new vSwitch
    vswitch update <id> <name> <vlan>        Update vSwitch name and VLAN
    vswitch delete <id>                      Cancel a vSwitch
    vswitch add-server <id> <ip>             Add server to vSwitch
    vswitch remove-server <id> <ip>          Remove server from vSwitch

Environment Variables:
  HROBOT_USERNAME                            Your Hetzner Robot username (e.g., #ws+XXXXX)
  HROBOT_PASSWORD                            Your Hetzner Robot password

`)
}

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
	for _, server := range servers {
		fmt.Printf("Server #%d: %s\n", server.ServerNumber, server.ServerName)
		fmt.Printf("  IP:      %s\n", server.ServerIP.String())
		fmt.Printf("  Product: %s\n", server.Product)
		fmt.Printf("  DC:      %s\n", server.DC)
		fmt.Printf("  Status:  %s\n\n", server.Status)
	}

	return nil
}

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

func listKeys(ctx context.Context, client *hrobot.Client) error {
	keys, err := client.Key.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list SSH keys: %w", err)
	}

	fmt.Printf("Found %d SSH key(s):\n\n", len(keys))
	for i, key := range keys {
		fmt.Printf("[%d]\n", i+1)
		fmt.Printf("    Name:        %s\n", key.Name)
		fmt.Printf("    Fingerprint: %s\n", key.Fingerprint)
		fmt.Printf("    Type:        %s\n", key.Type)
		fmt.Printf("    Size:        %d bits\n", key.Size)
		fmt.Printf("    Created:     %s\n\n", key.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// findKeyFingerprintByName looks up a key by name and returns its fingerprint.
func findKeyFingerprintByName(ctx context.Context, client *hrobot.Client, name string) (string, error) {
	keys, err := client.Key.List(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list SSH keys: %w", err)
	}

	for _, key := range keys {
		if key.Name == name {
			return key.Fingerprint, nil
		}
	}

	return "", fmt.Errorf("SSH key with name '%s' not found", name)
}

func getKey(ctx context.Context, client *hrobot.Client, name string) error {
	// Look up the fingerprint by name
	fingerprint, err := findKeyFingerprintByName(ctx, client, name)
	if err != nil {
		return err
	}

	key, err := client.Key.Get(ctx, fingerprint)
	if err != nil {
		return fmt.Errorf("failed to get SSH key: %w", err)
	}

	fmt.Printf("SSH Key Details:\n")
	fmt.Printf("  Name:        %s\n", key.Name)
	fmt.Printf("  Fingerprint: %s\n", key.Fingerprint)
	fmt.Printf("  Type:        %s\n", key.Type)
	fmt.Printf("  Size:        %d bits\n", key.Size)
	fmt.Printf("  Created:     %s\n", key.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Data:        %s\n", key.Data)

	// Also output as JSON for easy parsing
	fmt.Println("\nJSON Output:")
	data, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))

	return nil
}

func createKey(ctx context.Context, client *hrobot.Client, name, keyPath string) error {
	var keyData string

	// Check if reading from stdin
	if keyPath == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		keyData = strings.TrimSpace(string(data))
	} else {
		// keyPath must be a valid file path
		data, err := os.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("failed to read key file '%s': %w (use '-' to read from stdin)", keyPath, err)
		}
		keyData = strings.TrimSpace(string(data))
	}

	key, err := client.Key.Create(ctx, name, keyData)
	if err != nil {
		return fmt.Errorf("failed to create SSH key: %w", err)
	}

	fmt.Printf("✓ Successfully created SSH key\n")
	fmt.Printf("  Name:        %s\n", key.Name)
	fmt.Printf("  Fingerprint: %s\n", key.Fingerprint)
	fmt.Printf("  Type:        %s\n", key.Type)
	fmt.Printf("  Size:        %d bits\n", key.Size)

	return nil
}

func renameKey(ctx context.Context, client *hrobot.Client, name, newName string) error {
	// Look up the fingerprint by name
	fingerprint, err := findKeyFingerprintByName(ctx, client, name)
	if err != nil {
		return err
	}

	key, err := client.Key.Rename(ctx, fingerprint, newName)
	if err != nil {
		return fmt.Errorf("failed to rename SSH key: %w", err)
	}

	fmt.Printf("✓ Successfully renamed SSH key\n")
	fmt.Printf("  Old Name:    %s\n", name)
	fmt.Printf("  New Name:    %s\n", key.Name)
	fmt.Printf("  Fingerprint: %s\n", key.Fingerprint)

	return nil
}

func deleteKey(ctx context.Context, client *hrobot.Client, name string) error {
	// Look up the fingerprint by name
	fingerprint, err := findKeyFingerprintByName(ctx, client, name)
	if err != nil {
		return err
	}

	err = client.Key.Delete(ctx, fingerprint)
	if err != nil {
		return fmt.Errorf("failed to delete SSH key: %w", err)
	}

	fmt.Printf("✓ Successfully deleted SSH key '%s' (fingerprint: %s)\n", name, fingerprint)

	return nil
}

func listAuctionServers(ctx context.Context, client *hrobot.Client) error {
	servers, err := client.Auction.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list auction servers: %w", err)
	}

	fmt.Printf("Found %d auction server(s):\n\n", len(servers))
	for i, server := range servers {
		fmt.Printf("[%d] %s (ID: %d)\n", i+1, server.Name, server.ID)
		fmt.Printf("    CPU:       %s (Benchmark: %d)\n", server.CPU, server.CPUBenchmark)
		fmt.Printf("    Memory:    %.0f GB\n", server.MemorySize)
		fmt.Printf("    Storage:   %s\n", server.HDDText)
		fmt.Printf("    Price:     %.2f €/month (%.2f € incl. VAT)\n", server.Price.Float64(), server.PriceVAT.Float64())
		fmt.Printf("    Setup:     %.2f € (%.2f € incl. VAT)\n", server.PriceSetup.Float64(), server.PriceSetupVAT.Float64())
		if server.Datacenter != nil {
			fmt.Printf("    Location:  %s\n", *server.Datacenter)
		}
		if server.FixedPrice {
			fmt.Printf("    Status:    Fixed price (lowest price reached)\n")
		} else if server.NextReduce > 0 {
			hours := server.NextReduce / 3600
			minutes := (server.NextReduce % 3600) / 60
			fmt.Printf("    Next cut:  in %dh %dm (%s)\n", hours, minutes, server.NextReduceDate)
		}
		fmt.Println()
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
		fmt.Printf("Boot Configuration for Server #%d (%s)\n", serverNumber, serverIP)
	} else {
		fmt.Printf("Boot Configuration\n")
	}
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Display headers
	fmt.Printf("%-18s │ %-6s │ %-35s │ %s\n", "Installation Type", "Active", "Distribution/OS", "Languages")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━┼━━━━━━━━┼━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┼━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

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
				fmt.Printf("%-18s │ %-6s │ %-35s │ %s\n", installType, status, os, lang)
			}
		} else {
			fmt.Printf("%-18s │ %-6s │ %-35s │\n", "Rescue System", activeStatus, "(no OS options)")
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
				fmt.Printf("%-18s │ %-6s │ %-35s │ %s\n", installType, status, dist, lang)
			}
		} else {
			fmt.Printf("%-18s │ %-6s │ %-35s │ %s\n", "Linux Install", activeStatus, "(no distributions)", languages)
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
				fmt.Printf("%-18s │ %-6s │ %-35s │ %s\n", installType, status, dist, lang)
			}
		} else {
			fmt.Printf("%-18s │ %-6s │ %-35s │ %s\n", "VNC Install", activeStatus, "(no distributions)", languages)
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
				fmt.Printf("%-18s │ %-6s │ %-35s │ %s\n", installType, status, os, lang)
			}
		} else {
			fmt.Printf("%-18s │ %-6s │ %-35s │ %s\n", "Windows Install", activeStatus, "(no OS options)", languages)
		}
	}

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
	fmt.Printf("Traffic Statistics (GB)\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Display headers
	fmt.Printf("%-10s │ %10s │ %10s │ Graph\n", "Date", "Download", "Upload")
	fmt.Printf("━━━━━━━━━━━┼━━━━━━━━━━━━┼━━━━━━━━━━━━┼━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// Determine scale
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

		// Display line
		fmt.Printf("%-10s │ %7.2f GB │ %7.2f GB │ %s\n",
			displayDate, traffic.In, traffic.Out, bar)
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Total Traffic: %.2f GB (↓%.2f GB in, ↑%.2f GB out)\n", totalSum, totalIn, totalOut)
	fmt.Printf("Average per day: %.2f GB\n", totalSum/float64(len(dates)))

	return nil
}

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

func listProducts(ctx context.Context, client *hrobot.Client) error {
	products, err := client.Ordering.ListProducts(ctx)
	if err != nil {
		return fmt.Errorf("failed to list products: %w", err)
	}

	fmt.Printf("Available Product Servers (%d found)\n", len(products))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Display headers
	fmt.Printf("%-15s │ %-38s │ %-20s │ %-18s │ %s\n", "Product ID", "Name", "Price from", "Setup Fee", "Locations")
	fmt.Printf("━━━━━━━━━━━━━━━┼━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┼━━━━━━━━━━━━━━━━━━━━┼━━━━━━━━━━━━━━━━━━┼━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

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

		fmt.Printf("%-15s │ %-38s │ %-20s │ %-18s │ %s\n",
			product.ID,
			truncateString(product.Name, 38),
			priceStr,
			setupStr,
			locations)
	}

	fmt.Printf("\nNote: Prices shown are the lowest available across all locations\n")
	fmt.Printf("      Use 'hrobot product order <product-id>' for full details and location-specific pricing\n")

	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
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
