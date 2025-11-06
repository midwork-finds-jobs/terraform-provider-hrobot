// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the CLI configuration.
type Config struct {
	ActiveContext string    `toml:"active_context"`
	Contexts      []Context `toml:"contexts"`
}

// Context represents a named configuration with credentials.
type Context struct {
	Name     string `toml:"name"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

// getConfigPath returns the path to the config file.
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "hrobot")
	configPath := filepath.Join(configDir, "cli.toml")

	return configPath, nil
}

// ensureConfigDir creates the config directory if it doesn't exist.
func ensureConfigDir() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	return os.MkdirAll(configDir, 0700)
}

// loadConfig loads the configuration from the config file.
func loadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// If config doesn't exist, return empty config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{
			Contexts: []Context{},
		}, nil
	}

	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// saveConfig saves the configuration to the config file.
func saveConfig(config *Config) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}

	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Set file permissions to 0600 (read/write for owner only)
	if err := os.Chmod(configPath, 0600); err != nil {
		return fmt.Errorf("failed to set config file permissions: %w", err)
	}

	return nil
}

// getContext returns a context by name.
func (c *Config) getContext(name string) *Context {
	for i := range c.Contexts {
		if c.Contexts[i].Name == name {
			return &c.Contexts[i]
		}
	}
	return nil
}

// addContext adds a new context or updates an existing one.
func (c *Config) addContext(ctx Context) {
	for i := range c.Contexts {
		if c.Contexts[i].Name == ctx.Name {
			c.Contexts[i] = ctx
			return
		}
	}
	c.Contexts = append(c.Contexts, ctx)
}

// deleteContext removes a context by name.
func (c *Config) deleteContext(name string) bool {
	for i := range c.Contexts {
		if c.Contexts[i].Name == name {
			c.Contexts = append(c.Contexts[:i], c.Contexts[i+1:]...)
			return true
		}
	}
	return false
}

// getActiveContext returns the active context.
func (c *Config) getActiveContext() *Context {
	if c.ActiveContext == "" {
		return nil
	}
	return c.getContext(c.ActiveContext)
}

// listContexts lists all available contexts.
func listContexts() error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	if len(config.Contexts) == 0 {
		fmt.Println("No contexts found")
		return nil
	}

	for _, ctx := range config.Contexts {
		active := ""
		if ctx.Name == config.ActiveContext {
			active = " (active)"
		}
		fmt.Printf("%s%s\n", ctx.Name, active)
	}

	return nil
}

// createContext creates a new context.
func createContext(name, username, password string) error {
	if name == "" {
		return fmt.Errorf("context name cannot be empty")
	}
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	config, err := loadConfig()
	if err != nil {
		return err
	}

	// Check if context already exists
	if existing := config.getContext(name); existing != nil {
		return fmt.Errorf("context '%s' already exists", name)
	}

	config.addContext(Context{
		Name:     name,
		Username: username,
		Password: password,
	})

	// Set as active if it's the first context
	if len(config.Contexts) == 1 {
		config.ActiveContext = name
	}

	if err := saveConfig(config); err != nil {
		return err
	}

	fmt.Printf("Context '%s' created\n", name)
	if config.ActiveContext == name {
		fmt.Printf("Context '%s' is now active\n", name)
	}

	return nil
}

// deleteContext deletes a context.
func deleteContextCmd(name string) error {
	if name == "" {
		return fmt.Errorf("context name cannot be empty")
	}

	config, err := loadConfig()
	if err != nil {
		return err
	}

	if !config.deleteContext(name) {
		return fmt.Errorf("context '%s' not found", name)
	}

	// Clear active context if it was deleted
	if config.ActiveContext == name {
		config.ActiveContext = ""
	}

	if err := saveConfig(config); err != nil {
		return err
	}

	fmt.Printf("Context '%s' deleted\n", name)

	return nil
}

// useContext sets the active context.
func useContext(name string) error {
	if name == "" {
		return fmt.Errorf("context name cannot be empty")
	}

	config, err := loadConfig()
	if err != nil {
		return err
	}

	if config.getContext(name) == nil {
		return fmt.Errorf("context '%s' not found", name)
	}

	config.ActiveContext = name

	if err := saveConfig(config); err != nil {
		return err
	}

	fmt.Printf("Active context: %s\n", name)

	return nil
}

// showActiveContext shows the currently active context.
func showActiveContext() error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	if config.ActiveContext == "" {
		fmt.Println("No active context")
		return nil
	}

	fmt.Println(config.ActiveContext)

	return nil
}

// getCredentialsFromContext returns credentials from the active context.
// Returns empty strings if no active context is found.
func getCredentialsFromContext() (username, password string) {
	config, err := loadConfig()
	if err != nil {
		return "", ""
	}

	ctx := config.getActiveContext()
	if ctx == nil {
		return "", ""
	}

	return ctx.Username, ctx.Password
}
