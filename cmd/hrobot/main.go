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
	"strconv"
	"strings"

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
			return fmt.Errorf("usage: %s server <subcommand>\nSubcommands:\n  list                                         - List all servers\n  describe <id>                                - Describe server details by ID\n  reboot <id>                                  - Reboot server (hardware reset)\n  shutdown <id> [--order-manual-power-cycle-from-technician]\n                                               - Shutdown server (long power press or order manual reset)\n  poweron <id>                                 - Power on server\n  poweroff <id>                                - Power off server\n  enable-rescue <id> [os]                      - Enable rescue system (default: linux)\n  disable-rescue <id>                          - Disable rescue system", os.Args[0])
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
				fmt.Println("  --order-manual-power-cycle-from-technician   Order manual power cycle from datacenter technician")
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
				fmt.Printf("Usage: %s server enable-rescue <server-id> [os]\n\n", os.Args[0])
				fmt.Println("Enable rescue system for a server.")
				fmt.Println("\nArguments:")
				fmt.Println("  <server-id>    The server number")
				fmt.Println("  [os]           OS type for rescue system (default: linux, options: linux, vkvm)")
				return nil
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			// Default to 'linux' if os parameter is not provided
			osType := "linux"
			if len(os.Args) > 4 {
				osType = os.Args[4]
			}
			return enhanceAuthError(activateRescue(ctx, client, hrobot.ServerID(serverID), osType))

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

		default:
			return fmt.Errorf("unknown server subcommand: %s\nSubcommands:\n  list                                         - List all servers\n  describe <id>                                - Describe server details by ID\n  reboot <id>                                  - Reboot server (hardware reset)\n  shutdown <id> [--order-manual-power-cycle-from-technician]\n                                               - Shutdown server (long power press or order manual reset)\n  poweron <id>                                 - Power on server\n  poweroff <id>                                - Power off server\n  enable-rescue <id> [os]                      - Enable rescue system (default: linux)\n  disable-rescue <id>                          - Disable rescue system", subcommand)
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

	case "boot":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s boot <subcommand>\nSubcommands:\n  describe <server-id>  - Describe boot configuration", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "describe":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s boot describe <server-id>", os.Args[0])
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			return enhanceAuthError(getBootConfig(ctx, client, hrobot.ServerID(serverID)))

		default:
			return fmt.Errorf("unknown boot subcommand: %s\nSubcommands:\n  describe <server-id>  - Describe boot configuration", subcommand)
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
			return fmt.Errorf("usage: %s auction <subcommand>\nSubcommands:\n  list                                            - List available auction servers\n  purchase <product-id> <ssh-key-name> [--test]  - Purchase a server from auction", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "list":
			return enhanceOrderingAuthError(ctx, client, listAuctionServers(ctx, client))

		case "purchase":
			if len(os.Args) < 5 {
				return fmt.Errorf("usage: %s auction purchase <product-id> <ssh-key-name> [--test]", os.Args[0])
			}
			productID, err := strconv.ParseUint(os.Args[3], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid product ID: %s", os.Args[3])
			}
			sshKeyName := os.Args[4]
			// Look up the fingerprint by name
			sshKeyFingerprint, err := findKeyFingerprintByName(ctx, client, sshKeyName)
			if err != nil {
				return enhanceAuthError(err)
			}
			testMode := len(os.Args) > 5 && os.Args[5] == "--test"
			return enhanceOrderingAuthError(ctx, client, orderMarketServer(ctx, client, uint32(productID), sshKeyFingerprint, testMode))

		default:
			return fmt.Errorf("unknown auction subcommand: %s\nSubcommands:\n  list                                            - List available auction servers\n  purchase <product-id> <ssh-key-name> [--test]  - Purchase a server from auction", subcommand)
		}

	case "rdns":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s rdns <subcommand>\nSubcommands:\n  list [server-ip] - List all reverse DNS entries\n  describe <ip>         - Describe reverse DNS entry for an IP\n  set <ip> <ptr>   - Set reverse DNS entry for an IP\n  reset <ip>       - Reset reverse DNS entry for an IP", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "list":
			serverIP := ""
			if len(os.Args) > 3 {
				serverIP = os.Args[3]
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
			return fmt.Errorf("unknown rdns subcommand: %s\nSubcommands:\n  list [server-ip] - List all reverse DNS entries\n  describe <ip>         - Describe reverse DNS entry for an IP\n  set <ip> <ptr>   - Set reverse DNS entry for an IP\n  reset <ip>       - Reset reverse DNS entry for an IP", subcommand)
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
    server shutdown <id> [--order-manual-power-cycle-from-technician]
                                             Shutdown server (long power press or order manual reset)
    server poweron <id>                      Power on server
    server poweroff <id>                     Power off server
    server enable-rescue <id> [os]           Enable rescue system (default: linux, options: linux, vkvm)
    server disable-rescue <id>               Disable rescue system

  Firewall Commands:
    firewall describe <server-id>            Describe firewall configuration
    firewall allow <server-id> <ip>          Add IP to firewall allow list

  Boot Configuration Commands:
    boot describe <server-id>                Describe boot configuration

  SSH Key Commands:
    ssh-key list                             List all SSH keys
    ssh-key describe <name>                  Describe SSH key details
    ssh-key create <name> <file|->           Create a new SSH key from file or stdin
    ssh-key rename <name> <new-name>         Rename an SSH key
    ssh-key delete <name>                    Delete an SSH key

  Auction Commands:
    auction list                             List available auction servers
    auction purchase <server-id> <ssh-key>   Purchase a server from auction

  Reverse DNS Commands:
    rdns list [server-ip]                    List all reverse DNS entries (optionally filtered by server IP)
    rdns describe <ip>                       Describe reverse DNS entry for an IP
    rdns set <ip> <ptr>                      Set reverse DNS entry for an IP
    rdns reset <ip>                          Use default Hetzner reverse DNS entry for an IP

  Failover IP Commands:
    failover list                            List all failover IPs
    failover describe <ip>                   Describe failover IP details
    failover set <ip> <destination-ip>       Route failover IP to destination server
    failover delete <ip>                     Unroute failover IP

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
		fmt.Printf("    Price:     €%.2f/month (€%.2f incl. VAT)\n", server.Price, server.PriceVAT)
		fmt.Printf("    Setup:     €%.2f (€%.2f incl. VAT)\n", server.PriceSetup, server.PriceSetupVAT)
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

func orderMarketServer(ctx context.Context, client *hrobot.Client, productID uint32, sshKeyFingerprint string, testMode bool) error {
	order := hrobot.MarketProductOrder{
		ProductID: productID,
		Auth: hrobot.AuthorizationMethod{
			Keys: []string{sshKeyFingerprint},
		},
		Distribution: "Rescue system",
		Language:     "en",
		Test:         testMode,
	}

	fmt.Printf("Ordering server from marketplace...\n")
	fmt.Printf("  Product ID:  %d\n", productID)
	fmt.Printf("  SSH Key:     %s\n", sshKeyFingerprint)
	fmt.Printf("  Test Mode:   %v\n", testMode)
	fmt.Println()

	tx, err := client.Ordering.PlaceMarketOrder(ctx, order)
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	fmt.Printf("✓ Order placed successfully!\n")
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
		fmt.Printf("Boot Configuration for Server #%d:\n", serverNumber)
		fmt.Printf("  Server IP: %s\n\n", serverIP)
	} else {
		fmt.Printf("Boot Configuration:\n\n")
	}

	// Rescue System
	if config.Rescue != nil {
		fmt.Printf("Rescue System:\n")
		fmt.Printf("  Active:    %v\n", config.Rescue.Active)
		if config.Rescue.OS != nil {
			fmt.Printf("  OS:        %v\n", config.Rescue.OS)
		}
		if config.Rescue.Active && config.Rescue.Password != nil && *config.Rescue.Password != "" {
			fmt.Printf("  Password:  %s\n", *config.Rescue.Password)
		}
		if len(config.Rescue.AuthorizedKeys) > 0 {
			fmt.Printf("  SSH Keys:  %d authorized\n", len(config.Rescue.AuthorizedKeys))
		}
		fmt.Println()
	}

	// Linux Installation
	if config.Linux != nil {
		fmt.Printf("Linux Installation:\n")
		fmt.Printf("  Active:       %v\n", config.Linux.Active)
		if config.Linux.Dist != nil {
			fmt.Printf("  Distribution: %v\n", config.Linux.Dist)
		}
		if config.Linux.Arch != nil {
			fmt.Printf("  Architecture: %v\n", config.Linux.Arch)
		}
		if config.Linux.Lang != nil {
			fmt.Printf("  Language:     %v\n", config.Linux.Lang)
		}
		if config.Linux.Hostname != "" {
			fmt.Printf("  Hostname:     %s\n", config.Linux.Hostname)
		}
		fmt.Println()
	}

	// VNC Installation
	if config.VNC != nil {
		fmt.Printf("VNC Installation:\n")
		fmt.Printf("  Active: %v\n", config.VNC.Active)
		if config.VNC.Dist != nil {
			fmt.Printf("  Distribution: %v\n", config.VNC.Dist)
		}
		if config.VNC.Arch != nil {
			fmt.Printf("  Architecture: %v\n", config.VNC.Arch)
		}
		if config.VNC.Lang != nil {
			fmt.Printf("  Language:     %v\n", config.VNC.Lang)
		}
		fmt.Println()
	}

	// Windows Installation
	if config.Windows != nil {
		fmt.Printf("Windows Installation:\n")
		fmt.Printf("  Active:   %v\n", config.Windows.Active)
		if config.Windows.OS != nil {
			fmt.Printf("  OS:       %v\n", config.Windows.OS)
		}
		if config.Windows.Lang != nil {
			fmt.Printf("  Language: %v\n", config.Windows.Lang)
		}
		fmt.Println()
	}

	// Also output as JSON for easy parsing
	fmt.Println("JSON Output:")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))

	return nil
}

func activateRescue(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, os string) error {
	fmt.Printf("Activating rescue system for server #%d...\n", serverID)
	fmt.Printf("  OS: %s\n\n", os)

	// Use default architecture 64-bit and no SSH keys
	rescue, err := client.Boot.ActivateRescue(ctx, serverID, os, 64, nil)
	if err != nil {
		return fmt.Errorf("failed to activate rescue system: %w", err)
	}

	fmt.Printf("✓ Rescue system activated successfully!\n")
	fmt.Printf("  OS:       %s\n", rescue.OS)
	fmt.Printf("  Active:   %v\n", rescue.Active)
	if rescue.Password != nil {
		fmt.Printf("  Password: %s\n", *rescue.Password)
	}
	fmt.Println("\nIMPORTANT: Save the password above - it will not be shown again!")
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
