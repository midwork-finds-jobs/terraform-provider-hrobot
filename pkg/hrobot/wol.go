// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hrobot

import (
	"context"
	"fmt"
)

// WOLService handles Wake-on-LAN related API operations.
type WOLService struct {
	client *Client
}

// NewWOLService creates a new WOL service.
func NewWOLService(client *Client) *WOLService {
	return &WOLService{client: client}
}

// WOLResponse represents the response from a Wake-on-LAN request.
type WOLResponse struct {
	ServerIP      string `json:"server_ip"`
	ServerIPv6Net string `json:"server_ipv6_net"`
	ServerNumber  int    `json:"server_number"`
}

// WOLWrapper wraps the WOL response.
type WOLWrapper struct {
	WOL WOLResponse `json:"wol"`
}

// Send sends a Wake-on-LAN packet to the server.
func (w *WOLService) Send(ctx context.Context, serverID ServerID) (*WOLResponse, error) {
	var wrapper WOLWrapper
	path := fmt.Sprintf("/wol/%s", serverID.String())

	err := w.client.Post(ctx, path, nil, &wrapper)
	if err != nil {
		return nil, err
	}

	return &wrapper.WOL, nil
}
