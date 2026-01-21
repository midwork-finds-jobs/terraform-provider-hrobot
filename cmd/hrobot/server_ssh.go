// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

// sshToServer intelligently connects to a server via SSH, handling firewall configuration if needed.
func sshToServer(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, user string) error {
	// Step 1: Get server details to obtain IP address
	fmt.Printf("fetching server details for #%d...\n", serverID)
	server, err := client.Server.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get server details: %w", err)
	}

	serverIP := server.ServerIP.String()
	fmt.Printf("server IP: %s\n", serverIP)

	// Step 2: Check if SSH port is accessible
	fmt.Printf("checking SSH port accessibility...\n")
	accessible, err := isSSHPortOpen(serverIP, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to check SSH port: %w", err)
	}

	if accessible {
		fmt.Println("✓ SSH port is accessible")
	} else {
		fmt.Println("⊘ SSH port is not accessible, checking firewall configuration...")

		// Step 3: Get current public IP
		myIP, err := getMyIP()
		if err != nil {
			return fmt.Errorf("failed to detect your public IP: %w", err)
		}
		fmt.Printf("your public IP: %s\n", myIP)

		// Step 4: Check if current IP is in firewall rules
		fw, err := client.Firewall.Get(ctx, serverID)
		if err != nil {
			return fmt.Errorf("failed to get firewall configuration: %w", err)
		}

		myIPWithCIDR := myIP + "/32"
		hasAccess := checkIPInFirewallRules(fw.Rules.Input, myIPWithCIDR)

		if hasAccess {
			fmt.Printf("✓ your IP %s is already in firewall rules\n", myIP)
			fmt.Println("⚠ SSH port is not accessible despite firewall rules")
			fmt.Println("  possible reasons:")
			fmt.Println("  - firewall changes are still being applied (wait 30-40 seconds)")
			fmt.Println("  - server is down or SSH service is not running")
			fmt.Println("  - network issue between you and the server")
			fmt.Println("\nattempting SSH connection anyway...")
		} else {
			// Step 5: Add SSH rule for current IP
			fmt.Printf("adding SSH access rule for %s...\n", myIP)
			err = allowSSH(ctx, client, serverID, []string{}, true, "", false)
			if err != nil {
				return fmt.Errorf("failed to add SSH firewall rule: %w", err)
			}

			// Step 6: Wait for firewall to be ready
			fmt.Println("waiting for firewall changes to be applied...")
			err = client.Firewall.WaitForFirewallReady(ctx, serverID)
			if err != nil {
				return fmt.Errorf("failed while waiting for firewall: %w", err)
			}
			fmt.Println("✓ firewall is ready")

			// Give a bit more time for the rule to take effect
			fmt.Println("waiting 5 seconds for rules to propagate...")
			time.Sleep(5 * time.Second)
		}
	}

	// Step 7: Execute SSH
	fmt.Printf("\nconnecting to %s@%s via SSH...\n", user, serverIP)
	return execSSH(serverIP, user)
}

// isSSHPortOpen checks if the SSH port (22) is accessible on the given host.
func isSSHPortOpen(host string, timeout time.Duration) (bool, error) {
	address := net.JoinHostPort(host, "22")
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		// Port is not accessible
		return false, nil
	}
	defer conn.Close()
	return true, nil
}

// checkIPInFirewallRules checks if the given IP (with CIDR) is in the firewall rules.
func checkIPInFirewallRules(rules []hrobot.FirewallRule, ipWithCIDR string) bool {
	// Remove /32 suffix for comparison if present
	ip := strings.TrimSuffix(ipWithCIDR, "/32")

	for _, rule := range rules {
		// Skip rules that don't accept traffic
		if rule.Action != hrobot.ActionAccept {
			continue
		}

		// Skip rules with TCP flags (e.g., "ack" only allows established connections, not new SSH)
		if rule.TCPFlags != "" {
			continue
		}

		// Check if rule allows SSH port 22
		// A rule allows SSH if:
		// - Protocol is empty (any) or TCP, AND
		// - Port is empty (any), "22", or contains "22" in a range/list
		allowsSSH := false
		if rule.Protocol == "" || rule.Protocol == hrobot.ProtocolTCP {
			if rule.DestPort == "" || rule.DestPort == "22" || portContains(rule.DestPort, 22) {
				allowsSSH = true
			}
		}

		if !allowsSSH {
			continue
		}

		// Check if rule applies to our IP
		// Empty SourceIP means "any IP" - this rule allows SSH for everyone
		if rule.SourceIP == "" {
			return true
		}
		// Exact IP match
		if rule.SourceIP == ip || rule.SourceIP == ipWithCIDR {
			return true
		}
		// Check if it's in a CIDR range
		if strings.Contains(rule.SourceIP, "/") && ipInCIDR(ip, rule.SourceIP) {
			return true
		}
	}
	return false
}

// portContains checks if a port specification (can be single port, range like "1-100", or list like "22,80,443")
// contains the target port.
func portContains(portSpec string, targetPort int) bool {
	// Handle comma-separated list
	parts := strings.Split(portSpec, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Check for range (e.g., "32768-65535")
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				var start, end int
				if _, err := fmt.Sscanf(rangeParts[0], "%d", &start); err == nil {
					if _, err := fmt.Sscanf(rangeParts[1], "%d", &end); err == nil {
						if targetPort >= start && targetPort <= end {
							return true
						}
					}
				}
			}
		} else {
			// Single port
			var port int
			if _, err := fmt.Sscanf(part, "%d", &port); err == nil {
				if port == targetPort {
					return true
				}
			}
		}
	}
	return false
}

// ipInCIDR checks if an IP is within a CIDR range.
func ipInCIDR(ip, cidr string) bool {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false
	}
	return ipNet.Contains(ipAddr)
}

// execSSH executes the ssh command to connect to the server.
func execSSH(host, user string) error {
	// Find SSH binary
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh command not found in PATH: %w", err)
	}

	// Prepare SSH command arguments
	args := []string{"ssh", fmt.Sprintf("%s@%s", user, host)}

	// Execute SSH, replacing current process
	err = syscall.Exec(sshPath, args, os.Environ())
	if err != nil {
		return fmt.Errorf("failed to execute ssh: %w", err)
	}

	// This line will never be reached if exec succeeds
	return nil
}

// execMOSH executes the mosh command to connect to the server.
func execMOSH(host, user string) error {
	// Find mosh binary
	moshPath, err := exec.LookPath("mosh")
	if err != nil {
		return fmt.Errorf("mosh command not found in PATH: %w\n\nTo install mosh:\n  • macOS: brew install mosh\n  • Debian/Ubuntu: apt install mosh\n  • RHEL/Fedora: dnf install mosh", err)
	}

	// Prepare mosh command arguments
	args := []string{"mosh", fmt.Sprintf("%s@%s", user, host)}

	// Execute mosh, replacing current process
	err = syscall.Exec(moshPath, args, os.Environ())
	if err != nil {
		return fmt.Errorf("failed to execute mosh: %w", err)
	}

	// This line will never be reached if exec succeeds
	return nil
}

// connectToServer connects to a server via SSH or mosh, automatically managing firewall rules.
func connectToServer(ctx context.Context, client *hrobot.Client, serverID hrobot.ServerID, user string, useMosh bool, force bool) error {
	// Step 1: Get server details to obtain IP address
	fmt.Printf("fetching server details for #%d...\n", serverID)
	server, err := client.Server.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get server details: %w", err)
	}

	serverIP := server.ServerIP.String()
	fmt.Printf("server IP: %s\n", serverIP)

	// Step 2: Get current public IP
	myIP, err := getMyIP()
	if err != nil {
		return fmt.Errorf("failed to detect your public IP: %w", err)
	}
	fmt.Printf("your public IP: %s\n", myIP)

	// Step 3: Check firewall configuration
	fw, err := client.Firewall.Get(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get firewall configuration: %w", err)
	}

	myIPWithCIDR := myIP + "/32"
	hasAccess := checkIPInFirewallRules(fw.Rules.Input, myIPWithCIDR)

	// Step 4: Update firewall rules if needed
	// Only update if IP is not already in firewall
	// The --force flag is passed to allowSSH/allowMOSH to handle replacing old rules
	needsUpdate := !hasAccess

	if needsUpdate {
		if force {
			fmt.Printf("your IP changed to %s, updating firewall rule...\n", myIP)
		} else {
			fmt.Printf("your IP %s is not in firewall rules, adding access...\n", myIP)
		}

		// Use allowMOSH or allowSSH with --force flag to replace existing rules
		if useMosh {
			err = allowMOSH(ctx, client, serverID, []string{}, true, "", force)
		} else {
			err = allowSSH(ctx, client, serverID, []string{}, true, "", force)
		}

		if err != nil {
			return fmt.Errorf("failed to configure firewall rule: %w", err)
		}

		// Step 5: Wait for firewall to be ready
		fmt.Println("waiting for firewall changes to be applied...")
		err = client.Firewall.WaitForFirewallReady(ctx, serverID)
		if err != nil {
			return fmt.Errorf("failed while waiting for firewall: %w", err)
		}
		fmt.Println("✓ firewall is ready")

		// Give a bit more time for the rule to take effect
		fmt.Println("waiting 5 seconds for rules to propagate...")
		time.Sleep(5 * time.Second)
	} else {
		if force {
			fmt.Printf("✓ your IP %s is already in firewall rules (no update needed)\n", myIP)
		} else {
			fmt.Printf("✓ your IP %s is already allowed in firewall rules\n", myIP)
		}
	}

	// Step 6: Execute connection
	protocol := "SSH"
	if useMosh {
		protocol = "mosh"
	}
	fmt.Printf("\nconnecting to %s@%s via %s...\n", user, serverIP, protocol)

	if useMosh {
		return execMOSH(serverIP, user)
	}
	return execSSH(serverIP, user)
}
