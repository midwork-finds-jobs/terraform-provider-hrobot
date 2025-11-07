// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &FirewallResource{}
var _ resource.ResourceWithImportState = &FirewallResource{}

func NewFirewallResource() resource.Resource {
	return &FirewallResource{}
}

// FirewallResource defines the resource implementation.
type FirewallResource struct {
	client *hrobot.Client
}

// FirewallResourceModel describes the resource data model.
type FirewallResourceModel struct {
	ServerID                 types.Int64         `tfsdk:"server_id"`
	Status                   types.String        `tfsdk:"status"`
	WhitelistHetznerServices types.Bool          `tfsdk:"whitelist_hetzner_services"`
	FilterIPv6               types.Bool          `tfsdk:"filter_ipv6"`
	TemplateID               types.String        `tfsdk:"template_id"`
	InputRules               []FirewallRuleModel `tfsdk:"input_rules"`
	OutputRules              []FirewallRuleModel `tfsdk:"output_rules"`
	ID                       types.String        `tfsdk:"id"`
}

// FirewallRuleModel describes a firewall rule.
type FirewallRuleModel struct {
	Name            types.String `tfsdk:"name"`
	IPVersion       types.String `tfsdk:"ip_version"`
	Action          types.String `tfsdk:"action"`
	Protocol        types.String `tfsdk:"protocol"`
	SourceIPs       types.List   `tfsdk:"source_ips"`
	DestinationIPs  types.List   `tfsdk:"destination_ips"`
	SourcePort      types.String `tfsdk:"source_port"`
	DestinationPort types.String `tfsdk:"destination_port"`
	TCPFlags        types.String `tfsdk:"tcp_flags"`
}

func (r *FirewallResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall"
}

func (r *FirewallResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Hetzner Robot firewall configuration",

		Attributes: map[string]schema.Attribute{
			"server_id": schema.Int64Attribute{
				MarkdownDescription: "Server ID to configure firewall for",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{
					// Server number cannot be changed without recreating
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Firewall status (computed from API, typically 'active' or 'in process')",
				Computed:            true,
			},
			"whitelist_hetzner_services": schema.BoolAttribute{
				MarkdownDescription: "whitelist hetzner services (hetzner online gmbh). note: this setting is ignored when using template_id, as the template defines the whitelist setting.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"filter_ipv6": schema.BoolAttribute{
				MarkdownDescription: "enable ipv6 packet filtering. when enabled, the firewall will also filter ipv6 packets according to the configured rules. (default: true)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"template_id": schema.StringAttribute{
				MarkdownDescription: "firewall template id to apply. when set, this will apply the template rules to the server. cannot be used together with input_rules/output_rules. the whitelist_hetzner_services setting comes from the template.",
				Optional:            true,
			},
			"input_rules": schema.ListNestedAttribute{
				MarkdownDescription: "Input firewall rules",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Rule name",
							Optional:            true,
						},
						"ip_version": schema.StringAttribute{
							MarkdownDescription: "IP version: 'ipv4' or 'ipv6'",
							Optional:            true,
						},
						"action": schema.StringAttribute{
							MarkdownDescription: "Action: 'accept' or 'discard'",
							Required:            true,
						},
						"protocol": schema.StringAttribute{
							MarkdownDescription: "Protocol: 'tcp', 'udp', 'icmp', 'esp', 'gre'",
							Optional:            true,
						},
						"source_ips": schema.ListAttribute{
							MarkdownDescription: "List of source IP addresses or CIDRs. If CIDR notation is not specified, /32 will be automatically added for IPv4 addresses.",
							ElementType:         types.StringType,
							Optional:            true,
						},
						"destination_ips": schema.ListAttribute{
							MarkdownDescription: "List of destination IP addresses or CIDRs. If CIDR notation is not specified, /32 will be automatically added for IPv4 addresses.",
							ElementType:         types.StringType,
							Optional:            true,
						},
						"source_port": schema.StringAttribute{
							MarkdownDescription: "Source port or port range",
							Optional:            true,
						},
						"destination_port": schema.StringAttribute{
							MarkdownDescription: "Destination port or port range",
							Optional:            true,
						},
						"tcp_flags": schema.StringAttribute{
							MarkdownDescription: "TCP flags",
							Optional:            true,
						},
					},
				},
			},
			"output_rules": schema.ListNestedAttribute{
				MarkdownDescription: "Output firewall rules",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Rule name",
							Optional:            true,
						},
						"ip_version": schema.StringAttribute{
							MarkdownDescription: "IP version: 'ipv4' or 'ipv6'",
							Optional:            true,
						},
						"action": schema.StringAttribute{
							MarkdownDescription: "Action: 'accept' or 'discard'",
							Required:            true,
						},
						"protocol": schema.StringAttribute{
							MarkdownDescription: "Protocol: 'tcp', 'udp', 'icmp', 'esp', 'gre'",
							Optional:            true,
						},
						"source_ips": schema.ListAttribute{
							MarkdownDescription: "List of source IP addresses or CIDRs. If CIDR notation is not specified, /32 will be automatically added for IPv4 addresses.",
							ElementType:         types.StringType,
							Optional:            true,
						},
						"destination_ips": schema.ListAttribute{
							MarkdownDescription: "List of destination IP addresses or CIDRs. If CIDR notation is not specified, /32 will be automatically added for IPv4 addresses.",
							ElementType:         types.StringType,
							Optional:            true,
						},
						"source_port": schema.StringAttribute{
							MarkdownDescription: "Source port or port range",
							Optional:            true,
						},
						"destination_port": schema.StringAttribute{
							MarkdownDescription: "Destination port or port range",
							Optional:            true,
						},
						"tcp_flags": schema.StringAttribute{
							MarkdownDescription: "TCP flags",
							Optional:            true,
						},
					},
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Firewall identifier (server number as string)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *FirewallResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*hrobot.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"unexpected resource configure type",
			fmt.Sprintf("expected *hrobot.Client, got: %T. please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *FirewallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FirewallResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert server number to ServerID
	serverID := hrobot.ServerID(int(data.ServerID.ValueInt64()))

	// Wait for firewall to be ready if needed
	if err := r.client.Firewall.WaitForFirewallReady(ctx, serverID); err != nil {
		resp.Diagnostics.AddError("firewall not ready", fmt.Sprintf("firewall is busy, could not wait for it to be ready: %s", err))
		return
	}

	var firewallConfig *hrobot.FirewallConfig
	var err error

	// Check if template_id is set - if so, apply the template
	if !data.TemplateID.IsNull() && data.TemplateID.ValueString() != "" {
		// Apply template to server (whitelist setting comes from the template)
		firewallConfig, err = r.client.Firewall.ApplyTemplate(ctx, serverID, data.TemplateID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("client error", fmt.Sprintf("unable to apply firewall template, got error: %s", err))
			return
		}
	} else {
		// Build update config with manual rules
		updateConfig := hrobot.UpdateConfig{
			Status:       hrobot.FirewallStatusActive, // Always set to active when applying configuration
			WhitelistHOS: data.WhitelistHetznerServices.ValueBool(),
			FilterIPv6:   data.FilterIPv6.ValueBool(),
		}

		// Convert input rules (with array expansion)
		if len(data.InputRules) > 0 {
			updateConfig.Rules.Input = convertToAPIRules(data.InputRules)
			if len(updateConfig.Rules.Input) > 10 {
				resp.Diagnostics.AddError(
					"Too many input firewall rules after expansion",
					fmt.Sprintf("After expanding source_ips and destination_ips arrays, you have %d input rules, but Hetzner enforces a maximum of 10 rules. Please reduce the number of IPs or rules.", len(updateConfig.Rules.Input)),
				)
				return
			}
		}

		// Convert output rules (with array expansion)
		if len(data.OutputRules) > 0 {
			updateConfig.Rules.Output = convertToAPIRules(data.OutputRules)
			if len(updateConfig.Rules.Output) > 10 {
				resp.Diagnostics.AddError(
					"Too many output firewall rules after expansion",
					fmt.Sprintf("After expanding source_ips and destination_ips arrays, you have %d output rules, but Hetzner enforces a maximum of 10 rules. Please reduce the number of IPs or rules.", len(updateConfig.Rules.Output)),
				)
				return
			}
		}

		// Update the firewall configuration
		firewallConfig, err = r.client.Firewall.Update(ctx, serverID, updateConfig)
		if err != nil {
			resp.Diagnostics.AddError("client error", fmt.Sprintf("unable to create firewall configuration, got error: %s", err))
			return
		}
	}

	// Save plan values before updating from API response
	planWhitelistHetznerServices := data.WhitelistHetznerServices
	planFilterIPv6 := data.FilterIPv6

	// Update model with response data
	data.ID = types.StringValue(strconv.Itoa(firewallConfig.ServerNumber))
	data.ServerID = types.Int64Value(int64(firewallConfig.ServerNumber))
	data.Status = types.StringValue(string(firewallConfig.Status))
	// Preserve plan values - don't overwrite with template settings
	data.WhitelistHetznerServices = planWhitelistHetznerServices
	data.FilterIPv6 = planFilterIPv6

	// Write logs using the tflog package
	tflog.Trace(ctx, "created a firewall resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FirewallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FirewallResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert server number to ServerID
	serverID := hrobot.ServerID(int(data.ServerID.ValueInt64()))

	// Get current firewall configuration
	firewallConfig, err := r.client.Firewall.Get(ctx, serverID)
	if err != nil {
		resp.Diagnostics.AddError("client error", fmt.Sprintf("unable to read firewall configuration, got error: %s", err))
		return
	}

	// Save state values before updating
	stateWhitelistHetznerServices := data.WhitelistHetznerServices
	stateFilterIPv6 := data.FilterIPv6

	// Update model with current data
	data.ServerID = types.Int64Value(int64(firewallConfig.ServerNumber))
	data.Status = types.StringValue(string(firewallConfig.Status))
	// Preserve state values - don't overwrite with API values (which may come from template)
	data.WhitelistHetznerServices = stateWhitelistHetznerServices
	data.FilterIPv6 = stateFilterIPv6

	// Only update rules if not using template_id
	// When using template_id, rules come from the template and should not be stored in state
	if data.TemplateID.IsNull() || data.TemplateID.ValueString() == "" {
		// Convert input rules from API response
		if len(firewallConfig.Rules.Input) > 0 {
			data.InputRules = make([]FirewallRuleModel, len(firewallConfig.Rules.Input))
			for i, rule := range firewallConfig.Rules.Input {
				data.InputRules[i] = convertFromHRobotRule(rule)
			}
		} else {
			data.InputRules = nil
		}

		// Convert output rules from API response
		if len(firewallConfig.Rules.Output) > 0 {
			data.OutputRules = make([]FirewallRuleModel, len(firewallConfig.Rules.Output))
			for i, rule := range firewallConfig.Rules.Output {
				data.OutputRules[i] = convertFromHRobotRule(rule)
			}
		} else {
			data.OutputRules = nil
		}
	} else {
		// When using template_id, explicitly set rules to nil to avoid drift
		data.InputRules = nil
		data.OutputRules = nil
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FirewallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FirewallResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert server number to ServerID
	serverID := hrobot.ServerID(int(data.ServerID.ValueInt64()))

	// Wait for firewall to be ready if needed
	if err := r.client.Firewall.WaitForFirewallReady(ctx, serverID); err != nil {
		resp.Diagnostics.AddError("firewall not ready", fmt.Sprintf("firewall is busy, could not wait for it to be ready: %s", err))
		return
	}

	var firewallConfig *hrobot.FirewallConfig
	var err error

	// Check if template_id is set - if so, apply the template
	if !data.TemplateID.IsNull() && data.TemplateID.ValueString() != "" {
		// Apply template to server (whitelist setting comes from the template)
		firewallConfig, err = r.client.Firewall.ApplyTemplate(ctx, serverID, data.TemplateID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("client error", fmt.Sprintf("unable to apply firewall template, got error: %s", err))
			return
		}
	} else {
		// Build update config with manual rules
		updateConfig := hrobot.UpdateConfig{
			Status:       hrobot.FirewallStatusActive, // Always set to active when applying configuration
			WhitelistHOS: data.WhitelistHetznerServices.ValueBool(),
			FilterIPv6:   data.FilterIPv6.ValueBool(),
		}

		// Convert input rules (with array expansion)
		if len(data.InputRules) > 0 {
			updateConfig.Rules.Input = convertToAPIRules(data.InputRules)
			if len(updateConfig.Rules.Input) > 10 {
				resp.Diagnostics.AddError(
					"Too many input firewall rules after expansion",
					fmt.Sprintf("After expanding source_ips and destination_ips arrays, you have %d input rules, but Hetzner enforces a maximum of 10 rules. Please reduce the number of IPs or rules.", len(updateConfig.Rules.Input)),
				)
				return
			}
		}

		// Convert output rules (with array expansion)
		if len(data.OutputRules) > 0 {
			updateConfig.Rules.Output = convertToAPIRules(data.OutputRules)
			if len(updateConfig.Rules.Output) > 10 {
				resp.Diagnostics.AddError(
					"Too many output firewall rules after expansion",
					fmt.Sprintf("After expanding source_ips and destination_ips arrays, you have %d output rules, but Hetzner enforces a maximum of 10 rules. Please reduce the number of IPs or rules.", len(updateConfig.Rules.Output)),
				)
				return
			}
		}

		// Update the firewall configuration
		firewallConfig, err = r.client.Firewall.Update(ctx, serverID, updateConfig)
		if err != nil {
			resp.Diagnostics.AddError("client error", fmt.Sprintf("unable to update firewall configuration, got error: %s", err))
			return
		}
	}

	// Save plan values before updating from API response
	planWhitelistHetznerServices := data.WhitelistHetznerServices
	planFilterIPv6 := data.FilterIPv6

	// Update model with response data
	data.Status = types.StringValue(string(firewallConfig.Status))
	// Preserve plan values - don't overwrite with template settings
	data.WhitelistHetznerServices = planWhitelistHetznerServices
	data.FilterIPv6 = planFilterIPv6

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FirewallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FirewallResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Note: We don't actually disable the firewall here as that would leave the server unprotected.
	// The firewall configuration will remain active on the server.
	// Users need to manually disable or modify the firewall through the Robot interface if needed.
	resp.Diagnostics.AddWarning(
		"Firewall removed from Terraform state",
		fmt.Sprintf("The firewall configuration for server %d has been removed from Terraform state, but the firewall remains active on the server. You must manually disable or modify the firewall through Hetzner Robot if you want to change it.", data.ServerID.ValueInt64()),
	)
}

func (r *FirewallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using server number as ID
	serverNum, err := strconv.Atoi(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("invalid import id", fmt.Sprintf("import ID must be a valid server number, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), int64(serverNum))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// Helper function to convert Terraform model rule to hrobot rule with a specific source/dest IP.
func convertToHRobotRuleWithIPs(rule FirewallRuleModel, sourceIP, destIP string) hrobot.FirewallRule {
	return hrobot.FirewallRule{
		Name:       rule.Name.ValueString(),
		IPVersion:  hrobot.IPVersion(rule.IPVersion.ValueString()),
		Action:     hrobot.Action(rule.Action.ValueString()),
		Protocol:   hrobot.Protocol(rule.Protocol.ValueString()),
		SourceIP:   normalizeCIDR(sourceIP),
		DestIP:     normalizeCIDR(destIP),
		SourcePort: rule.SourcePort.ValueString(),
		DestPort:   rule.DestinationPort.ValueString(),
		TCPFlags:   rule.TCPFlags.ValueString(),
	}
}

// Helper function to normalize IP addresses by adding /32 or /128 if CIDR is missing.
func normalizeCIDR(ip string) string {
	if ip == "" {
		return ""
	}
	// If already has CIDR notation, return as-is
	if containsChar(ip, '/') {
		return ip
	}
	// Check if it's IPv6 (contains :)
	if containsChar(ip, ':') {
		return ip + "/128"
	}
	// Assume IPv4
	return ip + "/32"
}

// Helper function to check if string contains a character.
func containsChar(s string, c rune) bool {
	for _, ch := range s {
		if ch == c {
			return true
		}
	}
	return false
}

// Helper function to convert hrobot rule to Terraform model rule.
func convertFromHRobotRule(rule hrobot.FirewallRule) FirewallRuleModel {
	// Convert single IPs to lists for consistency
	var sourceIPList types.List
	if rule.SourceIP != "" {
		sourceIPList, _ = types.ListValueFrom(context.Background(), types.StringType, []string{rule.SourceIP})
	} else {
		sourceIPList = types.ListNull(types.StringType)
	}

	var destinationIPList types.List
	if rule.DestIP != "" {
		destinationIPList, _ = types.ListValueFrom(context.Background(), types.StringType, []string{rule.DestIP})
	} else {
		destinationIPList = types.ListNull(types.StringType)
	}

	return FirewallRuleModel{
		Name:            stringOrNull(rule.Name),
		IPVersion:       stringOrNull(string(rule.IPVersion)),
		Action:          stringOrNull(string(rule.Action)),
		Protocol:        stringOrNull(string(rule.Protocol)),
		SourceIPs:       sourceIPList,
		DestinationIPs:  destinationIPList,
		SourcePort:      stringOrNull(rule.SourcePort),
		DestinationPort: stringOrNull(rule.DestPort),
		TCPFlags:        stringOrNull(rule.TCPFlags),
	}
}

// Helper function to convert string to types.String, returning null for empty strings.
func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// Helper function to convert slice of Terraform rules to API rules.
// This function expands rules with multiple source_ips or destination_ips values into separate rules.
func convertToAPIRules(rules []FirewallRuleModel) []hrobot.FirewallRule {
	var apiRules []hrobot.FirewallRule

	for _, rule := range rules {
		// Extract source IPs from the list
		var sourceIPs []string
		if !rule.SourceIPs.IsNull() && !rule.SourceIPs.IsUnknown() {
			_ = rule.SourceIPs.ElementsAs(context.Background(), &sourceIPs, false)
		}
		if len(sourceIPs) == 0 {
			sourceIPs = []string{""} // Empty string for no source IP
		}

		// Extract destination IPs from the list
		var destinationIPs []string
		if !rule.DestinationIPs.IsNull() && !rule.DestinationIPs.IsUnknown() {
			_ = rule.DestinationIPs.ElementsAs(context.Background(), &destinationIPs, false)
		}
		if len(destinationIPs) == 0 {
			destinationIPs = []string{""} // Empty string for no destination IP
		}

		// Create a rule for each combination of source and destination IPs
		for _, sourceIP := range sourceIPs {
			for _, destinationIP := range destinationIPs {
				apiRules = append(apiRules, convertToHRobotRuleWithIPs(rule, sourceIP, destinationIP))
			}
		}
	}

	return apiRules
}

// Helper function to convert slice of API rules to Terraform rules.
func convertFromAPIRules(rules []hrobot.FirewallRule) []FirewallRuleModel {
	tfRules := make([]FirewallRuleModel, len(rules))
	for i, rule := range rules {
		tfRules[i] = convertFromHRobotRule(rule)
	}
	return tfRules
}
