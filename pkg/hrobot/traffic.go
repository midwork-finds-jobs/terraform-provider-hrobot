// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hrobot

import (
	"context"
	"net/url"
)

// TrafficService provides access to traffic related functions in the Hetzner Robot API.
type TrafficService struct {
	client *Client
}

// NewTrafficService creates a new TrafficService.
func NewTrafficService(client *Client) *TrafficService {
	return &TrafficService{client: client}
}

// TrafficType represents the type of traffic data to retrieve.
type TrafficType string

const (
	TrafficTypeDay   TrafficType = "day"
	TrafficTypeMonth TrafficType = "month"
	TrafficTypeYear  TrafficType = "year"
)

// ServerTrafficData represents traffic statistics for a server.
type ServerTrafficData struct {
	Type string                             `json:"type"`
	From string                             `json:"from"`
	To   string                             `json:"to"`
	Data map[string]map[string]TrafficStats `json:"data"` // IP -> Date -> Traffic
}

// TrafficStats represents traffic statistics for a specific time period.
type TrafficStats struct {
	In  float64 `json:"in"`  // Incoming traffic in GB
	Out float64 `json:"out"` // Outgoing traffic in GB
	Sum float64 `json:"sum"` // Total traffic in GB
}

// ServerTrafficResponse represents the API response for traffic data.
type ServerTrafficResponse struct {
	Traffic ServerTrafficData `json:"traffic"`
}

// TrafficGetParams represents parameters for retrieving traffic data.
type TrafficGetParams struct {
	Type         TrafficType // Type of data (day, month, year)
	From         string      // Start date (YYYY-MM-DD)
	To           string      // End date (YYYY-MM-DD)
	IP           string      // Server IP address
	SingleValues bool        // Return single values per day
}

// Get retrieves traffic statistics for a server.
//
// POST /traffic
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-traffic
func (t *TrafficService) Get(ctx context.Context, params TrafficGetParams) (*ServerTrafficData, error) {
	path := "/traffic"

	// Build form data (API uses POST, not GET)
	formData := url.Values{}
	formData.Set("type", string(params.Type))
	formData.Set("from", params.From)
	formData.Set("to", params.To)
	if params.IP != "" {
		formData.Set("ip", params.IP)
	}
	if params.SingleValues {
		formData.Set("single_values", "true")
	}

	// Try parsing directly as ServerTrafficData (without wrapper)
	var result ServerTrafficData
	if err := t.client.Post(ctx, path, formData, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
