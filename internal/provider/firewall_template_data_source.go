// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

// Ensure the implementation satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &FirewallTemplateDataSource{}

// NewFirewallTemplateDataSource is a helper function to simplify the provider implementation.
func NewFirewallTemplateDataSource() datasource.DataSource {
	return &FirewallTemplateDataSource{}
}

// FirewallTemplateDataSource is the data source implementation.
type FirewallTemplateDataSource struct {
	client *hrobot.Client
}

// FirewallTemplateDataSourceModel describes the data source data model.
type FirewallTemplateDataSourceModel struct {
	ID                       types.String        `tfsdk:"id"`
	Name                     types.String        `tfsdk:"name"`
	FilterIPv6               types.Bool          `tfsdk:"filter_ipv6"`
	WhitelistHetznerServices types.Bool          `tfsdk:"whitelist_hetzner_services"`
	IsDefault                types.Bool          `tfsdk:"is_default"`
	InputRules               []FirewallRuleModel `tfsdk:"input_rules"`
	OutputRules              []FirewallRuleModel `tfsdk:"output_rules"`
}

// Metadata returns the data source type name.
func (d *FirewallTemplateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_template"
}

// Schema defines the schema for the data source.
func (d *FirewallTemplateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a firewall template.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Template ID",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Template name",
				Computed:            true,
			},
			"filter_ipv6": schema.BoolAttribute{
				MarkdownDescription: "Filter IPv6 traffic",
				Computed:            true,
			},
			"whitelist_hetzner_services": schema.BoolAttribute{
				MarkdownDescription: "Whitelist Hetzner services",
				Computed:            true,
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "Whether this is a default template",
				Computed:            true,
			},
			"input_rules": schema.ListNestedAttribute{
				MarkdownDescription: "Input firewall rules",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Rule name",
							Computed:            true,
						},
						"ip_version": schema.StringAttribute{
							MarkdownDescription: "IP version (ipv4 or ipv6)",
							Computed:            true,
						},
						"action": schema.StringAttribute{
							MarkdownDescription: "Action (accept or discard)",
							Computed:            true,
						},
						"protocol": schema.StringAttribute{
							MarkdownDescription: "Protocol (tcp, udp, icmp, esp, gre)",
							Computed:            true,
						},
						"source_ip": schema.StringAttribute{
							MarkdownDescription: "Source IP address or CIDR",
							Computed:            true,
						},
						"dest_ip": schema.StringAttribute{
							MarkdownDescription: "Destination IP address or CIDR",
							Computed:            true,
						},
						"source_port": schema.StringAttribute{
							MarkdownDescription: "Source port or port range",
							Computed:            true,
						},
						"destination_port": schema.StringAttribute{
							MarkdownDescription: "Destination port or port range",
							Computed:            true,
						},
						"tcp_flags": schema.StringAttribute{
							MarkdownDescription: "TCP flags",
							Computed:            true,
						},
					},
				},
			},
			"output_rules": schema.ListNestedAttribute{
				MarkdownDescription: "Output firewall rules",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Rule name",
							Computed:            true,
						},
						"ip_version": schema.StringAttribute{
							MarkdownDescription: "IP version (ipv4 or ipv6)",
							Computed:            true,
						},
						"action": schema.StringAttribute{
							MarkdownDescription: "Action (accept or discard)",
							Computed:            true,
						},
						"protocol": schema.StringAttribute{
							MarkdownDescription: "Protocol (tcp, udp, icmp, esp, gre)",
							Computed:            true,
						},
						"source_ip": schema.StringAttribute{
							MarkdownDescription: "Source IP address or CIDR",
							Computed:            true,
						},
						"dest_ip": schema.StringAttribute{
							MarkdownDescription: "Destination IP address or CIDR",
							Computed:            true,
						},
						"source_port": schema.StringAttribute{
							MarkdownDescription: "Source port or port range",
							Computed:            true,
						},
						"destination_port": schema.StringAttribute{
							MarkdownDescription: "Destination port or port range",
							Computed:            true,
						},
						"tcp_flags": schema.StringAttribute{
							MarkdownDescription: "TCP flags",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *FirewallTemplateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*hrobot.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *hrobot.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *FirewallTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FirewallTemplateDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get template from API
	template, err := d.client.Firewall.GetTemplate(ctx, data.ID.ValueString())
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

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
