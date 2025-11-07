// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

// Ensure the implementation satisfies the resource.Resource interface.
var _ resource.Resource = &FirewallTemplateResource{}
var _ resource.ResourceWithImportState = &FirewallTemplateResource{}

// NewFirewallTemplateResource is a helper function to simplify the provider implementation.
func NewFirewallTemplateResource() resource.Resource {
	return &FirewallTemplateResource{}
}

// FirewallTemplateResource is the resource implementation.
type FirewallTemplateResource struct {
	client *hrobot.Client
}

// FirewallTemplateResourceModel describes the resource data model.
type FirewallTemplateResourceModel struct {
	ID                       types.String        `tfsdk:"id"`
	Name                     types.String        `tfsdk:"name"`
	FilterIPv6               types.Bool          `tfsdk:"filter_ipv6"`
	WhitelistHetznerServices types.Bool          `tfsdk:"whitelist_hetzner_services"`
	IsDefault                types.Bool          `tfsdk:"is_default"`
	InputRules               []FirewallRuleModel `tfsdk:"input_rules"`
	OutputRules              []FirewallRuleModel `tfsdk:"output_rules"`
}

// Metadata returns the resource type name.
func (r *FirewallTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_template"
}

// Schema defines the schema for the resource.
func (r *FirewallTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a firewall template that can be applied to servers.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Template ID",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Template name",
				Required:            true,
			},
			"filter_ipv6": schema.BoolAttribute{
				MarkdownDescription: "filter ipv6 traffic (default: true)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"whitelist_hetzner_services": schema.BoolAttribute{
				MarkdownDescription: "whitelist hetzner services (default: true)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "whether this is a default template (default: false)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
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
							MarkdownDescription: "IP version (ipv4 or ipv6)",
							Optional:            true,
						},
						"action": schema.StringAttribute{
							MarkdownDescription: "Action (accept or discard)",
							Required:            true,
						},
						"protocol": schema.StringAttribute{
							MarkdownDescription: "Protocol (tcp, udp, icmp, esp, gre)",
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
							MarkdownDescription: "IP version (ipv4 or ipv6)",
							Optional:            true,
						},
						"action": schema.StringAttribute{
							MarkdownDescription: "Action (accept or discard)",
							Required:            true,
						},
						"protocol": schema.StringAttribute{
							MarkdownDescription: "Protocol (tcp, udp, icmp, esp, gre)",
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
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *FirewallTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*hrobot.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *hrobot.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *FirewallTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FirewallTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Terraform model to API config
	templateConfig := hrobot.TemplateConfig{
		Name:         data.Name.ValueString(),
		FilterIPv6:   data.FilterIPv6.ValueBool(),
		WhitelistHOS: data.WhitelistHetznerServices.ValueBool(),
		IsDefault:    data.IsDefault.ValueBool(),
		Rules: hrobot.FirewallRules{
			Input:  convertToAPIRules(data.InputRules),
			Output: convertToAPIRules(data.OutputRules),
		},
	}

	// Create template via API
	template, err := r.client.Firewall.CreateTemplate(ctx, templateConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating firewall template",
			fmt.Sprintf("Could not create firewall template: %s", err),
		)
		return
	}

	// Map response to model
	data.ID = types.StringValue(fmt.Sprintf("%d", template.ID))
	data.Name = types.StringValue(template.Name)
	data.FilterIPv6 = types.BoolValue(template.FilterIPv6)
	data.WhitelistHetznerServices = types.BoolValue(template.WhitelistHOS)
	data.IsDefault = types.BoolValue(template.IsDefault)
	data.InputRules = convertFromAPIRules(template.Rules.Input)
	data.OutputRules = convertFromAPIRules(template.Rules.Output)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *FirewallTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FirewallTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get template from API
	template, err := r.client.Firewall.GetTemplate(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading firewall template",
			fmt.Sprintf("Could not read firewall template %s: %s", data.ID.ValueString(), err),
		)
		return
	}

	// Map response to model
	data.Name = types.StringValue(template.Name)
	data.FilterIPv6 = types.BoolValue(template.FilterIPv6)
	data.WhitelistHetznerServices = types.BoolValue(template.WhitelistHOS)
	data.IsDefault = types.BoolValue(template.IsDefault)
	data.InputRules = convertFromAPIRules(template.Rules.Input)
	data.OutputRules = convertFromAPIRules(template.Rules.Output)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *FirewallTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FirewallTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Terraform model to API config
	templateConfig := hrobot.TemplateConfig{
		Name:         data.Name.ValueString(),
		FilterIPv6:   data.FilterIPv6.ValueBool(),
		WhitelistHOS: data.WhitelistHetznerServices.ValueBool(),
		IsDefault:    data.IsDefault.ValueBool(),
		Rules: hrobot.FirewallRules{
			Input:  convertToAPIRules(data.InputRules),
			Output: convertToAPIRules(data.OutputRules),
		},
	}

	// Update template via API
	template, err := r.client.Firewall.UpdateTemplate(ctx, data.ID.ValueString(), templateConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating firewall template",
			fmt.Sprintf("Could not update firewall template %s: %s", data.ID.ValueString(), err),
		)
		return
	}

	// Map response to model
	data.Name = types.StringValue(template.Name)
	data.FilterIPv6 = types.BoolValue(template.FilterIPv6)
	data.WhitelistHetznerServices = types.BoolValue(template.WhitelistHOS)
	data.IsDefault = types.BoolValue(template.IsDefault)
	data.InputRules = convertFromAPIRules(template.Rules.Input)
	data.OutputRules = convertFromAPIRules(template.Rules.Output)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *FirewallTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FirewallTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete template via API
	err := r.client.Firewall.DeleteTemplate(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting firewall template",
			fmt.Sprintf("Could not delete firewall template %s: %s", data.ID.ValueString(), err),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *FirewallTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
