// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
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
		return fmt.Errorf("HROBOT_USERNAME and HROBOT_PASSWORD environment variables must be set")
	}

	// Create client
	client := hrobot.New(username, password)
	ctx := context.Background()

	switch command {
	case "server":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s server <subcommand>\nSubcommands:\n  list        - List all servers\n  get <id>    - Get server details by ID", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "list":
			return listServers(ctx, client)

		case "get":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s server get <server-id>", os.Args[0])
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			return getServer(ctx, client, hrobot.ServerID(serverID))

		default:
			return fmt.Errorf("unknown server subcommand: %s\nSubcommands:\n  list        - List all servers\n  get <id>    - Get server details by ID", subcommand)
		}

	case "firewall":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s firewall <subcommand>\nSubcommands:\n  get <server-id>           - Get firewall configuration\n  allow <server-id> <ip>    - Add IP to firewall allow list", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "get":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s firewall get <server-id>", os.Args[0])
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			return getFirewall(ctx, client, hrobot.ServerID(serverID))

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
			return allowIP(ctx, client, hrobot.ServerID(serverID), ipAddr)

		default:
			return fmt.Errorf("unknown firewall subcommand: %s\nSubcommands:\n  get <server-id>           - Get firewall configuration\n  allow <server-id> <ip>    - Add IP to firewall allow list", subcommand)
		}

	case "reset":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s reset <subcommand>\nSubcommands:\n  get <server-id>           - Get reset options\n  execute <server-id> <type> - Execute reset (types: sw, hw, power, man)", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "get":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s reset get <server-id>", os.Args[0])
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			return getResetOptions(ctx, client, hrobot.ServerID(serverID))

		case "execute":
			if len(os.Args) < 5 {
				return fmt.Errorf("usage: %s reset execute <server-id> <type>\nTypes: sw (software), hw (hardware), power (power cycle), man (manual)", os.Args[0])
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			resetType := os.Args[4]
			return executeReset(ctx, client, hrobot.ServerID(serverID), resetType)

		default:
			return fmt.Errorf("unknown reset subcommand: %s\nSubcommands:\n  get <server-id>           - Get reset options\n  execute <server-id> <type> - Execute reset (types: sw, hw, power, man)", subcommand)
		}

	case "boot":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s boot <subcommand>\nSubcommands:\n  get <server-id>                        - Get boot configuration\n  rescue enable <server-id> [os]         - Activate rescue system (default: linux)\n  rescue disable <server-id>             - Deactivate rescue system", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "get":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s boot get <server-id>", os.Args[0])
			}
			serverIDStr := os.Args[3]
			serverID, err := strconv.Atoi(serverIDStr)
			if err != nil {
				return fmt.Errorf("invalid server ID: %s", serverIDStr)
			}
			return getBootConfig(ctx, client, hrobot.ServerID(serverID))

		case "rescue":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s boot rescue <enable|disable> ...\nSubcommands:\n  rescue enable <server-id> [os]   - Activate rescue system (default: linux)\n  rescue disable <server-id>       - Deactivate rescue system", os.Args[0])
			}

			action := os.Args[3]
			switch action {
			case "enable":
				if len(os.Args) < 5 {
					return fmt.Errorf("usage: %s boot rescue enable <server-id> [os]", os.Args[0])
				}
				serverIDStr := os.Args[4]
				serverID, err := strconv.Atoi(serverIDStr)
				if err != nil {
					return fmt.Errorf("invalid server ID: %s", serverIDStr)
				}
				// Default to 'linux' if os parameter is not provided
				osType := "linux"
				if len(os.Args) > 5 {
					osType = os.Args[5]
				}
				return activateRescue(ctx, client, hrobot.ServerID(serverID), osType)

			case "disable":
				if len(os.Args) < 5 {
					return fmt.Errorf("usage: %s boot rescue disable <server-id>", os.Args[0])
				}
				serverIDStr := os.Args[4]
				serverID, err := strconv.Atoi(serverIDStr)
				if err != nil {
					return fmt.Errorf("invalid server ID: %s", serverIDStr)
				}
				return deactivateRescue(ctx, client, hrobot.ServerID(serverID))

			default:
				return fmt.Errorf("unknown rescue action: %s\nSubcommands:\n  rescue enable <server-id> [os]   - Activate rescue system (default: linux)\n  rescue disable <server-id>       - Deactivate rescue system", action)
			}

		default:
			return fmt.Errorf("unknown boot subcommand: %s\nSubcommands:\n  get <server-id>                        - Get boot configuration\n  rescue enable <server-id> [os]         - Activate rescue system (default: linux)\n  rescue disable <server-id>             - Deactivate rescue system", subcommand)
		}

	case "key":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s key <subcommand>\nSubcommands:\n  list                          - List all SSH keys\n  get <fingerprint>             - Get SSH key details\n  create <name> <key-data>      - Create a new SSH key\n  rename <fingerprint> <name>   - Rename an SSH key\n  delete <fingerprint>          - Delete an SSH key", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "list":
			return listKeys(ctx, client)

		case "get":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s key get <fingerprint>", os.Args[0])
			}
			fingerprint := os.Args[3]
			return getKey(ctx, client, fingerprint)

		case "create":
			if len(os.Args) < 5 {
				return fmt.Errorf("usage: %s key create <name> <key-data>", os.Args[0])
			}
			name := os.Args[3]
			keyData := strings.Join(os.Args[4:], " ")
			return createKey(ctx, client, name, keyData)

		case "rename":
			if len(os.Args) < 5 {
				return fmt.Errorf("usage: %s key rename <fingerprint> <new-name>", os.Args[0])
			}
			fingerprint := os.Args[3]
			newName := os.Args[4]
			return renameKey(ctx, client, fingerprint, newName)

		case "delete":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s key delete <fingerprint>", os.Args[0])
			}
			fingerprint := os.Args[3]
			return deleteKey(ctx, client, fingerprint)

		default:
			return fmt.Errorf("unknown key subcommand: %s\nSubcommands:\n  list                          - List all SSH keys\n  get <fingerprint>             - Get SSH key details\n  create <name> <key-data>      - Create a new SSH key\n  rename <fingerprint> <name>   - Rename an SSH key\n  delete <fingerprint>          - Delete an SSH key", subcommand)
		}

	case "auction":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s auction <subcommand>\nSubcommands:\n  list    - List available auction servers", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "list":
			return listAuctionServers(ctx, client)

		default:
			return fmt.Errorf("unknown auction subcommand: %s\nSubcommands:\n  list    - List available auction servers", subcommand)
		}

	case "order":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s order <subcommand>\nSubcommands:\n  market <product-id> <ssh-key-fingerprint> [--test]  - Order a server from marketplace", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "market":
			if len(os.Args) < 5 {
				return fmt.Errorf("usage: %s order market <product-id> <ssh-key-fingerprint> [--test]", os.Args[0])
			}
			productID, err := strconv.ParseUint(os.Args[3], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid product ID: %s", os.Args[3])
			}
			sshKeyFingerprint := os.Args[4]
			testMode := len(os.Args) > 5 && os.Args[5] == "--test"
			return orderMarketServer(ctx, client, uint32(productID), sshKeyFingerprint, testMode)

		default:
			return fmt.Errorf("unknown order subcommand: %s\nSubcommands:\n  market <product-id> <ssh-key-fingerprint> [--test]  - Order a server from marketplace", subcommand)
		}

	case "rdns":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s rdns <subcommand>\nSubcommands:\n  list [server-ip] - List all reverse DNS entries\n  get <ip>         - Get reverse DNS entry for an IP\n  set <ip> <ptr>   - Set reverse DNS entry for an IP\n  delete <ip>      - Delete reverse DNS entry for an IP", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "list":
			serverIP := ""
			if len(os.Args) > 3 {
				serverIP = os.Args[3]
			}
			return listRDNS(ctx, client, serverIP)

		case "get":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s rdns get <ip>", os.Args[0])
			}
			ip := os.Args[3]
			return getRDNS(ctx, client, ip)

		case "set":
			if len(os.Args) < 5 {
				return fmt.Errorf("usage: %s rdns set <ip> <ptr>", os.Args[0])
			}
			ip := os.Args[3]
			ptr := os.Args[4]
			return setRDNS(ctx, client, ip, ptr)

		case "delete":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s rdns delete <ip>", os.Args[0])
			}
			ip := os.Args[3]
			return deleteRDNS(ctx, client, ip)

		default:
			return fmt.Errorf("unknown rdns subcommand: %s\nSubcommands:\n  list [server-ip] - List all reverse DNS entries\n  get <ip>         - Get reverse DNS entry for an IP\n  set <ip> <ptr>   - Set reverse DNS entry for an IP\n  delete <ip>      - Delete reverse DNS entry for an IP", subcommand)
		}

	case "failover":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: %s failover <subcommand>\nSubcommands:\n  list            - List all failover IPs\n  get <ip>        - Get failover IP details\n  set <ip> <dst>  - Route failover IP to destination server\n  delete <ip>     - Unroute failover IP", os.Args[0])
		}

		subcommand := os.Args[2]
		switch subcommand {
		case "list":
			return listFailovers(ctx, client)

		case "get":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s failover get <ip>", os.Args[0])
			}
			ip := os.Args[3]
			return getFailover(ctx, client, ip)

		case "set":
			if len(os.Args) < 5 {
				return fmt.Errorf("usage: %s failover set <ip> <destination-server-ip>", os.Args[0])
			}
			ip := os.Args[3]
			destIP := os.Args[4]
			return setFailover(ctx, client, ip, destIP)

		case "delete":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: %s failover delete <ip>", os.Args[0])
			}
			ip := os.Args[3]
			return deleteFailover(ctx, client, ip)

		default:
			return fmt.Errorf("unknown failover subcommand: %s\nSubcommands:\n  list            - List all failover IPs\n  get <ip>        - Get failover IP details\n  set <ip> <dst>  - Route failover IP to destination server\n  delete <ip>     - Unroute failover IP", subcommand)
		}

	default:
		printHelp()
		return fmt.Errorf("unknown command: %s", command)
	}
}

func printHelp() {
	fmt.Printf(`hrobot - Command-line tool for Hetzner Robot API

Usage:
  hrobot [command] [subcommand] [args]

Available Commands:
  help                                       Show this help message

  Server Commands:
    server list                              List all servers
    server get <id>                          Get server details by ID

  Firewall Commands:
    firewall get <server-id>                 Get firewall configuration
    firewall allow <server-id> <ip>          Add IP to firewall allow list

  Reset Commands:
    reset get <server-id>                    Get reset options for server
    reset execute <server-id> <type>         Execute reset (sw/hw/power/man)

  Boot Configuration Commands:
    boot get <server-id>                     Get boot configuration
    boot rescue enable <server-id> [os]      Activate rescue system (default: linux, options: linux, vkvm)
    boot rescue disable <server-id>          Deactivate rescue system

  SSH Key Commands:
    key list                                 List all SSH keys
    key get <fingerprint>                    Get SSH key details
    key create <name> <key-data>             Create a new SSH key
    key rename <fingerprint> <new-name>      Rename an SSH key
    key delete <fingerprint>                 Delete an SSH key

  Auction Commands:
    auction list                             List available auction servers

  Order Commands:
    order market <product-id> <ssh-key-fingerprint> [--test]
                                             Order a server from marketplace

  Reverse DNS Commands:
    rdns list [server-ip]                    List all reverse DNS entries (optionally filtered by server IP)
    rdns get <ip>                            Get reverse DNS entry for an IP
    rdns set <ip> <ptr>                      Set reverse DNS entry for an IP
    rdns delete <ip>                         Delete reverse DNS entry for an IP

  Failover IP Commands:
    failover list                            List all failover IPs
    failover get <ip>                        Get failover IP details
    failover set <ip> <destination-ip>       Route failover IP to destination server
    failover delete <ip>                     Unroute failover IP

Environment Variables:
  HROBOT_USERNAME                            Your Hetzner Robot username (e.g., #ws+XXXXX)
  HROBOT_PASSWORD                            Your Hetzner Robot password

Examples:
  # List all servers
  hrobot server list

  # Get details for a specific server
  hrobot server get 1234567

  # List SSH keys
  hrobot key list

  # View firewall configuration
  hrobot firewall get 1234567

  # List available auction servers
  hrobot auction list

`)
}

func getServer(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID) error {
	server, err := client.Server.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	// Pretty print the server details
	fmt.Printf("Server Details:\n")
	fmt.Printf("  Server Number: %d\n", server.ServerNumber)
	fmt.Printf("  Server Name:   %s\n", server.ServerName)
	fmt.Printf("  Server IP:     %s\n", server.ServerIP.String())
	fmt.Printf("  Product:       %s\n", server.Product)
	fmt.Printf("  DC:            %s\n", server.DC)
	fmt.Printf("  Status:        %s\n", server.Status)
	fmt.Printf("  Traffic:       %s\n", server.Traffic.String())
	fmt.Printf("  Cancelled:     %v\n", server.Cancelled)
	fmt.Printf("  Paid Until:    %s\n", server.PaidUntil)

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

	// Also output as JSON for easy parsing
	fmt.Println("\nJSON Output:")
	data, err := json.MarshalIndent(server, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))

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
		fmt.Printf("[%d] %s\n", i+1, key.Name)
		fmt.Printf("    Fingerprint: %s\n", key.Fingerprint)
		fmt.Printf("    Type:        %s\n", key.Type)
		fmt.Printf("    Size:        %d bits\n", key.Size)
		fmt.Printf("    Created:     %s\n\n", key.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func getKey(ctx context.Context, client *hrobot.Client, fingerprint string) error {
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

func createKey(ctx context.Context, client *hrobot.Client, name, keyData string) error {
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

func renameKey(ctx context.Context, client *hrobot.Client, fingerprint, newName string) error {
	key, err := client.Key.Rename(ctx, fingerprint, newName)
	if err != nil {
		return fmt.Errorf("failed to rename SSH key: %w", err)
	}

	fmt.Printf("✓ Successfully renamed SSH key\n")
	fmt.Printf("  Fingerprint: %s\n", key.Fingerprint)
	fmt.Printf("  New Name:    %s\n", key.Name)

	return nil
}

func deleteKey(ctx context.Context, client *hrobot.Client, fingerprint string) error {
	err := client.Key.Delete(ctx, fingerprint)
	if err != nil {
		return fmt.Errorf("failed to delete SSH key: %w", err)
	}

	fmt.Printf("✓ Successfully deleted SSH key %s\n", fingerprint)

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

	// Get server number from any available config
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

func getResetOptions(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID) error {
	reset, err := client.Reset.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get reset options: %w", err)
	}

	fmt.Printf("Reset Options for Server #%d:\n", reset.ServerNumber)
	fmt.Printf("  Server IP:        %s\n", reset.ServerIP.String())
	fmt.Printf("  Available Types:  %s\n", reset.Type)
	if reset.OperatingStatus != "" {
		fmt.Printf("  Operating Status: %s\n", reset.OperatingStatus)
	}

	fmt.Println("\nAvailable Reset Types:")
	fmt.Println("  sw    - Software reset (CTRL+ALT+DEL)")
	fmt.Println("  hw    - Hardware reset (reset button)")
	fmt.Println("  power - Power cycle (short press power button)")
	fmt.Println("  man   - Manual reset")

	// Also output as JSON for easy parsing
	fmt.Println("\nJSON Output:")
	data, err := json.MarshalIndent(reset, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))

	return nil
}

func executeReset(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, resetType string) error {
	// Validate reset type
	validTypes := map[string]string{
		"sw":    "software reset (CTRL+ALT+DEL)",
		"hw":    "hardware reset (reset button)",
		"power": "power cycle",
		"man":   "manual reset",
	}

	description, valid := validTypes[resetType]
	if !valid {
		return fmt.Errorf("invalid reset type: %s\nValid types: sw, hw, power, man", resetType)
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
