package hrobot

import (
	"context"
	"fmt"
	"net/url"
)

// ResetService handles reset-related API operations.
type ResetService struct {
	client *Client
}

// NewResetService creates a new reset service.
func NewResetService(client *Client) *ResetService {
	return &ResetService{client: client}
}

// Get retrieves the reset configuration for a server.
func (r *ResetService) Get(ctx context.Context, serverID ServerID) (*Reset, error) {
	var reset Reset
	path := fmt.Sprintf("/reset/%s", serverID.String())
	err := r.client.Get(ctx, path, &reset)
	if err != nil {
		return nil, err
	}
	return &reset, nil
}

// Execute performs a reset on the server.
func (r *ResetService) Execute(ctx context.Context, serverID ServerID, resetType ResetType) (*Reset, error) {
	var reset Reset
	path := fmt.Sprintf("/reset/%s", serverID.String())

	data := url.Values{}
	data.Set("type", string(resetType))

	err := r.client.Post(ctx, path, data, &reset)
	if err != nil {
		return nil, err
	}

	return &reset, nil
}

// ExecuteSoftware performs a software reset (CTRL+ALT+DEL).
func (r *ResetService) ExecuteSoftware(ctx context.Context, serverID ServerID) (*Reset, error) {
	return r.Execute(ctx, serverID, ResetTypeSoftware)
}

// ExecuteHardware performs a hardware reset (reset button).
func (r *ResetService) ExecuteHardware(ctx context.Context, serverID ServerID) (*Reset, error) {
	return r.Execute(ctx, serverID, ResetTypeHardware)
}

// ExecutePower performs a power cycle.
func (r *ResetService) ExecutePower(ctx context.Context, serverID ServerID) (*Reset, error) {
	return r.Execute(ctx, serverID, ResetTypePower)
}
