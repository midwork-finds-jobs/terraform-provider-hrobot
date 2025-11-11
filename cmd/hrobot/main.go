// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bufio"
	"context"
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

// printGlobalFlags prints the global flags section for help output.
func printGlobalFlags() {
	fmt.Println("\nGlobal Flags:")
	fmt.Println("      --config string              Config file path (default \"~/.config/hrobot/cli.toml\")")
	fmt.Println("      --context string             Currently active context")
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

	// Handle context command (doesn't require credentials)
	if command == "context" {
		return handleContextCommand()
	}

	// Get credentials from context first, then fall back to environment
	username, password := getCredentialsFromContext()

	// Fall back to environment variables if no context is active
	if username == "" || password == "" {
		username = os.Getenv("HROBOT_USERNAME")
		password = os.Getenv("HROBOT_PASSWORD")
	}

	if username == "" || password == "" {
		return fmt.Errorf(`HROBOT_USERNAME and HROBOT_PASSWORD environment variables must be set, or use 'hrobot context' to manage credentials

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

Example usage with environment variables:
  export HROBOT_USERNAME='#ws+XXXXXXX'
  export HROBOT_PASSWORD='YYYYYY'

Or use context management:
  hrobot context create <name>  # Will prompt for credentials
  hrobot context use <name>`)
	}

	// Check for verbose flag
	verbose := parseFlagBool(os.Args, "--verbose")

	// Create client
	var clientOpts []hrobot.ClientOption
	if verbose {
		clientOpts = append(clientOpts, hrobot.WithDebug(true))
	}
	client := hrobot.New(username, password, clientOpts...)
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
			return nil
		}
		serverIDStr := os.Args[3]
		serverID, err := strconv.Atoi(serverIDStr)
		if err != nil {
			return fmt.Errorf("invalid server ID: %s", serverIDStr)
		}
		return enhanceAuthError(powerOffServer(ctx, client, hrobot.ServerID(serverID)))

	case "wake":
		if isHelpRequested() || len(os.Args) < 4 {
			fmt.Printf("Usage: %s server wake <server-id>\n\n", os.Args[0])
			fmt.Println("Send a Wake-on-LAN packet to wake the server.")
			fmt.Println("\nArguments:")
			fmt.Println("  <server-id>    The server number to wake")
			printGlobalFlags()
			return nil
		}
		serverIDStr := os.Args[3]
		serverID, err := strconv.Atoi(serverIDStr)
		if err != nil {
			return fmt.Errorf("invalid server ID: %s", serverIDStr)
		}
		return enhanceAuthError(wakeServer(ctx, client, hrobot.ServerID(serverID)))

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
			printGlobalFlags()
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
			printGlobalFlags()
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
			fmt.Println("  --days <n>     Number of days to show (default: 14)")
			fmt.Println("  --from <date>  Start date in YYYY-MM-DD format")
			fmt.Println("  --to <date>    End date in YYYY-MM-DD format")
			fmt.Println("\nNote: If --from and --to are specified, --days is ignored.")
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
		printFirewallHelp()
		return nil
	}

	subcommand := os.Args[2]
	switch subcommand {
	// Phase 1: Convenience commands
	case "allow-ssh":
		return handleAllowSSH(ctx, client)

	case "allow-https":
		return handleAllowHTTPS(ctx, client)

	case "allow-mosh":
		return handleAllowMOSH(ctx, client)

	case "block-http":
		return handleBlockHTTP(ctx, client)

	case "harden":
		return handleHarden(ctx, client)

	// Phase 2: Granular rule management
	case "add-rule":
		return handleAddRule(ctx, client)

	case "delete-rule":
		return handleDeleteRule(ctx, client)

	case "list-rules":
		return handleListRules(ctx, client)

	// Phase 3: Template management
	case "template":
		return handleTemplateCommand(ctx, client)

	// Phase 4: Status management
	case "enable":
		return handleEnableFirewall(ctx, client)

	case "disable":
		return handleDisableFirewall(ctx, client)

	case "status":
		return handleFirewallStatus(ctx, client)

	case "wait":
		return handleWaitFirewall(ctx, client)

	case "reset":
		return handleResetFirewall(ctx, client)

	default:
		printFirewallHelp()
		return fmt.Errorf("unknown firewall subcommand: %s", subcommand)
	}
}

func printFirewallHelp() {
	fmt.Printf("Usage: %s firewall <subcommand> [options]\n\n", os.Args[0])
	fmt.Println("Subcommands:")
	fmt.Println("\nConvenience Commands:")
	fmt.Println("  allow-ssh <server-id> --source-ips <ips> | --my-ip")
	fmt.Println("      allow SSH access from specific IPs")
	fmt.Println("  allow-https <server-id> --source-ips <ips>")
	fmt.Println("      allow HTTPS access from specific IPs (supports IPv6)")
	fmt.Println("  block-http <server-id>")
	fmt.Println("      block insecure HTTP (port 80)")
	fmt.Println("  harden <server-id> --block-http")
	fmt.Println("      apply common security hardening")
	fmt.Println("\nRule Management:")
	fmt.Println("  add-rule <server-id> --direction <in|out> --protocol <proto> [options]")
	fmt.Println("      add a firewall rule")
	fmt.Println("  delete-rule <server-id> --name <name> | --index <n> [--direction <in|out>]")
	fmt.Println("      delete a firewall rule")
	fmt.Println("  list-rules <server-id> [--direction <in|out>] [--output json]")
	fmt.Println("      list firewall rules")
	fmt.Println("\nTemplate Management:")
	fmt.Println("  template list [--output json]")
	fmt.Println("      list firewall templates")
	fmt.Println("  template describe <template-id> [--output json]")
	fmt.Println("      describe a template")
	fmt.Println("  template apply <server-id> <template-id>")
	fmt.Println("      apply template to server")
	fmt.Println("  template create --name <name> [--from-server <id> | --rules-file <file>]")
	fmt.Println("      create a new template")
	fmt.Println("  template delete <template-id> --confirm")
	fmt.Println("      delete a template")
	fmt.Println("\nStatus Management:")
	fmt.Println("  enable <server-id>")
	fmt.Println("      enable firewall")
	fmt.Println("  disable <server-id>")
	fmt.Println("      disable firewall")
	fmt.Println("  status <server-id>")
	fmt.Println("      show firewall status")
	fmt.Println("  wait <server-id>")
	fmt.Println("      wait for firewall to be ready")
	fmt.Println("  reset <server-id> --confirm")
	fmt.Println("      reset firewall (delete all rules)")
}

func parseServerID(s string) (hrobot.ServerID, error) {
	id, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid server ID: %s", s)
	}
	return hrobot.ServerID(id), nil
}

func parseFlagStringSlice(args []string, flag string) []string {
	var results []string
	for i, arg := range args {
		// Support both --flag=value and --flag value formats
		if strings.HasPrefix(arg, flag+"=") {
			value := strings.TrimPrefix(arg, flag+"=")
			// Remove surrounding quotes if present
			value = strings.Trim(value, "'\"")
			// Split by comma for comma-separated values
			values := strings.Split(value, ",")
			for _, v := range values {
				v = strings.TrimSpace(v)
				if v != "" {
					results = append(results, v)
				}
			}
		} else if arg == flag && i+1 < len(args) {
			// Split by comma for comma-separated values
			values := strings.Split(args[i+1], ",")
			for _, v := range values {
				v = strings.TrimSpace(v)
				if v != "" {
					results = append(results, v)
				}
			}
		}
	}
	return results
}

func parseFlagString(args []string, flag string) string {
	for i, arg := range args {
		// Support both --flag=value and --flag value formats
		if strings.HasPrefix(arg, flag+"=") {
			value := strings.TrimPrefix(arg, flag+"=")
			// Remove surrounding quotes if present
			return strings.Trim(value, "'\"")
		} else if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func parseFlagInt(args []string, flag string) int {
	s := parseFlagString(args, flag)
	if s == "" {
		return -1
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return val
}

func parseFlagBool(args []string, flag string) bool {
	for _, arg := range args {
		// Support both --flag and --flag=true/false formats
		if arg == flag {
			return true
		}
		if strings.HasPrefix(arg, flag+"=") {
			value := strings.TrimPrefix(arg, flag+"=")
			value = strings.Trim(value, "'\"")
			// Accept true, 1, yes as true values
			switch strings.ToLower(value) {
			case "true", "1", "yes":
				return true
			}
		}
	}
	return false
}

// Phase 1 command handlers.
func handleAllowSSH(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s firewall allow-ssh <server-id> --source-ips <ips> | --my-ip\n\n", os.Args[0])
		fmt.Println("allow SSH access from specific IPs")
		fmt.Println("\nArguments:")
		fmt.Println("  <server-id>    The server number")
		fmt.Println("\nFlags:")
		fmt.Println("  --source-ips   Comma-separated list of IPs/CIDRs")
		fmt.Println("  --my-ip        Use your current public IP")
		return nil
	}

	serverID, err := parseServerID(os.Args[3])
	if err != nil {
		return err
	}

	sourceIPs := parseFlagStringSlice(os.Args, "--source-ips")
	myIP := parseFlagBool(os.Args, "--my-ip")

	return enhanceAuthError(allowSSH(ctx, client, serverID, sourceIPs, myIP))
}

func handleAllowHTTPS(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s firewall allow-https <server-id> --source-ips <ips>\n\n", os.Args[0])
		fmt.Println("allow HTTPS access from specific IPs (supports IPv6)")
		fmt.Println("\nArguments:")
		fmt.Println("  <server-id>    The server number")
		fmt.Println("\nFlags:")
		fmt.Println("  --source-ips   Comma-separated list of IPs/CIDRs (IPv4 or IPv6)")
		return nil
	}

	serverID, err := parseServerID(os.Args[3])
	if err != nil {
		return err
	}

	sourceIPs := parseFlagStringSlice(os.Args, "--source-ips")
	if len(sourceIPs) == 0 {
		return fmt.Errorf("--source-ips is required")
	}

	return enhanceAuthError(allowHTTPS(ctx, client, serverID, sourceIPs))
}

func handleAllowMOSH(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s firewall allow-mosh <server-id> --source-ips <ips> | --my-ip\n\n", os.Args[0])
		fmt.Println("allow MOSH access from specific IPs")
		fmt.Println("\nArguments:")
		fmt.Println("  <server-id>    The server number")
		fmt.Println("\nFlags:")
		fmt.Println("  --source-ips   Comma-separated list of IPs/CIDRs")
		fmt.Println("  --my-ip        Use your current public IP")
		fmt.Println("\nCreates 3 rules per IP:")
		fmt.Println("  • SSH (TCP port 22)")
		fmt.Println("  • MOSH (UDP ports 60000-61000)")
		fmt.Println("  • TCP established (ACK, ports 32768-65535)")
		return nil
	}

	serverID, err := parseServerID(os.Args[3])
	if err != nil {
		return err
	}

	sourceIPs := parseFlagStringSlice(os.Args, "--source-ips")
	myIP := parseFlagBool(os.Args, "--my-ip")

	return enhanceAuthError(allowMOSH(ctx, client, serverID, sourceIPs, myIP))
}

func handleBlockHTTP(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s firewall block-http <server-id>\n\n", os.Args[0])
		fmt.Println("block insecure HTTP (port 80)")
		fmt.Println("\nArguments:")
		fmt.Println("  <server-id>    The server number")
		return nil
	}

	serverID, err := parseServerID(os.Args[3])
	if err != nil {
		return err
	}

	return enhanceAuthError(blockHTTP(ctx, client, serverID))
}

func handleHarden(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s firewall harden <server-id> --block-http\n\n", os.Args[0])
		fmt.Println("apply common security hardening")
		fmt.Println("\nArguments:")
		fmt.Println("  <server-id>    The server number")
		fmt.Println("\nFlags:")
		fmt.Println("  --block-http   Block insecure HTTP")
		return nil
	}

	serverID, err := parseServerID(os.Args[3])
	if err != nil {
		return err
	}

	blockHTTPFlag := parseFlagBool(os.Args, "--block-http")

	return enhanceAuthError(hardenFirewall(ctx, client, serverID, blockHTTPFlag))
}

// Phase 2 command handlers.
func handleAddRule(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s firewall add-rule <server-id> --direction <in|out> --protocol <proto> [options]\n\n", os.Args[0])
		fmt.Println("add a firewall rule")
		fmt.Println("\nArguments:")
		fmt.Println("  <server-id>       The server number")
		fmt.Println("\nRequired Flags:")
		fmt.Println("  --direction       in or out")
		fmt.Println("  --protocol        tcp, udp, icmp, esp, or gre")
		fmt.Println("\nOptional Flags:")
		fmt.Println("  --source-ips      Comma-separated source IPs (for direction=in)")
		fmt.Println("  --destination-ips Comma-separated dest IPs (for direction=out)")
		fmt.Println("  --port            Port or port range (required for tcp/udp)")
		fmt.Println("  --action          accept or discard (default: accept)")
		fmt.Println("  --name            Rule name")
		return nil
	}

	serverID, err := parseServerID(os.Args[3])
	if err != nil {
		return err
	}

	direction := parseFlagString(os.Args, "--direction")
	protocol := parseFlagString(os.Args, "--protocol")
	action := parseFlagString(os.Args, "--action")
	name := parseFlagString(os.Args, "--name")
	port := parseFlagString(os.Args, "--port")
	sourceIPs := parseFlagStringSlice(os.Args, "--source-ips")
	destIPs := parseFlagStringSlice(os.Args, "--destination-ips")

	if direction == "" {
		return fmt.Errorf("--direction is required")
	}
	if protocol == "" {
		return fmt.Errorf("--protocol is required")
	}
	if name == "" {
		name = fmt.Sprintf("custom %s rule", protocol)
	}

	return enhanceAuthError(addRule(ctx, client, serverID, direction, protocol, action, name, sourceIPs, destIPs, port))
}

func handleDeleteRule(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s firewall delete-rule <server-id> --name <name> | --index <n> [--direction <in|out>]\n\n", os.Args[0])
		fmt.Println("delete a firewall rule")
		fmt.Println("\nArguments:")
		fmt.Println("  <server-id>    The server number")
		fmt.Println("\nFlags:")
		fmt.Println("  --name         Rule name to delete")
		fmt.Println("  --index        Rule index to delete")
		fmt.Println("  --direction    in or out (default: in)")
		return nil
	}

	serverID, err := parseServerID(os.Args[3])
	if err != nil {
		return err
	}

	name := parseFlagString(os.Args, "--name")
	index := parseFlagInt(os.Args, "--index")
	direction := parseFlagString(os.Args, "--direction")

	return enhanceAuthError(deleteRule(ctx, client, serverID, name, index, direction))
}

func handleListRules(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s firewall list-rules <server-id> [--direction <in|out>] [--output json]\n\n", os.Args[0])
		fmt.Println("list firewall rules")
		fmt.Println("\nArguments:")
		fmt.Println("  <server-id>    The server number")
		fmt.Println("\nFlags:")
		fmt.Println("  --direction    Filter by direction (in or out)")
		fmt.Println("  --output       Output format (json)")
		return nil
	}

	serverID, err := parseServerID(os.Args[3])
	if err != nil {
		return err
	}

	direction := parseFlagString(os.Args, "--direction")
	outputFormat := parseFlagString(os.Args, "--output")

	return enhanceAuthError(listRules(ctx, client, serverID, direction, outputFormat))
}

// Phase 3 template command handlers.
func handleTemplateCommand(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s firewall template <subcommand>\n\n", os.Args[0])
		fmt.Println("Subcommands:")
		fmt.Println("  list [--output json]")
		fmt.Println("  describe <template-id> [--output json]")
		fmt.Println("  apply <server-id> <template-id>")
		fmt.Println("  create --name <name> [options]")
		fmt.Println("  delete <template-id> --confirm")
		return nil
	}

	subcommand := os.Args[3]
	switch subcommand {
	case "list":
		outputFormat := parseFlagString(os.Args, "--output")
		return enhanceAuthError(listTemplates(ctx, client, outputFormat))

	case "describe":
		if len(os.Args) < 5 {
			fmt.Printf("Usage: %s firewall template describe <template-id> [--output json]\n", os.Args[0])
			return nil
		}
		templateID, err := strconv.Atoi(os.Args[4])
		if err != nil {
			return fmt.Errorf("invalid template ID: %s", os.Args[4])
		}
		outputFormat := parseFlagString(os.Args, "--output")
		return enhanceAuthError(describeTemplate(ctx, client, templateID, outputFormat))

	case "apply":
		if len(os.Args) < 6 {
			fmt.Printf("Usage: %s firewall template apply <server-id> <template-id>\n", os.Args[0])
			return nil
		}
		serverID, err := parseServerID(os.Args[4])
		if err != nil {
			return err
		}
		templateID, err := strconv.Atoi(os.Args[5])
		if err != nil {
			return fmt.Errorf("invalid template ID: %s", os.Args[5])
		}
		return enhanceAuthError(applyTemplate(ctx, client, serverID, templateID))

	case "create":
		name := parseFlagString(os.Args, "--name")
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		fromServerID := parseFlagInt(os.Args, "--from-server")
		rulesFile := parseFlagString(os.Args, "--rules-file")
		whitelistHOS := parseFlagBool(os.Args, "--whitelist-hos")
		filterIPv6 := parseFlagBool(os.Args, "--filter-ipv6")

		return enhanceAuthError(createTemplate(ctx, client, name, hrobot.ServerID(fromServerID), rulesFile, whitelistHOS, filterIPv6))

	case "delete":
		if len(os.Args) < 5 {
			fmt.Printf("Usage: %s firewall template delete <template-id> --confirm\n", os.Args[0])
			return nil
		}
		templateID, err := strconv.Atoi(os.Args[4])
		if err != nil {
			return fmt.Errorf("invalid template ID: %s", os.Args[4])
		}
		confirm := parseFlagBool(os.Args, "--confirm")
		return enhanceAuthError(deleteTemplate(ctx, client, templateID, confirm))

	default:
		return fmt.Errorf("unknown template subcommand: %s", subcommand)
	}
}

// Phase 4 status management command handlers.
func handleEnableFirewall(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s firewall enable <server-id> [--filter-ipv6=true|false]\n\n", os.Args[0])
		fmt.Println("enable firewall")
		fmt.Println("\nArguments:")
		fmt.Println("  <server-id>    The server number")
		fmt.Println("\nFlags:")
		fmt.Println("  --filter-ipv6=true|false    Enable or disable IPv6 filtering (optional)")
		return nil
	}

	serverID, err := parseServerID(os.Args[3])
	if err != nil {
		return err
	}

	// Parse --filter-ipv6 flag
	var filterIPv6 *bool
	filterIPv6Str := parseFlagString(os.Args, "--filter-ipv6")
	if filterIPv6Str != "" {
		val, err := strconv.ParseBool(filterIPv6Str)
		if err != nil {
			return fmt.Errorf("invalid --filter-ipv6 value: %s (must be 'true' or 'false')", filterIPv6Str)
		}
		filterIPv6 = &val
	}

	return enhanceAuthError(enableFirewall(ctx, client, serverID, filterIPv6))
}

func handleDisableFirewall(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s firewall disable <server-id>\n\n", os.Args[0])
		fmt.Println("disable firewall")
		fmt.Println("\nArguments:")
		fmt.Println("  <server-id>    The server number")
		return nil
	}

	serverID, err := parseServerID(os.Args[3])
	if err != nil {
		return err
	}

	return enhanceAuthError(disableFirewall(ctx, client, serverID))
}

func handleFirewallStatus(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s firewall status <server-id>\n\n", os.Args[0])
		fmt.Println("show firewall status")
		fmt.Println("\nArguments:")
		fmt.Println("  <server-id>    The server number")
		return nil
	}

	serverID, err := parseServerID(os.Args[3])
	if err != nil {
		return err
	}

	return enhanceAuthError(getFirewallStatus(ctx, client, serverID))
}

func handleWaitFirewall(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s firewall wait <server-id>\n\n", os.Args[0])
		fmt.Println("wait for firewall to be ready")
		fmt.Println("\nArguments:")
		fmt.Println("  <server-id>    The server number")
		return nil
	}

	serverID, err := parseServerID(os.Args[3])
	if err != nil {
		return err
	}

	return enhanceAuthError(waitForFirewall(ctx, client, serverID))
}

func handleResetFirewall(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s firewall reset <server-id> --confirm\n\n", os.Args[0])
		fmt.Println("reset firewall (delete all rules)")
		fmt.Println("\nArguments:")
		fmt.Println("  <server-id>    The server number")
		fmt.Println("\nFlags:")
		fmt.Println("  --confirm      Required confirmation flag")
		return nil
	}

	serverID, err := parseServerID(os.Args[3])
	if err != nil {
		return err
	}

	confirm := parseFlagBool(os.Args, "--confirm")

	return enhanceAuthError(resetFirewall(ctx, client, serverID, confirm))
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
			printGlobalFlags()
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
		return fmt.Errorf("usage: %s auction <subcommand>\nSubcommands:\n  list                 - List available auction servers\n  describe <server-id> - Show details about a specific auction server\n  order <product-id>   - Order a server from auction", os.Args[0])
	}

	subcommand := os.Args[2]
	switch subcommand {
	case "list":
		if isHelpRequested() {
			fmt.Printf("Usage: %s auction list [--location=<location>] [--memory-min=<gb>] [--cpu=<type>] [--cpu-benchmark-min=<score>] [--disk-space-min=<gb>] [--price-max=<euros>] [--gpu]\n\n", os.Args[0])
			fmt.Println("List available auction servers with optional filters.")
			fmt.Println("\nFlags:")
			fmt.Println("  --location=<loc>            Filter by location (e.g., HEL, FSN, NBG)")
			fmt.Println("  --memory-min=<gb>           Minimum memory in GB (e.g., 128)")
			fmt.Println("  --cpu=<type>                Filter by CPU vendor (amd or intel)")
			fmt.Println("  --cpu-benchmark-min=<score> Minimum CPU benchmark score (e.g., 10000)")
			fmt.Println("  --disk-space-min=<gb>       Minimum disk space in GB (e.g., 7000)")
			fmt.Println("  --price-max=<euros>         Maximum monthly price in euros (e.g., 200)")
			fmt.Println("  --gpu                       Show only servers with GPU")
			printGlobalFlags()
			return nil
		}

		// Parse filter flags
		var location string
		var memoryMin float64
		var cpu string
		var cpuBenchmarkMin uint32
		var diskSpaceMin float64
		var priceMax float64
		var gpuOnly bool

		for i := 3; i < len(os.Args); i++ {
			arg := os.Args[i]
			if arg == "--gpu" {
				gpuOnly = true
			} else if len(arg) > 11 && arg[:11] == "--location=" {
				location = arg[11:]
			} else if len(arg) > 13 && arg[:13] == "--memory-min=" {
				val, err := strconv.ParseFloat(arg[13:], 64)
				if err != nil {
					return fmt.Errorf("invalid memory-min value: %s", arg[13:])
				}
				memoryMin = val
			} else if len(arg) > 6 && arg[:6] == "--cpu=" {
				cpu = arg[6:]
			} else if len(arg) > 20 && arg[:20] == "--cpu-benchmark-min=" {
				val, err := strconv.ParseUint(arg[20:], 10, 32)
				if err != nil {
					return fmt.Errorf("invalid cpu-benchmark-min value: %s", arg[20:])
				}
				cpuBenchmarkMin = uint32(val)
			} else if len(arg) > 17 && arg[:17] == "--disk-space-min=" {
				val, err := strconv.ParseFloat(arg[17:], 64)
				if err != nil {
					return fmt.Errorf("invalid disk-space-min value: %s", arg[17:])
				}
				diskSpaceMin = val
			} else if len(arg) > 12 && arg[:12] == "--price-max=" {
				val, err := strconv.ParseFloat(arg[12:], 64)
				if err != nil {
					return fmt.Errorf("invalid price-max value: %s", arg[12:])
				}
				priceMax = val
			}
		}

		return enhanceOrderingAuthError(ctx, client, listAuctionServers(ctx, client, location, memoryMin, cpu, cpuBenchmarkMin, diskSpaceMin, priceMax, gpuOnly))

	case "describe":
		if isHelpRequested() || len(os.Args) < 4 {
			fmt.Printf("Usage: %s auction describe <server-id>\n\n", os.Args[0])
			fmt.Println("Show detailed information about a specific auction server.")
			fmt.Println("\nArguments:")
			fmt.Println("  <server-id>   The auction server ID")
			printGlobalFlags()
			return nil
		}
		serverID, err := strconv.ParseUint(os.Args[3], 10, 32)
		if err != nil {
			return fmt.Errorf("invalid server ID: %s", os.Args[3])
		}
		return describeAuctionServer(ctx, client, uint32(serverID))

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
			printGlobalFlags()
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
		return fmt.Errorf("unknown auction subcommand: %s\nSubcommands:\n  list                 - List available auction servers\n  describe <server-id> - Show details about a specific auction server\n  order <product-id>   - Order a server from auction", subcommand)
	}
}

// handleProductCommand handles all product-related subcommands.
func handleProductCommand(ctx context.Context, client *hrobot.Client) error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: %s product <subcommand>\nSubcommands:\n  list                  - List available product servers\n  describe <product-id> - Show details about a specific product\n  order <product-id>    - Order a product server", os.Args[0])
	}

	subcommand := os.Args[2]
	switch subcommand {
	case "list":
		if isHelpRequested() {
			fmt.Printf("Usage: %s product list [--location=<location>] [--memory-min=<gb>] [--cpu=<type>] [--cpu-benchmark-min=<score>] [--disk-space-min=<gb>] [--price-max=<euros>] [--gpu]\n\n", os.Args[0])
			fmt.Println("List available product servers with optional filters.")
			fmt.Println("\nFlags:")
			fmt.Println("  --location=<loc>            Filter by location (e.g., HEL, FSN, NBG)")
			fmt.Println("  --memory-min=<gb>           Minimum memory in GB (e.g., 128)")
			fmt.Println("  --cpu=<type>                Filter by CPU vendor (amd or intel)")
			fmt.Println("  --cpu-benchmark-min=<score> Minimum CPU benchmark score (e.g., 10000)")
			fmt.Println("  --disk-space-min=<gb>       Minimum disk space in GB (e.g., 7000)")
			fmt.Println("  --price-max=<euros>         Maximum monthly price in euros (e.g., 200)")
			fmt.Println("  --gpu                       Show only servers with GPU")
			printGlobalFlags()
			return nil
		}

		// Parse filter flags
		var location string
		var memoryMin float64
		var cpu string
		var cpuBenchmarkMin uint32
		var diskSpaceMin float64
		var priceMax float64
		var gpuOnly bool

		for i := 3; i < len(os.Args); i++ {
			arg := os.Args[i]
			if arg == "--gpu" {
				gpuOnly = true
			} else if len(arg) > 11 && arg[:11] == "--location=" {
				location = arg[11:]
			} else if len(arg) > 13 && arg[:13] == "--memory-min=" {
				val, err := strconv.ParseFloat(arg[13:], 64)
				if err != nil {
					return fmt.Errorf("invalid memory-min value: %s", arg[13:])
				}
				memoryMin = val
			} else if len(arg) > 6 && arg[:6] == "--cpu=" {
				cpu = arg[6:]
			} else if len(arg) > 20 && arg[:20] == "--cpu-benchmark-min=" {
				val, err := strconv.ParseUint(arg[20:], 10, 32)
				if err != nil {
					return fmt.Errorf("invalid cpu-benchmark-min value: %s", arg[20:])
				}
				cpuBenchmarkMin = uint32(val)
			} else if len(arg) > 17 && arg[:17] == "--disk-space-min=" {
				val, err := strconv.ParseFloat(arg[17:], 64)
				if err != nil {
					return fmt.Errorf("invalid disk-space-min value: %s", arg[17:])
				}
				diskSpaceMin = val
			} else if len(arg) > 12 && arg[:12] == "--price-max=" {
				val, err := strconv.ParseFloat(arg[12:], 64)
				if err != nil {
					return fmt.Errorf("invalid price-max value: %s", arg[12:])
				}
				priceMax = val
			}
		}

		return enhanceOrderingAuthError(ctx, client, listProducts(ctx, client, location, memoryMin, cpu, cpuBenchmarkMin, diskSpaceMin, priceMax, gpuOnly))

	case "describe":
		if isHelpRequested() || len(os.Args) < 4 {
			fmt.Printf("Usage: %s product describe <product-id>\n\n", os.Args[0])
			fmt.Println("Show detailed information about a specific product.")
			fmt.Println("\nArguments:")
			fmt.Println("  <product-id>   The product ID (e.g., EX44, AX41-NVMe)")
			printGlobalFlags()
			return nil
		}
		productID := os.Args[3]
		return describeProduct(ctx, client, productID)

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
			printGlobalFlags()
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
		return fmt.Errorf("unknown product subcommand: %s\nSubcommands:\n  list                  - List available product servers\n  describe <product-id> - Show details about a specific product\n  order <product-id>    - Order a product server", subcommand)
	}
}

// handleContextCommand handles all context-related subcommands.
func handleContextCommand() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: %s context <subcommand>\nSubcommands:\n  list           - List all contexts\n  create <name>  - Create a new context\n  use <name>     - Switch to a context\n  active         - Show active context\n  delete <name>  - Delete a context", os.Args[0])
	}

	subcommand := os.Args[2]
	switch subcommand {
	case "list":
		return listContexts()

	case "create":
		if len(os.Args) < 4 {
			return fmt.Errorf("usage: %s context create <name> [--username <username>] [--password <password>]", os.Args[0])
		}
		name := os.Args[3]

		// Use centralized flag parsing that handles both --flag=value and --flag value
		username := parseFlagString(os.Args, "--username")
		password := parseFlagString(os.Args, "--password")

		// Prompt for missing credentials
		reader := bufio.NewReader(os.Stdin)

		if username == "" {
			fmt.Print("Enter username (e.g., #ws+XXXXXXX): ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read username: %w", err)
			}
			username = strings.TrimSpace(input)
			if username == "" {
				return fmt.Errorf("username cannot be empty")
			}
		}

		if password == "" {
			fmt.Print("Enter password: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			password = strings.TrimSpace(input)
			if password == "" {
				return fmt.Errorf("password cannot be empty")
			}
		}

		return createContext(name, username, password)

	case "use":
		if len(os.Args) < 4 {
			return fmt.Errorf("usage: %s context use <name>", os.Args[0])
		}
		return useContext(os.Args[3])

	case "active":
		return showActiveContext()

	case "delete":
		if len(os.Args) < 4 {
			return fmt.Errorf("usage: %s context delete <name>", os.Args[0])
		}
		return deleteContextCmd(os.Args[3])

	default:
		return fmt.Errorf("unknown context subcommand: %s\nSubcommands:\n  list           - List all contexts\n  create <name>  - Create a new context\n  use <name>     - Switch to a context\n  active         - Show active context\n  delete <name>  - Delete a context", subcommand)
	}
}
