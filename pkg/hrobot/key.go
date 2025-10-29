package hrobot

import (
	"context"
	"fmt"
	"net/url"
)

// KeyService provides access to SSH key related functions in the Hetzner Robot API.
type KeyService struct {
	client *Client
}

// NewKeyService creates a new KeyService.
func NewKeyService(client *Client) *KeyService {
	return &KeyService{client: client}
}

// SSHKey represents an SSH public key stored in Hetzner Robot.
type SSHKey struct {
	Name        string     `json:"name"`
	Fingerprint string     `json:"fingerprint"`
	Type        string     `json:"type"`
	Size        int        `json:"size"`
	Data        string     `json:"data"`
	CreatedAt   BerlinTime `json:"created_at"`
}

// SSHKeyReference represents SSH key metadata without the key data itself.
type SSHKeyReference struct {
	Name        string     `json:"name"`
	Fingerprint string     `json:"fingerprint"`
	Type        string     `json:"type"`
	Size        int        `json:"size"`
	CreatedAt   BerlinTime `json:"created_at"`
}

// List retrieves all SSH keys.
//
// GET /key
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-key
func (k *KeyService) List(ctx context.Context) ([]SSHKey, error) {
	path := "/key"
	var result []SSHKey
	if err := k.client.GetWrappedList(ctx, path, "key", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Get retrieves a specific SSH key by fingerprint.
//
// GET /key/{fingerprint}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-key-fingerprint
func (k *KeyService) Get(ctx context.Context, fingerprint string) (*SSHKey, error) {
	path := fmt.Sprintf("/key/%s", url.PathEscape(fingerprint))
	var result SSHKey
	if err := k.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Create uploads a new SSH key.
//
// POST /key
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-key
func (k *KeyService) Create(ctx context.Context, name, data string) (*SSHKey, error) {
	path := "/key"
	formData := url.Values{}
	formData.Set("name", name)
	formData.Set("data", data)

	var result SSHKey
	if err := k.client.Post(ctx, path, formData, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Rename updates the name of an existing SSH key.
//
// POST /key/{fingerprint}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-key-fingerprint
func (k *KeyService) Rename(ctx context.Context, fingerprint, newName string) (*SSHKey, error) {
	path := fmt.Sprintf("/key/%s", url.PathEscape(fingerprint))
	formData := url.Values{}
	formData.Set("name", newName)

	var result SSHKey
	if err := k.client.Post(ctx, path, formData, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete removes an SSH key from Hetzner Robot.
//
// DELETE /key/{fingerprint}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#delete-key-fingerprint
func (k *KeyService) Delete(ctx context.Context, fingerprint string) error {
	path := fmt.Sprintf("/key/%s", url.PathEscape(fingerprint))
	return k.client.Delete(ctx, path)
}
