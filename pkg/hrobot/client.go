// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hrobot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://robot-ws.your-server.de"
	DefaultTimeout = 30 * time.Second
	UserAgent      = "hrobot-go/1.0.0"
)

// Client is the main API client for Hetzner Robot.
type Client struct {
	baseURL    string
	httpClient *http.Client
	username   string
	password   string
	userAgent  string

	// API Services
	Server   *ServerService
	Firewall *FirewallService
	Reset    *ResetService
	Boot     *BootService
	IP       *IPService
	Key      *KeyService
	Auction  *AuctionService
	Ordering *OrderingService
	VSwitch  *VSwitchService
	RDNS     *RDNSService
	Failover *FailoverService
	Traffic  *TrafficService
}

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL.
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = strings.TrimSuffix(url, "/")
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithUserAgent sets a custom user agent.
func WithUserAgent(ua string) ClientOption {
	return func(c *Client) {
		c.userAgent = ua
	}
}

// NewClient creates a new Hetzner Robot API client.
func NewClient(username, password string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL:   DefaultBaseURL,
		username:  username,
		password:  password,
		userAgent: UserAgent,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	// Initialize services
	c.Server = NewServerService(c)
	c.Firewall = NewFirewallService(c)
	c.Reset = NewResetService(c)
	c.Boot = NewBootService(c)
	c.IP = NewIPService(c)
	c.Key = NewKeyService(c)
	c.Auction = NewAuctionService(c)
	c.Ordering = NewOrderingService(c)
	c.VSwitch = NewVSwitchService(c)
	c.RDNS = NewRDNSService(c)
	c.Failover = NewFailoverService(c)
	c.Traffic = NewTrafficService(c)

	return c
}

// New creates a new Hetzner Robot API client (alias for NewClient).
func New(username, password string, opts ...ClientOption) *Client {
	return NewClient(username, password, opts...)
}

// Response wrapper types to handle Hetzner's response structure.
type responseWrapper struct {
	Data json.RawMessage `json:"data,omitempty"`
	// Hetzner wraps responses in various keys
	Server                  json.RawMessage `json:"server,omitempty"`
	Servers                 json.RawMessage `json:"servers,omitempty"`
	Firewall                json.RawMessage `json:"firewall,omitempty"`
	IP                      json.RawMessage `json:"ip,omitempty"`
	Reset                   json.RawMessage `json:"reset,omitempty"`
	Boot                    json.RawMessage `json:"boot,omitempty"`
	Rescue                  json.RawMessage `json:"rescue,omitempty"`
	Key                     json.RawMessage `json:"key,omitempty"`
	VSwitch                 json.RawMessage `json:"vswitch,omitempty"`
	RDNS                    json.RawMessage `json:"rdns,omitempty"`
	Failover                json.RawMessage `json:"failover,omitempty"`
	Traffic                 json.RawMessage `json:"traffic,omitempty"`
	ServerMarketProduct     json.RawMessage `json:"server_market_product,omitempty"`
	ServerMarketTransaction json.RawMessage `json:"server_market_transaction,omitempty"`
	ServerAddonTransaction  json.RawMessage `json:"server_addon_transaction,omitempty"`
	ServerAddonProduct      json.RawMessage `json:"server_addon_product,omitempty"`
	Transaction             json.RawMessage `json:"transaction,omitempty"`
}

// unwrapArrayResponse handles arrays where each item is wrapped in an object
// e.g. [{"server": {...}}, {"server": {...}}].
func unwrapArrayResponse(data []byte, wrapperKey string) (json.RawMessage, error) {
	var wrappers []map[string]json.RawMessage
	if err := json.Unmarshal(data, &wrappers); err != nil {
		return nil, err
	}

	result := make([]json.RawMessage, 0, len(wrappers))
	for _, wrapper := range wrappers {
		if item, ok := wrapper[wrapperKey]; ok {
			result = append(result, item)
		}
	}

	return json.Marshal(result)
}

// unwrapResponse extracts the actual data from Hetzner's wrapped response.
func unwrapResponse(data []byte) (json.RawMessage, error) {
	// First, check if the response is an array
	if len(data) > 0 && data[0] == '[' {
		// Try to unwrap array elements that might be wrapped in objects
		// e.g., [{"vswitch": {...}}] -> {...}
		var arrayOfWrappers []responseWrapper
		if err := json.Unmarshal(data, &arrayOfWrappers); err == nil && len(arrayOfWrappers) > 0 {
			// Check if the first element has any wrapper keys
			wrapper := arrayOfWrappers[0]
			if len(wrapper.VSwitch) > 0 {
				return wrapper.VSwitch, nil
			}
			if len(wrapper.Server) > 0 {
				return wrapper.Server, nil
			}
			if len(wrapper.Firewall) > 0 {
				return wrapper.Firewall, nil
			}
			if len(wrapper.Key) > 0 {
				return wrapper.Key, nil
			}
			if len(wrapper.Failover) > 0 {
				return wrapper.Failover, nil
			}
			// Add other wrapper keys as needed
		}
		// If no wrapper keys found or unmarshal failed, return the array as-is
		return data, nil
	}

	// First, check if this is actually a wrapped response by looking for known API wrapper patterns
	// The Hetzner API typically wraps responses in keys like {"server": {...}}, {"firewall": {...}}, etc.
	// If the JSON has other keys at the top level (like "id", "name"), it's not wrapped.
	var topLevelKeys map[string]json.RawMessage
	if err := json.Unmarshal(data, &topLevelKeys); err == nil {
		// Check if this looks like an actual resource object (has "id" key) rather than a wrapper
		if _, hasID := topLevelKeys["id"]; hasID {
			// This is a resource object, not a wrapped response
			return data, nil
		}
	}

	var wrapper responseWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		// If we can't unmarshal as a wrapper, return the original data
		return data, err
	}

	// Try each possible wrapper key
	if len(wrapper.Data) > 0 {
		return wrapper.Data, nil
	}
	if len(wrapper.Server) > 0 {
		return wrapper.Server, nil
	}
	if len(wrapper.Servers) > 0 {
		return wrapper.Servers, nil
	}
	if len(wrapper.Firewall) > 0 {
		return wrapper.Firewall, nil
	}
	if len(wrapper.IP) > 0 {
		return wrapper.IP, nil
	}
	if len(wrapper.Reset) > 0 {
		return wrapper.Reset, nil
	}
	if len(wrapper.Boot) > 0 {
		return wrapper.Boot, nil
	}
	if len(wrapper.Rescue) > 0 {
		return wrapper.Rescue, nil
	}
	if len(wrapper.Key) > 0 {
		return wrapper.Key, nil
	}
	if len(wrapper.VSwitch) > 0 {
		return wrapper.VSwitch, nil
	}
	if len(wrapper.RDNS) > 0 {
		return wrapper.RDNS, nil
	}
	if len(wrapper.Failover) > 0 {
		return wrapper.Failover, nil
	}
	if len(wrapper.Traffic) > 0 {
		return wrapper.Traffic, nil
	}
	if len(wrapper.ServerMarketProduct) > 0 {
		return wrapper.ServerMarketProduct, nil
	}
	if len(wrapper.ServerMarketTransaction) > 0 {
		return wrapper.ServerMarketTransaction, nil
	}
	if len(wrapper.ServerAddonTransaction) > 0 {
		return wrapper.ServerAddonTransaction, nil
	}
	if len(wrapper.ServerAddonProduct) > 0 {
		return wrapper.ServerAddonProduct, nil
	}
	if len(wrapper.Transaction) > 0 {
		return wrapper.Transaction, nil
	}

	// No wrapper found, return original data
	return data, nil
}

// doRequest executes an HTTP request with authentication.
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, NewNetworkError("failed to create request", err)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("User-Agent", c.userAgent)

	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, NewNetworkError("request failed", err)
	}

	return resp, nil
}

// handleResponse processes the HTTP response and handles errors.
func (c *Client) handleResponse(resp *http.Response, v interface{}) error {
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewNetworkError("failed to read response body", err)
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		var apiErr APIErrorResponse
		if err := json.Unmarshal(body, &apiErr); err != nil {
			return NewAPIError(ErrUnknown, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
		}
		return NewAPIError(apiErr.Error.Code, apiErr.Error.Message)
	}

	// Handle empty responses (204 No Content)
	if resp.StatusCode == http.StatusNoContent || len(body) == 0 {
		return nil
	}

	// Unwrap the response
	unwrapped, err := unwrapResponse(body)
	if err != nil {
		return NewParseError("failed to unwrap response", err)
	}

	// Unmarshal into the target type
	if v != nil {
		if err := json.Unmarshal(unwrapped, v); err != nil {
			return NewParseError("failed to unmarshal response", err)
		}
	}

	return nil
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string, v interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, v)
}

// Post performs a POST request with form data.
func (c *Client) Post(ctx context.Context, path string, data url.Values, v interface{}) error {
	var body io.Reader
	if data != nil {
		body = strings.NewReader(data.Encode())
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, v)
}

// PostRaw performs a POST request with pre-encoded form data string.
// This is useful when the API expects literal brackets in form keys (not URL-encoded).
func (c *Client) PostRaw(ctx context.Context, path string, data string, v interface{}) error {
	var body io.Reader
	if data != "" {
		body = strings.NewReader(data)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, v)
}

// Put performs a PUT request with form data.
func (c *Client) Put(ctx context.Context, path string, data url.Values, v interface{}) error {
	var body io.Reader
	if data != nil {
		body = strings.NewReader(data.Encode())
	}

	resp, err := c.doRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, v)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

// DeleteWithBody performs a DELETE request with form data.
// This is used for APIs that require a DELETE request with a body, like vSwitch cancellation.
func (c *Client) DeleteWithBody(ctx context.Context, path string, data url.Values, v interface{}) error {
	var body io.Reader
	if data != nil {
		body = strings.NewReader(data.Encode())
	}

	resp, err := c.doRequest(ctx, http.MethodDelete, path, body)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, v)
}

// GetWrappedList performs a GET request for array responses where each item is wrapped
// e.g. [{"server": {...}}, {"server": {...}}].
func (c *Client) GetWrappedList(ctx context.Context, path string, wrapperKey string, v interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewNetworkError("failed to read response body", err)
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		var apiErr APIErrorResponse
		if err := json.Unmarshal(body, &apiErr); err != nil {
			return NewAPIError(ErrUnknown, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
		}
		return NewAPIError(apiErr.Error.Code, apiErr.Error.Message)
	}

	// Unwrap the array
	unwrapped, err := unwrapArrayResponse(body, wrapperKey)
	if err != nil {
		return NewParseError("failed to unwrap array response", err)
	}

	// Unmarshal into the target type
	if v != nil {
		if err := json.Unmarshal(unwrapped, v); err != nil {
			return NewParseError("failed to unmarshal response", err)
		}
	}

	return nil
}

// PostJSON performs a POST request with JSON body.
func (c *Client) PostJSON(ctx context.Context, path string, body interface{}, v interface{}) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return NewParseError("failed to marshal request body", err)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return NewNetworkError("failed to create request", err)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return NewNetworkError("request failed", err)
	}

	return c.handleResponse(resp, v)
}
