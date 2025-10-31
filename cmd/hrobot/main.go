// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

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

	// Route commands to their handlers
	switch command {
	case "server":
		return handleServerCommand(ctx, client)

	case "firewall":
		return handleFirewallCommand(ctx, client)

	case "ssh-key":
		return handleSSHKeyCommand(ctx, client)

	case "rdns":
		return handleRDNSCommand(ctx, client)

	case "failover":
		return handleFailoverCommand(ctx, client)

	case "vswitch":
		return handleVSwitchCommand(ctx, client)

	case "auction":
		return handleAuctionCommand(ctx, client)

	case "product":
		return handleProductCommand(ctx, client)

	default:
		printHelp()
		return fmt.Errorf("unknown command: %s", command)
	}
}

// handleServerCommand handles all server-related subcommands.
func handleServerCommand(ctx context.Context, client *hrobot.Client) error {
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
		manualPowerCycle := false
		for _, arg := range os.Args[4:] {
			if arg == "--order-manual-power-cycle-from-technician" {
				manualPowerCycle = true
				break
			}
		}
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
		osType := "linux"
		usePassword := false
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
}

// handleFirewallCommand handles all firewall-related subcommands.
func handleFirewallCommand(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: %s firewall <subcommand>\nSubcommands:\n  describe <server-id>      - Describe firewall configuration\n  allow <server-id> <ip>    - Add IP to firewall allow list", os.Args[0])
	}

	subcommand := os.Args[2]
	switch subcommand {
	case "describe":
		if len(os.Args) < 4 {
			fmt.Printf("Usage: %s firewall describe <server-id>\n\n", os.Args[0])
			fmt.Println("Describe firewall configuration for a server.")
			fmt.Println("\nArguments:")
			fmt.Println("  <server-id>    The server number")
			return nil
		}
		serverIDStr := os.Args[3]
		serverID, err := strconv.Atoi(serverIDStr)
		if err != nil {
			return fmt.Errorf("invalid server ID: %s", serverIDStr)
		}
		return enhanceAuthError(getFirewall(ctx, client, hrobot.ServerID(serverID)))

	case "allow":
		if len(os.Args) < 5 {
			fmt.Printf("Usage: %s firewall allow <server-id> <ip>\n\n", os.Args[0])
			fmt.Println("Add an IP address to the firewall allow list.")
			fmt.Println("\nArguments:")
			fmt.Println("  <server-id>    The server number")
			fmt.Println("  <ip>           The IP address to allow")
			return nil
		}
		serverIDStr := os.Args[3]
		serverID, err := strconv.Atoi(serverIDStr)
		if err != nil {
			return fmt.Errorf("invalid server ID: %s", serverIDStr)
		}
		ipAddr := os.Args[4]
		return enhanceAuthError(allowIP(ctx, client, hrobot.ServerID(serverID), ipAddr))

	default:
		return fmt.Errorf("unknown firewall subcommand: %s\nSubcommands:\n  describe <server-id>      - Describe firewall configuration\n  allow <server-id> <ip>    - Add IP to firewall allow list", subcommand)
	}
}

// handleSSHKeyCommand handles all ssh-key-related subcommands.
func handleSSHKeyCommand(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: %s ssh-key <subcommand>\nSubcommands:\n  list                     - List all SSH keys\n  describe <name>          - Describe SSH key details\n  create <name> <file|->   - Create a new SSH key from file or stdin\n  rename <name> <new-name> - Rename an SSH key\n  delete <name>            - Delete an SSH key", os.Args[0])
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
		return fmt.Errorf("unknown ssh-key subcommand: %s\nSubcommands:\n  list                     - List all SSH keys\n  describe <name>          - Describe SSH key details\n  create <name> <file|->   - Create a new SSH key from file or stdin\n  rename <name> <new-name> - Rename an SSH key\n  delete <name>            - Delete an SSH key", subcommand)
	}
}

// handleRDNSCommand handles all rdns-related subcommands.
func handleRDNSCommand(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: %s rdns <subcommand>\nSubcommands:\n  list [server-ip]        - List all reverse DNS entries\n  describe <ip>           - Describe reverse DNS entry for an IP\n  set <ip> <ptr>          - Set reverse DNS entry for an IP\n  reset <ip>              - Reset reverse DNS entry to default", os.Args[0])
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
		if isHelpRequested() || len(os.Args) < 4 {
			fmt.Printf("Usage: %s rdns describe <ip>\n\n", os.Args[0])
			fmt.Println("Describe reverse DNS entry for an IP address.")
			fmt.Println("\nArguments:")
			fmt.Println("  <ip>    The IP address to query")
			return nil
		}
		ip := os.Args[3]
		return enhanceAuthError(getRDNS(ctx, client, ip))

	case "set":
		if isHelpRequested() || len(os.Args) < 5 {
			fmt.Printf("Usage: %s rdns set <ip> <ptr>\n\n", os.Args[0])
			fmt.Println("Set reverse DNS entry for an IP address.")
			fmt.Println("\nArguments:")
			fmt.Println("  <ip>     The IP address to configure")
			fmt.Println("  <ptr>    The PTR record value (hostname)")
			return nil
		}
		ip := os.Args[3]
		ptr := os.Args[4]
		return enhanceAuthError(setRDNS(ctx, client, ip, ptr))

	case "reset":
		if isHelpRequested() || len(os.Args) < 4 {
			fmt.Printf("Usage: %s rdns reset <ip>\n\n", os.Args[0])
			fmt.Println("Reset reverse DNS entry to default Hetzner value.")
			fmt.Println("\nArguments:")
			fmt.Println("  <ip>    The IP address to reset")
			return nil
		}
		ip := os.Args[3]
		return enhanceAuthError(deleteRDNS(ctx, client, ip))

	default:
		return fmt.Errorf("unknown rdns subcommand: %s\nSubcommands:\n  list [server-ip]        - List all reverse DNS entries\n  describe <ip>           - Describe reverse DNS entry for an IP\n  set <ip> <ptr>          - Set reverse DNS entry for an IP\n  reset <ip>              - Reset reverse DNS entry to default", subcommand)
	}
}

// handleFailoverCommand handles all failover-related subcommands.
func handleFailoverCommand(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: %s failover <subcommand>\nSubcommands:\n  list                       - List all failover IPs\n  describe <ip>              - Describe failover IP details\n  set <ip> <destination-ip>  - Route failover IP to destination server\n  delete <ip>                - Unroute failover IP", os.Args[0])
	}

	subcommand := os.Args[2]
	switch subcommand {
	case "list":
		return enhanceAuthError(listFailovers(ctx, client))

	case "describe":
		if isHelpRequested() || len(os.Args) < 4 {
			fmt.Printf("Usage: %s failover describe <ip>\n\n", os.Args[0])
			fmt.Println("Describe failover IP details.")
			fmt.Println("\nArguments:")
			fmt.Println("  <ip>    The failover IP address")
			return nil
		}
		ip := os.Args[3]
		return enhanceAuthError(getFailover(ctx, client, ip))

	case "set":
		if isHelpRequested() || len(os.Args) < 5 {
			fmt.Printf("Usage: %s failover set <ip> <destination-ip>\n\n", os.Args[0])
			fmt.Println("Route failover IP to a destination server.")
			fmt.Println("\nArguments:")
			fmt.Println("  <ip>              The failover IP address")
			fmt.Println("  <destination-ip>  The destination server IP")
			return nil
		}
		ip := os.Args[3]
		destIP := os.Args[4]
		return enhanceAuthError(setFailover(ctx, client, ip, destIP))

	case "delete":
		if isHelpRequested() || len(os.Args) < 4 {
			fmt.Printf("Usage: %s failover delete <ip>\n\n", os.Args[0])
			fmt.Println("Unroute a failover IP.")
			fmt.Println("\nArguments:")
			fmt.Println("  <ip>    The failover IP address to unroute")
			return nil
		}
		ip := os.Args[3]
		return enhanceAuthError(deleteFailover(ctx, client, ip))

	default:
		return fmt.Errorf("unknown failover subcommand: %s\nSubcommands:\n  list                       - List all failover IPs\n  describe <ip>              - Describe failover IP details\n  set <ip> <destination-ip>  - Route failover IP to destination server\n  delete <ip>                - Unroute failover IP", subcommand)
	}
}

// handleVSwitchCommand handles all vswitch-related subcommands.
func handleVSwitchCommand(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: %s vswitch <subcommand>\nSubcommands:\n  list                          - List all vSwitches\n  describe <id>                 - Describe vSwitch details\n  create <name> <vlan>          - Create a new vSwitch\n  update <id> <name> <vlan>     - Update vSwitch name and VLAN\n  delete <id> [--immediate]     - Cancel a vSwitch\n  add-server <id> <ip> [...]    - Add server(s) to vSwitch\n  remove-server <id> <ip> [...] - Remove server(s) from vSwitch", os.Args[0])
	}

	subcommand := os.Args[2]
	switch subcommand {
	case "list":
		return enhanceAuthError(listVSwitches(ctx, client))

	case "describe":
		if isHelpRequested() || len(os.Args) < 4 {
			fmt.Printf("Usage: %s vswitch describe <id>\n\n", os.Args[0])
			fmt.Println("Describe vSwitch details.")
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
			fmt.Println("  <name>    The name for the new vSwitch")
			fmt.Println("  <vlan>    The VLAN ID (4000-4091)")
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
			fmt.Println("Update vSwitch name and VLAN.")
			fmt.Println("\nArguments:")
			fmt.Println("  <id>      The vSwitch ID")
			fmt.Println("  <name>    The new name for the vSwitch")
			fmt.Println("  <vlan>    The new VLAN ID (4000-4091)")
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
				break
			}
		}
		return enhanceAuthError(deleteVSwitch(ctx, client, id, immediate))

	case "add-server":
		if isHelpRequested() || len(os.Args) < 5 {
			fmt.Printf("Usage: %s vswitch add-server <id> <ip> [...]\n\n", os.Args[0])
			fmt.Println("Add one or more servers to a vSwitch.")
			fmt.Println("\nArguments:")
			fmt.Println("  <id>     The vSwitch ID")
			fmt.Println("  <ip>     Server IP address(es) to add")
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
			fmt.Printf("Usage: %s vswitch remove-server <id> <ip> [...]\n\n", os.Args[0])
			fmt.Println("Remove one or more servers from a vSwitch.")
			fmt.Println("\nArguments:")
			fmt.Println("  <id>     The vSwitch ID")
			fmt.Println("  <ip>     Server IP address(es) to remove")
			return nil
		}
		id, err := strconv.Atoi(os.Args[3])
		if err != nil {
			return fmt.Errorf("invalid vSwitch ID: %s", os.Args[3])
		}
		servers := os.Args[4:]
		return enhanceAuthError(removeServersFromVSwitch(ctx, client, id, servers))

	default:
		return fmt.Errorf("unknown vswitch subcommand: %s\nSubcommands:\n  list                          - List all vSwitches\n  describe <id>                 - Describe vSwitch details\n  create <name> <vlan>          - Create a new vSwitch\n  update <id> <name> <vlan>     - Update vSwitch name and VLAN\n  delete <id> [--immediate]     - Cancel a vSwitch\n  add-server <id> <ip> [...]    - Add server(s) to vSwitch\n  remove-server <id> <ip> [...] - Remove server(s) from vSwitch", subcommand)
	}
}

// handleAuctionCommand handles all auction-related subcommands.
func handleAuctionCommand(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: %s auction <subcommand>\nSubcommands:\n  list                - List available auction servers\n  order <product-id>  - Order a server from auction", os.Args[0])
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

		for i := 4; i < len(os.Args); i++ {
			arg := os.Args[i]
			switch arg {
			case "--test":
				testMode = true
			case "--yes":
				skipConfirmation = true
			default:
				if sshKeyName == "" {
					sshKeyName = arg
				}
			}
		}

		if sshKeyName != "" {
			fingerprint, err := findKeyFingerprintByName(ctx, client, sshKeyName)
			if err != nil {
				return err
			}
			sshKeyFingerprints = []string{fingerprint}
		} else {
			keys, err := client.Key.List(ctx)
			if err != nil {
				return fmt.Errorf("failed to list SSH keys: %w", err)
			}
			if len(keys) == 0 {
				return fmt.Errorf("no SSH keys found in your account. Please create at least one SSH key first")
			}
			for _, key := range keys {
				sshKeyFingerprints = append(sshKeyFingerprints, key.Fingerprint)
			}
		}

		return enhanceOrderingAuthError(ctx, client, orderMarketServer(ctx, client, uint32(productID), sshKeyFingerprints, testMode, skipConfirmation))

	default:
		return fmt.Errorf("unknown auction subcommand: %s\nSubcommands:\n  list                - List available auction servers\n  order <product-id>  - Order a server from auction", subcommand)
	}
}

// handleProductCommand handles all product-related subcommands.
func handleProductCommand(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: %s product <subcommand>\nSubcommands:\n  list                - List available product servers\n  order <product-id>  - Order a product server", os.Args[0])
	}

	subcommand := os.Args[2]
	switch subcommand {
	case "list":
		return enhanceOrderingAuthError(ctx, client, listProducts(ctx, client))

	case "order":
		if isHelpRequested() || len(os.Args) < 4 {
			fmt.Printf("Usage: %s product order <product-id> [<ssh-key-name>] [--location=<dc>] [--yes] [--test]\n\n", os.Args[0])
			fmt.Println("Order a product server.")
			fmt.Println("\nArguments:")
			fmt.Println("  <product-id>      The product ID (e.g., EX44, AX41)")
			fmt.Println("  <ssh-key-name>    Optional: specific SSH key to use (default: all keys)")
			fmt.Println("\nFlags:")
			fmt.Println("  --location=<dc>   Data center location (e.g., FSN1, NBG1, HEL1)")
			fmt.Println("                    If not specified, automatically selects location with shortest availability")
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

		for i := 4; i < len(os.Args); i++ {
			arg := os.Args[i]
			if arg == "--test" {
				testMode = true
			} else if arg == "--yes" {
				skipConfirmation = true
			} else if len(arg) > 11 && arg[:11] == "--location=" {
				location = arg[11:]
			} else if sshKeyName == "" {
				sshKeyName = arg
			}
		}

		if sshKeyName != "" {
			fingerprint, err := findKeyFingerprintByName(ctx, client, sshKeyName)
			if err != nil {
				return err
			}
			sshKeyFingerprints = []string{fingerprint}
		} else {
			keys, err := client.Key.List(ctx)
			if err != nil {
				return fmt.Errorf("failed to list SSH keys: %w", err)
			}
			if len(keys) == 0 {
				return fmt.Errorf("no SSH keys found in your account. Please create at least one SSH key first")
			}
			for _, key := range keys {
				sshKeyFingerprints = append(sshKeyFingerprints, key.Fingerprint)
			}
		}

		return enhanceOrderingAuthError(ctx, client, orderProductServer(ctx, client, productID, location, sshKeyFingerprints, testMode, skipConfirmation))

	default:
		return fmt.Errorf("unknown product subcommand: %s\nSubcommands:\n  list                - List available product servers\n  order <product-id>  - Order a product server", subcommand)
	}
}
