package hrobot

import (
	"context"
	"fmt"
	"net"
	"net/url"
)

// IPService handles IP address related API operations.
type IPService struct {
	client *Client
}

// NewIPService creates a new IP service.
func NewIPService(client *Client) *IPService {
	return &IPService{client: client}
}

// List returns all IP addresses.
func (i *IPService) List(ctx context.Context) ([]IPAddress, error) {
	var ips []IPAddress
	err := i.client.Get(ctx, "/ip", &ips)
	if err != nil {
		return nil, err
	}
	return ips, nil
}

// Get returns details for a specific IP address.
func (i *IPService) Get(ctx context.Context, ip net.IP) (*IPAddress, error) {
	var ipAddr IPAddress
	path := fmt.Sprintf("/ip/%s", ip.String())
	err := i.client.Get(ctx, path, &ipAddr)
	if err != nil {
		return nil, err
	}
	return &ipAddr, nil
}

// ReverseDNS represents reverse DNS configuration.
type ReverseDNS struct {
	IP  net.IP `json:"ip"`
	PTR string `json:"ptr"`
}

// GetReverseDNS retrieves the reverse DNS entry for an IP.
func (i *IPService) GetReverseDNS(ctx context.Context, ip net.IP) (*ReverseDNS, error) {
	var rdns ReverseDNS
	path := fmt.Sprintf("/rdns/%s", ip.String())
	err := i.client.Get(ctx, path, &rdns)
	if err != nil {
		return nil, err
	}
	return &rdns, nil
}

// SetReverseDNS sets the reverse DNS entry for an IP.
func (i *IPService) SetReverseDNS(ctx context.Context, ip net.IP, ptr string) (*ReverseDNS, error) {
	var rdns ReverseDNS
	path := fmt.Sprintf("/rdns/%s", ip.String())

	data := url.Values{}
	data.Set("ptr", ptr)

	err := i.client.Post(ctx, path, data, &rdns)
	if err != nil {
		return nil, err
	}

	return &rdns, nil
}

// DeleteReverseDNS removes the reverse DNS entry for an IP.
func (i *IPService) DeleteReverseDNS(ctx context.Context, ip net.IP) error {
	path := fmt.Sprintf("/rdns/%s", ip.String())
	return i.client.Delete(ctx, path)
}

// TrafficData represents traffic statistics.
type TrafficData struct {
	Type string         `json:"type"`
	Data []TrafficEntry `json:"data"`
}

// TrafficEntry represents a single traffic data point.
type TrafficEntry struct {
	Timestamp string `json:"timestamp"`
	In        uint64 `json:"in"`
	Out       uint64 `json:"out"`
}

// GetTraffic retrieves traffic data for an IP.
func (i *IPService) GetTraffic(ctx context.Context, ip net.IP, trafficType string, from, to string) (*TrafficData, error) {
	var traffic TrafficData
	path := fmt.Sprintf("/traffic/%s", ip.String())

	// Add query parameters
	params := url.Values{}
	params.Set("type", trafficType)
	if from != "" {
		params.Set("from", from)
	}
	if to != "" {
		params.Set("to", to)
	}

	fullPath := fmt.Sprintf("%s?%s", path, params.Encode())

	err := i.client.Get(ctx, fullPath, &traffic)
	if err != nil {
		return nil, err
	}

	return &traffic, nil
}

// SetTrafficWarnings enables or disables traffic warnings.
func (i *IPService) SetTrafficWarnings(ctx context.Context, ip net.IP, enabled bool) error {
	path := fmt.Sprintf("/ip/%s", ip.String())

	data := url.Values{}
	if enabled {
		data.Set("traffic_warnings", "true")
	} else {
		data.Set("traffic_warnings", "false")
	}

	return i.client.Post(ctx, path, data, nil)
}

// CancelIP cancels an additional IP address.
func (i *IPService) CancelIP(ctx context.Context, ip net.IP, cancellationDate string) error {
	path := fmt.Sprintf("/ip/%s/cancellation", ip.String())

	data := url.Values{}
	data.Set("cancellation_date", cancellationDate)

	return i.client.Post(ctx, path, data, nil)
}

// WithdrawIPCancellation withdraws an IP cancellation.
func (i *IPService) WithdrawIPCancellation(ctx context.Context, ip net.IP) error {
	path := fmt.Sprintf("/ip/%s/cancellation", ip.String())
	return i.client.Delete(ctx, path)
}
