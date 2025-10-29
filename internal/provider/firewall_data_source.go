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

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &FirewallDataSource{}

func NewFirewallDataSource() datasource.DataSource {
	return &FirewallDataSource{}
}

// FirewallDataSource defines the data source implementation.
type FirewallDataSource struct {
	client *hrobot.Client
}

// FirewallDataSourceModel describes the data source data model.
type FirewallDataSourceModel struct {
	ServerID                 types.Int64         `tfsdk:"server_id"`
	ServerIP                 types.String        `tfsdk:"server_ip"`
	Status                   types.String        `tfsdk:"status"`
	WhitelistHetznerServices types.Bool          `tfsdk:"whitelist_hetzner_services"`
	Port                     types.String        `tfsdk:"port"`
	InputRules               []FirewallRuleModel `tfsdk:"input_rules"`
	OutputRules              []FirewallRuleModel `tfsdk:"output_rules"`
	ID                       types.String        `tfsdk:"id"`
}

func (d *FirewallDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall"
}

func (d *FirewallDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Hetzner Robot firewall configuration data source",

		Attributes: map[string]schema.Attribute{
			"server_id": schema.Int64Attribute{
				MarkdownDescription: "Server ID to get firewall configuration for",
				Required:            true,
			},
			"server_ip": schema.StringAttribute{
				MarkdownDescription: "IP address of the server",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Firewall status: 'active' or 'disabled'",
				Computed:            true,
			},
			"whitelist_hetzner_services": schema.BoolAttribute{
				MarkdownDescription: "Whether Hetzner services are whitelisted",
				Computed:            true,
			},
			"port": schema.StringAttribute{
				MarkdownDescription: "Port for firewall (main or kvm)",
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
							MarkdownDescription: "IP version: 'ipv4' or 'ipv6'",
							Computed:            true,
						},
						"action": schema.StringAttribute{
							MarkdownDescription: "Action: 'accept' or 'discard'",
							Computed:            true,
						},
						"protocol": schema.StringAttribute{
							MarkdownDescription: "Protocol: 'tcp', 'udp', 'icmp', 'esp', 'gre'",
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
						"dest_port": schema.StringAttribute{
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
							MarkdownDescription: "IP version: 'ipv4' or 'ipv6'",
							Computed:            true,
						},
						"action": schema.StringAttribute{
							MarkdownDescription: "Action: 'accept' or 'discard'",
							Computed:            true,
						},
						"protocol": schema.StringAttribute{
							MarkdownDescription: "Protocol: 'tcp', 'udp', 'icmp', 'esp', 'gre'",
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
						"dest_port": schema.StringAttribute{
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
			"id": schema.StringAttribute{
				MarkdownDescription: "Firewall identifier (server number as string)",
				Computed:            true,
			},
		},
	}
}

func (d *FirewallDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*hrobot.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"unexpected data source configure type",
			fmt.Sprintf("expected *hrobot.Client, got: %T. please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *FirewallDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FirewallDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert server number to ServerID
	serverID := hrobot.ServerID(int(data.ServerID.ValueInt64()))

	// Get firewall configuration from API
	firewallConfig, err := d.client.Firewall.Get(ctx, serverID)
	if err != nil {
		resp.Diagnostics.AddError("client error", fmt.Sprintf("unable to read firewall configuration, got error: %s", err))
		return
	}

	// Map response data to model
	data.ID = types.StringValue(fmt.Sprintf("%d", firewallConfig.ServerNumber))
	data.ServerIP = types.StringValue(firewallConfig.ServerIP)
	data.ServerID = types.Int64Value(int64(firewallConfig.ServerNumber))
	data.Status = types.StringValue(string(firewallConfig.Status))
	data.WhitelistHetznerServices = types.BoolValue(firewallConfig.WhitelistHOS)
	data.Port = types.StringValue(firewallConfig.Port)

	// Convert input rules from API response
	if len(firewallConfig.Rules.Input) > 0 {
		data.InputRules = make([]FirewallRuleModel, len(firewallConfig.Rules.Input))
		for i, rule := range firewallConfig.Rules.Input {
			data.InputRules[i] = convertFromHRobotRule(rule)
		}
	}

	// Convert output rules from API response
	if len(firewallConfig.Rules.Output) > 0 {
		data.OutputRules = make([]FirewallRuleModel, len(firewallConfig.Rules.Output))
		for i, rule := range firewallConfig.Rules.Output {
			data.OutputRules[i] = convertFromHRobotRule(rule)
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
