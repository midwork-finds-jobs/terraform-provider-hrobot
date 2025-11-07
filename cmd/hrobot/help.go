// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

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
    server wake <id>                         Wake server using Wake-on-LAN
    server enable-rescue <id>                Enable rescue system
    server disable-rescue <id>               Disable rescue system
    server traffic <id>                      Show traffic statistics
    server images <id>                       Show boot/image configuration
    server install <id>                      Install operating system on server

  Firewall Commands:
    firewall allow-ssh <server-id>           Allow SSH access (see firewall --help)
    firewall allow-https <server-id>         Allow HTTPS access (see firewall --help)
    firewall allow-mosh <server-id>          Allow MOSH access (SSH + UDP 60000-61000)
    firewall block-http <server-id>          Block insecure HTTP
    firewall harden <server-id>              Apply security hardening
    firewall add-rule <server-id>            Add firewall rule
    firewall delete-rule <server-id>         Delete firewall rule
    firewall list-rules <server-id>          List firewall rules
    firewall template list                   List firewall templates
    firewall template apply <id> <tmpl-id>   Apply template to server
    firewall enable <server-id>              Enable firewall (use --filter-ipv6=true|false)
    firewall disable <server-id>             Disable firewall
    firewall status <server-id>              Show firewall status
    firewall reset <server-id>               Reset firewall

    For detailed firewall usage, run: hrobot firewall

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

Global Flags:
  --config string                            Config file path (default "~/.config/hrobot/cli.toml")
  --context string                           Currently active context

Environment Variables:
  HROBOT_USERNAME                            Your Hetzner Robot username (e.g., #ws+XXXXX)
  HROBOT_PASSWORD                            Your Hetzner Robot password

`)
}
