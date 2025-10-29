// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hrobot

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// VSwitchService provides access to vSwitch related functions in the Hetzner Robot API.
type VSwitchService struct {
	client *Client
}

// NewVSwitchService creates a new VSwitchService.
func NewVSwitchService(client *Client) *VSwitchService {
	return &VSwitchService{client: client}
}

// VSwitch represents a vSwitch in Hetzner Robot.
type VSwitch struct {
	ID           int             `json:"id"`
	Name         string          `json:"name"`
	VLAN         int             `json:"vlan"`
	Cancelled    bool            `json:"cancelled"`
	Servers      []VSwitchServer `json:"server,omitempty"`
	Subnets      []VSwitchSubnet `json:"subnet,omitempty"`
	CloudNetwork []CloudNetwork  `json:"cloud_network,omitempty"`
}

// VSwitchListItem represents a vSwitch in list responses (simplified).
type VSwitchListItem struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	VLAN      int    `json:"vlan"`
	Cancelled bool   `json:"cancelled"`
}

// VSwitchServer represents a server attached to a vSwitch.
type VSwitchServer struct {
	ServerIP      string `json:"server_ip"`
	ServerIPv6Net string `json:"server_ipv6_net"`
	ServerNumber  int    `json:"server_number"`
	Status        string `json:"status"` // "ready", "in process", "failed"
}

// VSwitchSubnet represents a subnet attached to a vSwitch.
type VSwitchSubnet struct {
	IP      string `json:"ip"`
	Mask    int    `json:"mask"`
	Gateway string `json:"gateway"`
}

// CloudNetwork represents a cloud network attached to a vSwitch.
type CloudNetwork struct {
	ID      int    `json:"id"`
	IP      string `json:"ip"`
	Mask    int    `json:"mask"`
	Gateway string `json:"gateway"`
}

// List retrieves all vSwitches.
//
// GET /vswitch
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-vswitch
func (v *VSwitchService) List(ctx context.Context) ([]VSwitchListItem, error) {
	path := "/vswitch"
	var result []VSwitchListItem
	if err := v.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Get retrieves a specific vSwitch by ID.
//
// GET /vswitch/{vswitch-id}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-vswitch-vswitch-id
func (v *VSwitchService) Get(ctx context.Context, id int) (*VSwitch, error) {
	path := fmt.Sprintf("/vswitch/%d", id)
	var result VSwitch
	if err := v.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Create creates a new vSwitch.
//
// POST /vswitch
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-vswitch
func (v *VSwitchService) Create(ctx context.Context, name string, vlan int) (*VSwitch, error) {
	path := "/vswitch"
	formData := url.Values{}
	formData.Set("name", name)
	formData.Set("vlan", strconv.Itoa(vlan))

	var result VSwitch
	if err := v.client.Post(ctx, path, formData, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Update changes the name or VLAN ID of a vSwitch.
//
// POST /vswitch/{vswitch-id}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-vswitch-vswitch-id
func (v *VSwitchService) Update(ctx context.Context, id int, name string, vlan int) error {
	path := fmt.Sprintf("/vswitch/%d", id)
	formData := url.Values{}
	formData.Set("name", name)
	formData.Set("vlan", strconv.Itoa(vlan))

	// API returns no output for successful updates
	return v.client.Post(ctx, path, formData, nil)
}

// Delete cancels a vSwitch.
//
// DELETE /vswitch/{vswitch-id}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#delete-vswitch-vswitch-id
func (v *VSwitchService) Delete(ctx context.Context, id int, cancellationDate string) error {
	path := fmt.Sprintf("/vswitch/%d", id)
	formData := url.Values{}
	formData.Set("cancellation_date", cancellationDate)

	// Use DeleteWithBody to send a DELETE request with form data
	return v.client.DeleteWithBody(ctx, path, formData, nil)
}

// AddServers adds one or more servers to a vSwitch.
//
// POST /vswitch/{vswitch-id}/server
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-vswitch-vswitch-id-server
func (v *VSwitchService) AddServers(ctx context.Context, id int, servers []string) error {
	path := fmt.Sprintf("/vswitch/%d/server", id)

	// Build form data with server[] array
	// The API expects server[]=value1&server[]=value2
	var formParts []string
	for _, server := range servers {
		formParts = append(formParts, fmt.Sprintf("server[]=%s", url.QueryEscape(server)))
	}
	formData := strings.Join(formParts, "&")

	// Use PostRaw to avoid url.Values encoding the brackets
	return v.client.PostRaw(ctx, path, formData, nil)
}

// RemoveServers removes one or more servers from a vSwitch.
//
// DELETE /vswitch/{vswitch-id}/server
//
// Note: The API uses DELETE with a body, which is unusual.
// We'll use PostRaw with the appropriate method.
//
// See: https://robot.hetzner.com/doc/webservice/en.html#delete-vswitch-vswitch-id-server
func (v *VSwitchService) RemoveServers(ctx context.Context, id int, servers []string) error {
	path := fmt.Sprintf("/vswitch/%d/server", id)

	// Build form data with server[] array
	var formParts []string
	for _, server := range servers {
		formParts = append(formParts, fmt.Sprintf("server[]=%s", url.QueryEscape(server)))
	}
	formData := strings.Join(formParts, "&")

	// For now, use PostRaw - we may need to enhance the client to support DELETE with body
	return v.client.PostRaw(ctx, path, formData, nil)
}

// WaitForVSwitchReady waits for a vSwitch to finish processing and become ready.
// This is useful after adding or removing servers, as the API returns VSWITCH_IN_PROCESS
// errors if operations are attempted while the vSwitch is processing.
func (v *VSwitchService) WaitForVSwitchReady(ctx context.Context, id int) error {
	return waitForCondition(ctx, func() (bool, error) {
		vswitch, err := v.Get(ctx, id)
		if err != nil {
			return false, err
		}
		// Check if all servers are in "ready" status (not "processing" or "failed")
		for _, server := range vswitch.Servers {
			if server.Status != "ready" {
				return false, nil
			}
		}
		return true, nil
	})
}
