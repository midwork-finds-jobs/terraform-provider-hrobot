// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aquasecurity/table"
	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

func listKeys(ctx context.Context, client *hrobot.Client) error {
	keys, err := client.Key.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list SSH keys: %w", err)
	}

	fmt.Printf("Found %d SSH key(s):\n\n", len(keys))

	// Create table
	t := table.New(nil)
	t.SetHeaders("#", "Name", "Fingerprint", "Type", "Size", "Created")

	for i, key := range keys {
		t.AddRow(
			fmt.Sprintf("%d", i+1),
			key.Name,
			key.Fingerprint,
			key.Type,
			fmt.Sprintf("%d bits", key.Size),
			key.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}

	t.Render()
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
