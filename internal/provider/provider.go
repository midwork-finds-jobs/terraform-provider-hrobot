// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

// Ensure HetznerRobotProvider satisfies various provider interfaces.
var _ provider.Provider = &HetznerRobotProvider{}

// HetznerRobotProvider defines the provider implementation.
type HetznerRobotProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// HetznerRobotProviderModel describes the provider data model.
type HetznerRobotProviderModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func (p *HetznerRobotProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "hrobot"
	resp.Version = p.version
}

func (p *HetznerRobotProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Hetzner Dedicated Servers Provider",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				MarkdownDescription: "Hetzner Robot API username. Can also be set via HROBOT_USERNAME environment variable.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Hetzner Robot API password. Can also be set via HROBOT_PASSWORD environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *HetznerRobotProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data HetznerRobotProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get username from configuration or environment variable
	username := data.Username.ValueString()
	if username == "" {
		username = os.Getenv("HROBOT_USERNAME")
	}

	// Get password from configuration or environment variable
	password := data.Password.ValueString()
	if password == "" {
		password = os.Getenv("HROBOT_PASSWORD")
	}

	// Validate that credentials are provided
	if username == "" {
		resp.Diagnostics.AddError(
			"missing username configuration",
			"username must be set in provider configuration or via HROBOT_USERNAME environment variable",
		)
	}

	if password == "" {
		resp.Diagnostics.AddError(
			"missing password configuration",
			"password must be set in provider configuration or via HROBOT_PASSWORD environment variable",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create hrobot client
	var opts []hrobot.ClientOption

	// Enable debug logging if HROBOT_DEBUG environment variable is set
	if os.Getenv("HROBOT_DEBUG") != "" {
		opts = append(opts, hrobot.WithDebug(true))
	}

	client := hrobot.NewClient(username, password, opts...)

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *HetznerRobotProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewFirewallResource,
		NewFirewallTemplateResource,
		NewSSHKeyResource,
		NewServerResource,
		NewVSwitchResource,
		NewRDNSResource,
		NewFailoverResource,
	}
}

func (p *HetznerRobotProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewFirewallDataSource,
		NewFirewallTemplateDataSource,
		NewAuctionServersDataSource,
		NewServerDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &HetznerRobotProvider{
			version: version,
		}
	}
}
