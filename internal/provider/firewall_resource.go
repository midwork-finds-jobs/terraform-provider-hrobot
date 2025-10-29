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
	TemplateID               types.String        `tfsdk:"template_id"`
	InputRules               []FirewallRuleModel `tfsdk:"input_rules"`
	OutputRules              []FirewallRuleModel `tfsdk:"output_rules"`
	ID                       types.String        `tfsdk:"id"`
}

// FirewallRuleModel describes a firewall rule.
type FirewallRuleModel struct {
	Name       types.String `tfsdk:"name"`
	IPVersion  types.String `tfsdk:"ip_version"`
	Action     types.String `tfsdk:"action"`
	Protocol   types.String `tfsdk:"protocol"`
	SourceIP   types.String `tfsdk:"source_ip"`
	DestIP     types.String `tfsdk:"dest_ip"`
	SourcePort types.String `tfsdk:"source_port"`
	DestPort   types.String `tfsdk:"dest_port"`
	TCPFlags   types.String `tfsdk:"tcp_flags"`
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
						"source_ip": schema.StringAttribute{
							MarkdownDescription: "Source IP address or CIDR",
							Optional:            true,
						},
						"dest_ip": schema.StringAttribute{
							MarkdownDescription: "Destination IP address or CIDR",
							Optional:            true,
						},
						"source_port": schema.StringAttribute{
							MarkdownDescription: "Source port or port range",
							Optional:            true,
						},
						"dest_port": schema.StringAttribute{
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
						"source_ip": schema.StringAttribute{
							MarkdownDescription: "Source IP address or CIDR",
							Optional:            true,
						},
						"dest_ip": schema.StringAttribute{
							MarkdownDescription: "Destination IP address or CIDR",
							Optional:            true,
						},
						"source_port": schema.StringAttribute{
							MarkdownDescription: "Source port or port range",
							Optional:            true,
						},
						"dest_port": schema.StringAttribute{
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
		}

		// Convert input rules
		if len(data.InputRules) > 0 {
			updateConfig.Rules.Input = make([]hrobot.FirewallRule, len(data.InputRules))
			for i, rule := range data.InputRules {
				updateConfig.Rules.Input[i] = convertToHRobotRule(rule)
			}
		}

		// Convert output rules
		if len(data.OutputRules) > 0 {
			updateConfig.Rules.Output = make([]hrobot.FirewallRule, len(data.OutputRules))
			for i, rule := range data.OutputRules {
				updateConfig.Rules.Output[i] = convertToHRobotRule(rule)
			}
		}

		// Update the firewall configuration
		firewallConfig, err = r.client.Firewall.Update(ctx, serverID, updateConfig)
		if err != nil {
			resp.Diagnostics.AddError("client error", fmt.Sprintf("unable to create firewall configuration, got error: %s", err))
			return
		}
	}

	// Update model with response data
	data.ID = types.StringValue(strconv.Itoa(firewallConfig.ServerNumber))
	data.ServerID = types.Int64Value(int64(firewallConfig.ServerNumber))
	data.Status = types.StringValue(string(firewallConfig.Status))
	data.WhitelistHetznerServices = types.BoolValue(firewallConfig.WhitelistHOS)

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

	// Update model with current data
	data.ServerID = types.Int64Value(int64(firewallConfig.ServerNumber))
	data.Status = types.StringValue(string(firewallConfig.Status))
	data.WhitelistHetznerServices = types.BoolValue(firewallConfig.WhitelistHOS)

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
		}

		// Convert input rules
		if len(data.InputRules) > 0 {
			updateConfig.Rules.Input = make([]hrobot.FirewallRule, len(data.InputRules))
			for i, rule := range data.InputRules {
				updateConfig.Rules.Input[i] = convertToHRobotRule(rule)
			}
		}

		// Convert output rules
		if len(data.OutputRules) > 0 {
			updateConfig.Rules.Output = make([]hrobot.FirewallRule, len(data.OutputRules))
			for i, rule := range data.OutputRules {
				updateConfig.Rules.Output[i] = convertToHRobotRule(rule)
			}
		}

		// Update the firewall configuration
		firewallConfig, err = r.client.Firewall.Update(ctx, serverID, updateConfig)
		if err != nil {
			resp.Diagnostics.AddError("client error", fmt.Sprintf("unable to update firewall configuration, got error: %s", err))
			return
		}
	}

	// Update model with response data
	data.Status = types.StringValue(string(firewallConfig.Status))
	data.WhitelistHetznerServices = types.BoolValue(firewallConfig.WhitelistHOS)

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

// Helper function to convert Terraform model rule to hrobot rule.
func convertToHRobotRule(rule FirewallRuleModel) hrobot.FirewallRule {
	return hrobot.FirewallRule{
		Name:       rule.Name.ValueString(),
		IPVersion:  hrobot.IPVersion(rule.IPVersion.ValueString()),
		Action:     hrobot.Action(rule.Action.ValueString()),
		Protocol:   hrobot.Protocol(rule.Protocol.ValueString()),
		SourceIP:   normalizeCIDR(rule.SourceIP.ValueString()),
		DestIP:     normalizeCIDR(rule.DestIP.ValueString()),
		SourcePort: rule.SourcePort.ValueString(),
		DestPort:   rule.DestPort.ValueString(),
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
	return FirewallRuleModel{
		Name:       stringOrNull(rule.Name),
		IPVersion:  stringOrNull(string(rule.IPVersion)),
		Action:     stringOrNull(string(rule.Action)),
		Protocol:   stringOrNull(string(rule.Protocol)),
		SourceIP:   stringOrNull(rule.SourceIP),
		DestIP:     stringOrNull(rule.DestIP),
		SourcePort: stringOrNull(rule.SourcePort),
		DestPort:   stringOrNull(rule.DestPort),
		TCPFlags:   stringOrNull(rule.TCPFlags),
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
func convertToAPIRules(rules []FirewallRuleModel) []hrobot.FirewallRule {
	apiRules := make([]hrobot.FirewallRule, len(rules))
	for i, rule := range rules {
		apiRules[i] = convertToHRobotRule(rule)
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
