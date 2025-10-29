// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hrobot

import (
	"context"
	"fmt"
	"net/url"
)

// FailoverService provides access to failover IP related functions in the Hetzner Robot API.
type FailoverService struct {
	client *Client
}

// NewFailoverService creates a new FailoverService.
func NewFailoverService(client *Client) *FailoverService {
	return &FailoverService{client: client}
}

// Failover represents a failover IP configuration.
type Failover struct {
	IP             string  `json:"ip"`
	Netmask        string  `json:"netmask"`
	ServerIP       string  `json:"server_ip"`
	ServerIPv6Net  string  `json:"server_ipv6_net"`
	ServerNumber   int     `json:"server_number"`
	ActiveServerIP *string `json:"active_server_ip"` // nil when unrouted
}

// FailoverListItem represents a failover entry in list responses.
type FailoverListItem struct {
	Failover Failover `json:"failover"`
}

// FailoverResponse wraps a single failover response.
type FailoverResponse struct {
	Failover Failover `json:"failover"`
}

// List retrieves all failover IPs for servers.
//
// GET /failover
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-failover
func (f *FailoverService) List(ctx context.Context) ([]Failover, error) {
	path := "/failover"

	var result []Failover
	if err := f.client.GetWrappedList(ctx, path, "failover", &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Get retrieves the details for a specific failover IP.
//
// GET /failover/{failover-ip}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-failover-failover-ip
func (f *FailoverService) Get(ctx context.Context, ip string) (*Failover, error) {
	path := fmt.Sprintf("/failover/%s", url.PathEscape(ip))

	var result Failover
	if err := f.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Update switches routing of failover IP address to another server.
//
// POST /failover/{failover-ip}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-failover-failover-ip
func (f *FailoverService) Update(ctx context.Context, ip, activeServerIP string) (*Failover, error) {
	path := fmt.Sprintf("/failover/%s", url.PathEscape(ip))

	formData := url.Values{}
	formData.Set("active_server_ip", activeServerIP)

	var result Failover
	if err := f.client.Post(ctx, path, formData, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete deletes the routing of a failover IP (sets active_server_ip to null).
//
// DELETE /failover/{failover-ip}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#delete-failover-failover-ip
func (f *FailoverService) Delete(ctx context.Context, ip string) error {
	path := fmt.Sprintf("/failover/%s", url.PathEscape(ip))

	return f.client.Delete(ctx, path)
}
