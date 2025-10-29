package hrobot

import (
	"context"
	"fmt"
	"net/url"
)

// RDNSService provides access to reverse DNS related functions in the Hetzner Robot API.
type RDNSService struct {
	client *Client
}

// NewRDNSService creates a new RDNSService.
func NewRDNSService(client *Client) *RDNSService {
	return &RDNSService{client: client}
}

// RDNS represents a reverse DNS entry.
type RDNS struct {
	IP  string `json:"ip"`
	PTR string `json:"ptr"`
}

// RDNSListItem represents a reverse DNS entry in list responses.
type RDNSListItem struct {
	RDNS RDNS `json:"rdns"`
}

// List retrieves all reverse DNS entries.
// Optionally filter by server IP address.
//
// GET /rdns
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-rdns
func (r *RDNSService) List(ctx context.Context, serverIP string) ([]RDNS, error) {
	path := "/rdns"

	// Add optional server_ip filter
	if serverIP != "" {
		path += "?server_ip=" + url.QueryEscape(serverIP)
	}

	var result []RDNSListItem
	if err := r.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}

	// Extract RDNS entries from the wrapped response
	entries := make([]RDNS, 0, len(result))
	for _, item := range result {
		entries = append(entries, item.RDNS)
	}

	return entries, nil
}

// Get retrieves the reverse DNS entry for a specific IP address.
//
// GET /rdns/{ip}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-rdns-ip
func (r *RDNSService) Get(ctx context.Context, ip string) (*RDNS, error) {
	path := fmt.Sprintf("/rdns/%s", url.PathEscape(ip))

	var result RDNS
	if err := r.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Create creates a new reverse DNS entry for an IP address.
// Returns an error if an entry already exists.
//
// PUT /rdns/{ip}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#put-rdns-ip
func (r *RDNSService) Create(ctx context.Context, ip, ptr string) (*RDNS, error) {
	path := fmt.Sprintf("/rdns/%s", url.PathEscape(ip))

	formData := url.Values{}
	formData.Set("ptr", ptr)

	var result RDNS
	if err := r.client.Put(ctx, path, formData, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Update updates or creates a reverse DNS entry for an IP address.
// This method can be used to create a new entry or update an existing one.
//
// POST /rdns/{ip}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-rdns-ip
func (r *RDNSService) Update(ctx context.Context, ip, ptr string) (*RDNS, error) {
	path := fmt.Sprintf("/rdns/%s", url.PathEscape(ip))

	formData := url.Values{}
	formData.Set("ptr", ptr)

	var result RDNS
	if err := r.client.Post(ctx, path, formData, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete deletes the reverse DNS entry for an IP address.
//
// DELETE /rdns/{ip}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#delete-rdns-ip
func (r *RDNSService) Delete(ctx context.Context, ip string) error {
	path := fmt.Sprintf("/rdns/%s", url.PathEscape(ip))

	return r.client.Delete(ctx, path)
}
