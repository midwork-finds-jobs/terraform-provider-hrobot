package hrobot

import (
	"context"
	"fmt"
	"net/url"

	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot/internal/urlencode"
)

// FirewallService handles firewall-related API operations.
type FirewallService struct {
	client *Client
}

// NewFirewallService creates a new firewall service.
func NewFirewallService(client *Client) *FirewallService {
	return &FirewallService{client: client}
}

// FirewallStatus represents the firewall status.
type FirewallStatus string

const (
	FirewallStatusActive   FirewallStatus = "active"
	FirewallStatusDisabled FirewallStatus = "disabled"
)

// FirewallRule represents a single firewall rule.
type FirewallRule struct {
	Name       string    `json:"name,omitempty"`
	IPVersion  IPVersion `json:"ip_version,omitempty"`
	Action     Action    `json:"action"`
	Protocol   Protocol  `json:"protocol,omitempty"`
	SourceIP   string    `json:"src_ip,omitempty"`
	DestIP     string    `json:"dst_ip,omitempty"`
	SourcePort string    `json:"src_port,omitempty"`
	DestPort   string    `json:"dst_port,omitempty"`
	TCPFlags   string    `json:"tcp_flags,omitempty"`
}

// FirewallConfig represents the complete firewall configuration.
type FirewallConfig struct {
	ServerIP     string         `json:"server_ip"`
	ServerNumber int            `json:"server_number"`
	Status       FirewallStatus `json:"status"`
	WhitelistHOS bool           `json:"whitelist_hos"`
	Port         string         `json:"port"`
	Rules        FirewallRules  `json:"rules"`
}

// FirewallRules contains input and output rules.
type FirewallRules struct {
	Input  []FirewallRule `json:"input"`
	Output []FirewallRule `json:"output"`
}

// Get retrieves the firewall configuration for a server.
func (f *FirewallService) Get(ctx context.Context, serverID ServerID) (*FirewallConfig, error) {
	var config FirewallConfig
	path := fmt.Sprintf("/firewall/%s", serverID.String())
	err := f.client.Get(ctx, path, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// UpdateConfig updates the firewall configuration.
type UpdateConfig struct {
	Status       FirewallStatus
	WhitelistHOS bool
	Rules        FirewallRules
}

// Update updates the firewall configuration for a server.
func (f *FirewallService) Update(ctx context.Context, serverID ServerID, config UpdateConfig) (*FirewallConfig, error) {
	path := fmt.Sprintf("/firewall/%s", serverID.String())

	// Build the form data with hierarchical rule encoding
	encoder := urlencode.NewFirewallRuleEncoder()

	// Add input rules
	for _, rule := range config.Rules.Input {
		ruleData := f.encodeRule(rule)
		encoder.AddInputRule(ruleData)
	}

	// Add output rules
	for _, rule := range config.Rules.Output {
		ruleData := f.encodeRule(rule)
		encoder.AddOutputRule(ruleData)
	}

	// Add status and whitelist settings
	additional := url.Values{}
	additional.Set("status", string(config.Status))
	if config.WhitelistHOS {
		additional.Set("whitelist_hos", "true")
	} else {
		additional.Set("whitelist_hos", "false")
	}

	formData := encoder.MergeValues(additional)

	var result FirewallConfig
	err := f.client.Post(ctx, path, formData, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// encodeRule converts a FirewallRule to a map for URL encoding.
func (f *FirewallService) encodeRule(rule FirewallRule) map[string]string {
	data := make(map[string]string)

	if rule.Name != "" {
		data["name"] = rule.Name
	}
	if rule.IPVersion != "" {
		data["ip_version"] = string(rule.IPVersion)
	}
	data["action"] = string(rule.Action)
	if rule.Protocol != "" {
		data["protocol"] = string(rule.Protocol)
	}
	if rule.SourceIP != "" {
		data["src_ip"] = rule.SourceIP
	}
	if rule.DestIP != "" {
		data["dst_ip"] = rule.DestIP
	}
	if rule.SourcePort != "" {
		data["src_port"] = rule.SourcePort
	}
	if rule.DestPort != "" {
		data["dst_port"] = rule.DestPort
	}
	if rule.TCPFlags != "" {
		data["tcp_flags"] = rule.TCPFlags
	}

	return data
}

// Activate activates the firewall for a server.
func (f *FirewallService) Activate(ctx context.Context, serverID ServerID) (*FirewallConfig, error) {
	path := fmt.Sprintf("/firewall/%s", serverID.String())

	data := url.Values{}
	data.Set("status", string(FirewallStatusActive))

	var config FirewallConfig
	err := f.client.Post(ctx, path, data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Disable disables the firewall for a server.
func (f *FirewallService) Disable(ctx context.Context, serverID ServerID) (*FirewallConfig, error) {
	path := fmt.Sprintf("/firewall/%s", serverID.String())

	data := url.Values{}
	data.Set("status", string(FirewallStatusDisabled))

	var config FirewallConfig
	err := f.client.Post(ctx, path, data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Delete removes all firewall rules (resets to empty configuration).
func (f *FirewallService) Delete(ctx context.Context, serverID ServerID) error {
	path := fmt.Sprintf("/firewall/%s", serverID.String())
	return f.client.Delete(ctx, path)
}

// FirewallTemplate represents a firewall template.
type FirewallTemplate struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	FilterIPv6   bool          `json:"filter_ipv6"`
	WhitelistHOS bool          `json:"whitelist_hos"`
	IsDefault    bool          `json:"is_default"`
	Rules        FirewallRules `json:"rules"`
}

// FirewallTemplateWrapper wraps the template in the API response.
type FirewallTemplateWrapper struct {
	Template FirewallTemplate `json:"firewall_template"`
}

// TemplateConfig is used for creating/updating templates.
type TemplateConfig struct {
	Name         string
	FilterIPv6   bool
	WhitelistHOS bool
	IsDefault    bool
	Rules        FirewallRules
}

// ListTemplates retrieves all firewall templates.
func (f *FirewallService) ListTemplates(ctx context.Context) ([]FirewallTemplate, error) {
	var templates []FirewallTemplate
	err := f.client.Get(ctx, "/firewall/template", &templates)
	if err != nil {
		return nil, err
	}
	return templates, nil
}

// GetTemplate retrieves a firewall template.
func (f *FirewallService) GetTemplate(ctx context.Context, templateID string) (*FirewallTemplate, error) {
	var wrapper FirewallTemplateWrapper
	path := fmt.Sprintf("/firewall/template/%s", templateID)
	err := f.client.Get(ctx, path, &wrapper)
	if err != nil {
		return nil, err
	}
	return &wrapper.Template, nil
}

// CreateTemplate creates a new firewall template.
func (f *FirewallService) CreateTemplate(ctx context.Context, config TemplateConfig) (*FirewallTemplate, error) {
	// Build the form data with hierarchical rule encoding
	encoder := urlencode.NewFirewallRuleEncoder()

	// Add input rules
	for _, rule := range config.Rules.Input {
		ruleData := f.encodeRule(rule)
		encoder.AddInputRule(ruleData)
	}

	// Add output rules
	for _, rule := range config.Rules.Output {
		ruleData := f.encodeRule(rule)
		encoder.AddOutputRule(ruleData)
	}

	// Add name, filter_ipv6, whitelist, and is_default settings
	additional := map[string]string{
		"name":          config.Name,
		"filter_ipv6":   "false",
		"whitelist_hos": "false",
		"is_default":    "false",
	}
	if config.FilterIPv6 {
		additional["filter_ipv6"] = "true"
	}
	if config.WhitelistHOS {
		additional["whitelist_hos"] = "true"
	}
	if config.IsDefault {
		additional["is_default"] = "true"
	}

	formData := encoder.EncodeToString(additional)

	var wrapper FirewallTemplateWrapper
	err := f.client.PostRaw(ctx, "/firewall/template", formData, &wrapper)
	if err != nil {
		return nil, err
	}

	return &wrapper.Template, nil
}

// UpdateTemplate updates an existing firewall template.
func (f *FirewallService) UpdateTemplate(ctx context.Context, templateID string, config TemplateConfig) (*FirewallTemplate, error) {
	path := fmt.Sprintf("/firewall/template/%s", templateID)

	// Build the form data with hierarchical rule encoding
	encoder := urlencode.NewFirewallRuleEncoder()

	// Add input rules
	for _, rule := range config.Rules.Input {
		ruleData := f.encodeRule(rule)
		encoder.AddInputRule(ruleData)
	}

	// Add output rules
	for _, rule := range config.Rules.Output {
		ruleData := f.encodeRule(rule)
		encoder.AddOutputRule(ruleData)
	}

	// Add name, filter_ipv6, whitelist, and is_default settings
	additional := map[string]string{
		"name":          config.Name,
		"filter_ipv6":   "false",
		"whitelist_hos": "false",
		"is_default":    "false",
	}
	if config.FilterIPv6 {
		additional["filter_ipv6"] = "true"
	}
	if config.WhitelistHOS {
		additional["whitelist_hos"] = "true"
	}
	if config.IsDefault {
		additional["is_default"] = "true"
	}

	formData := encoder.EncodeToString(additional)

	var wrapper FirewallTemplateWrapper
	err := f.client.PostRaw(ctx, path, formData, &wrapper)
	if err != nil {
		return nil, err
	}

	return &wrapper.Template, nil
}

// DeleteTemplate deletes a firewall template.
func (f *FirewallService) DeleteTemplate(ctx context.Context, templateID string) error {
	path := fmt.Sprintf("/firewall/template/%s", templateID)
	return f.client.Delete(ctx, path)
}

// WaitForFirewallReady waits for the firewall to be ready (not in process state).
// It polls the firewall status with exponential backoff until it's ready or the context times out.
func (f *FirewallService) WaitForFirewallReady(ctx context.Context, serverID ServerID) error {
	return waitForCondition(ctx, func() (bool, error) {
		config, err := f.Get(ctx, serverID)
		if err != nil {
			return false, err
		}
		// Check if status is not "in process"
		return config.Status != "in process", nil
	})
}

// ApplyTemplate applies a firewall template to a server.
// Note: The whitelist_hos setting comes from the template itself and cannot be overridden.
func (f *FirewallService) ApplyTemplate(ctx context.Context, serverID ServerID, templateID string) (*FirewallConfig, error) {
	path := fmt.Sprintf("/firewall/%s", serverID.String())

	data := url.Values{}
	data.Set("template_id", templateID)
	// Note: whitelist_hos cannot be passed with template_id according to API docs

	var config FirewallConfig
	err := f.client.Post(ctx, path, data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
