// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hrobot

import (
	"context"
	"fmt"
	"net/url"
)

// BootService handles boot configuration API operations.
type BootService struct {
	client *Client
}

// NewBootService creates a new boot service.
func NewBootService(client *Client) *BootService {
	return &BootService{client: client}
}

// BootConfig represents boot configuration response.
type BootConfig struct {
	Rescue  *RescueConfig  `json:"rescue,omitempty"`
	Linux   *LinuxConfig   `json:"linux,omitempty"`
	VNC     *VNCConfig     `json:"vnc,omitempty"`
	Windows *WindowsConfig `json:"windows,omitempty"`
	Plesk   *PleskConfig   `json:"plesk,omitempty"`
	CPanel  *CPanelConfig  `json:"cpanel,omitempty"`
}

// RescueConfig represents rescue system configuration.
type RescueConfig struct {
	ServerIP       string      `json:"server_ip"`
	ServerIPv6Net  string      `json:"server_ipv6_net"`
	ServerNumber   int         `json:"server_number"`
	Active         bool        `json:"active"`
	OS             interface{} `json:"os,omitempty"`   // string when active, []string when not
	Arch           interface{} `json:"arch,omitempty"` // int when active, []int when not
	AuthorizedKeys []string    `json:"authorized_key,omitempty"`
	HostKey        []string    `json:"host_key,omitempty"`
	Password       *string     `json:"password,omitempty"`
}

// LinuxConfig represents Linux installation configuration.
type LinuxConfig struct {
	ServerIP       string      `json:"server_ip"`
	ServerIPv6Net  string      `json:"server_ipv6_net"`
	ServerNumber   int         `json:"server_number"`
	Dist           interface{} `json:"dist"` // string when active, []string when not
	Arch           interface{} `json:"arch"` // int when active, []int when not
	Lang           interface{} `json:"lang"` // string when active, []string when not
	Active         bool        `json:"active"`
	Hostname       string      `json:"hostname,omitempty"`
	Password       *string     `json:"password,omitempty"`
	AuthorizedKeys []string    `json:"authorized_key,omitempty"`
	HostKey        []string    `json:"host_key,omitempty"`
}

// VNCConfig represents VNC configuration.
type VNCConfig struct {
	ServerIP      string      `json:"server_ip"`
	ServerIPv6Net string      `json:"server_ipv6_net"`
	ServerNumber  int         `json:"server_number"`
	Active        bool        `json:"active"`
	Dist          interface{} `json:"dist,omitempty"` // string when active, []string when not
	Arch          interface{} `json:"arch,omitempty"` // int when active, []int when not
	Lang          interface{} `json:"lang,omitempty"` // string when active, []string when not
	Password      *string     `json:"password,omitempty"`
}

// WindowsConfig represents Windows installation configuration.
type WindowsConfig struct {
	ServerIP      string      `json:"server_ip"`
	ServerIPv6Net string      `json:"server_ipv6_net"`
	ServerNumber  int         `json:"server_number"`
	Active        bool        `json:"active"`
	OS            interface{} `json:"os,omitempty"`   // string when active, []string when not
	Lang          interface{} `json:"lang,omitempty"` // string when active, []string when not
	Password      *string     `json:"password,omitempty"`
}

// PleskConfig represents Plesk installation configuration.
type PleskConfig struct {
	Active   bool   `json:"active"`
	Hostname string `json:"hostname,omitempty"`
}

// CPanelConfig represents cPanel installation configuration.
type CPanelConfig struct {
	Active   bool   `json:"active"`
	Hostname string `json:"hostname,omitempty"`
}

// Get retrieves the boot configuration for a server.
func (b *BootService) Get(ctx context.Context, serverID ServerID) (*BootConfig, error) {
	var config BootConfig
	path := fmt.Sprintf("/boot/%s", serverID.String())
	err := b.client.Get(ctx, path, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ActivateRescue activates the rescue system.
func (b *BootService) ActivateRescue(ctx context.Context, serverID ServerID, os string, arch int, authorizedKeys []string) (*RescueConfig, error) {
	path := fmt.Sprintf("/boot/%s/rescue", serverID.String())

	data := url.Values{}
	data.Set("os", os)
	data.Set("arch", fmt.Sprintf("%d", arch))

	for _, key := range authorizedKeys {
		data.Add("authorized_key[]", key)
	}

	var rescue RescueConfig
	err := b.client.Post(ctx, path, data, &rescue)
	if err != nil {
		return nil, err
	}

	return &rescue, nil
}

// DeactivateRescue deactivates the rescue system.
func (b *BootService) DeactivateRescue(ctx context.Context, serverID ServerID) error {
	path := fmt.Sprintf("/boot/%s/rescue", serverID.String())
	return b.client.Delete(ctx, path)
}

// GetLastRescue retrieves the last activated rescue system information.
func (b *BootService) GetLastRescue(ctx context.Context, serverID ServerID) (*RescueConfig, error) {
	var rescue RescueConfig
	path := fmt.Sprintf("/boot/%s/rescue/last", serverID.String())
	err := b.client.Get(ctx, path, &rescue)
	if err != nil {
		return nil, err
	}
	return &rescue, nil
}

// ActivateLinux activates Linux installation.
func (b *BootService) ActivateLinux(ctx context.Context, serverID ServerID, dist string, arch int, lang string, authorizedKeys []string) (*LinuxConfig, error) {
	path := fmt.Sprintf("/boot/%s/linux", serverID.String())

	data := url.Values{}
	data.Set("dist", dist)
	data.Set("arch", fmt.Sprintf("%d", arch))
	data.Set("lang", lang)

	for _, key := range authorizedKeys {
		data.Add("authorized_key[]", key)
	}

	var linux LinuxConfig
	err := b.client.Post(ctx, path, data, &linux)
	if err != nil {
		return nil, err
	}

	return &linux, nil
}

// DeactivateLinux deactivates Linux installation.
func (b *BootService) DeactivateLinux(ctx context.Context, serverID ServerID) error {
	path := fmt.Sprintf("/boot/%s/linux", serverID.String())
	return b.client.Delete(ctx, path)
}

// ActivateVNC activates VNC installation.
func (b *BootService) ActivateVNC(ctx context.Context, serverID ServerID, dist string, arch int, lang string) (*VNCConfig, error) {
	path := fmt.Sprintf("/boot/%s/vnc", serverID.String())

	data := url.Values{}
	data.Set("dist", dist)
	data.Set("arch", fmt.Sprintf("%d", arch))
	data.Set("lang", lang)

	var vnc VNCConfig
	err := b.client.Post(ctx, path, data, &vnc)
	if err != nil {
		return nil, err
	}

	return &vnc, nil
}

// DeactivateVNC deactivates VNC installation.
func (b *BootService) DeactivateVNC(ctx context.Context, serverID ServerID) error {
	path := fmt.Sprintf("/boot/%s/vnc", serverID.String())
	return b.client.Delete(ctx, path)
}
