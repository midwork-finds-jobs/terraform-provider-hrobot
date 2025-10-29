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
var _ datasource.DataSource = &ServerDataSource{}

// NewServerDataSource is a helper function to simplify the provider implementation.
func NewServerDataSource() datasource.DataSource {
	return &ServerDataSource{}
}

// ServerDataSource is the data source implementation.
type ServerDataSource struct {
	client *hrobot.Client
}

// ServerDataSourceModel describes the data source data model.
type ServerDataSourceModel struct {
	ServerID   types.Int64  `tfsdk:"server_id"`
	ServerIP   types.String `tfsdk:"server_ip"`
	ServerName types.String `tfsdk:"server_name"`
	Product    types.String `tfsdk:"product"`
	Datacenter types.String `tfsdk:"datacenter"`
	Traffic    types.String `tfsdk:"traffic"`
	Status     types.String `tfsdk:"status"`
	Cancelled  types.Bool   `tfsdk:"cancelled"`
	PaidUntil  types.String `tfsdk:"paid_until"`
}

// Metadata returns the data source type name.
func (d *ServerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

// Schema defines the schema for the data source.
func (d *ServerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches information about a specific server.",
		Attributes: map[string]schema.Attribute{
			"server_id": schema.Int64Attribute{
				MarkdownDescription: "Server ID",
				Required:            true,
			},
			"server_ip": schema.StringAttribute{
				MarkdownDescription: "Primary server IP address",
				Computed:            true,
			},
			"server_name": schema.StringAttribute{
				MarkdownDescription: "Server name",
				Computed:            true,
			},
			"product": schema.StringAttribute{
				MarkdownDescription: "Server product model",
				Computed:            true,
			},
			"datacenter": schema.StringAttribute{
				MarkdownDescription: "Datacenter location",
				Computed:            true,
			},
			"traffic": schema.StringAttribute{
				MarkdownDescription: "Traffic limit",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Server status (ready, in process, etc.)",
				Computed:            true,
			},
			"cancelled": schema.BoolAttribute{
				MarkdownDescription: "Whether the server is cancelled",
				Computed:            true,
			},
			"paid_until": schema.StringAttribute{
				MarkdownDescription: "Paid until date",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *ServerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *ServerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ServerDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get server from API
	serverID := int(config.ServerID.ValueInt64())
	server, err := d.client.Server.Get(ctx, hrobot.ServerID(serverID))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading server",
			fmt.Sprintf("Could not read server %d: %s", serverID, err.Error()),
		)
		return
	}

	// Map API response to data source model
	config.ServerIP = types.StringValue(server.ServerIP.String())
	config.ServerName = types.StringValue(server.ServerName)
	config.Product = types.StringValue(server.Product)
	config.Datacenter = types.StringValue(server.DC)
	config.Traffic = types.StringValue(server.Traffic.String())
	config.Status = types.StringValue(string(server.Status))
	config.Cancelled = types.BoolValue(server.Cancelled)
	config.PaidUntil = types.StringValue(server.PaidUntil)

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
