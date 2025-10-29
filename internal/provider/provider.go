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

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &ScaffoldingProvider{}

// ScaffoldingProvider defines the provider implementation.
type ScaffoldingProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ScaffoldingProviderModel describes the provider data model.
type ScaffoldingProviderModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Endpoint types.String `tfsdk:"endpoint"`
}

func (p *ScaffoldingProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "hrobot"
	resp.Version = p.version
}

func (p *ScaffoldingProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provider for Hetzner Robot API",
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
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Hetzner Robot API endpoint URL. Defaults to https://robot-ws.your-server.de",
				Optional:            true,
			},
		},
	}
}

func (p *ScaffoldingProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ScaffoldingProviderModel

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

	// Create hrobot client with optional custom endpoint
	var clientOpts []hrobot.ClientOption
	if !data.Endpoint.IsNull() && data.Endpoint.ValueString() != "" {
		clientOpts = append(clientOpts, hrobot.WithBaseURL(data.Endpoint.ValueString()))
	}

	client := hrobot.NewClient(username, password, clientOpts...)

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ScaffoldingProvider) Resources(ctx context.Context) []func() resource.Resource {
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

func (p *ScaffoldingProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewFirewallDataSource,
		NewFirewallTemplateDataSource,
		NewAuctionServersDataSource,
		NewServerDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ScaffoldingProvider{
			version: version,
		}
	}
}
