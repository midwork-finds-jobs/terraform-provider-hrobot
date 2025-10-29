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
var _ datasource.DataSource = &AuctionServersDataSource{}

// NewAuctionServersDataSource is a helper function to simplify the provider implementation.
func NewAuctionServersDataSource() datasource.DataSource {
	return &AuctionServersDataSource{}
}

// AuctionServersDataSource is the data source implementation.
type AuctionServersDataSource struct {
	client *hrobot.Client
}

// AuctionServersDataSourceModel describes the data source data model.
type AuctionServersDataSourceModel struct {
	Servers []AuctionServerModel `tfsdk:"servers"`
	ID      types.String         `tfsdk:"id"`
}

// AuctionServerModel describes a single auction server.
type AuctionServerModel struct {
	ID             types.Int64    `tfsdk:"id"`
	Name           types.String   `tfsdk:"name"`
	Description    []types.String `tfsdk:"description"`
	Traffic        types.String   `tfsdk:"traffic"`
	Datacenter     types.String   `tfsdk:"datacenter"`
	CPU            types.String   `tfsdk:"cpu"`
	CPUBenchmark   types.Int64    `tfsdk:"cpu_benchmark"`
	MemorySize     types.Float64  `tfsdk:"memory_size"`
	HDDSize        types.Float64  `tfsdk:"hdd_size"`
	HDDText        types.String   `tfsdk:"hdd_text"`
	HDDCount       types.Int64    `tfsdk:"hdd_count"`
	Price          types.Float64  `tfsdk:"price"`
	PriceVAT       types.Float64  `tfsdk:"price_vat"`
	PriceSetup     types.Float64  `tfsdk:"price_setup"`
	PriceSetupVAT  types.Float64  `tfsdk:"price_setup_vat"`
	PriceHourly    types.Float64  `tfsdk:"price_hourly"`
	PriceHourlyVAT types.Float64  `tfsdk:"price_hourly_vat"`
	FixedPrice     types.Bool     `tfsdk:"fixed_price"`
	NextReduce     types.Int64    `tfsdk:"next_reduce"`
	NextReduceDate types.String   `tfsdk:"next_reduce_date"`
}

// Metadata returns the data source type name.
func (d *AuctionServersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auction_servers"
}

// Schema defines the schema for the data source.
func (d *AuctionServersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches all servers currently available on the Hetzner auction/server market.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Placeholder identifier (always set to 'auction_servers')",
				Computed:            true,
			},
			"servers": schema.ListNestedAttribute{
				MarkdownDescription: "List of auction servers",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "Unique auction server ID",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Server product name",
							Computed:            true,
						},
						"description": schema.ListAttribute{
							MarkdownDescription: "List of server features",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"traffic": schema.StringAttribute{
							MarkdownDescription: "Monthly traffic limit",
							Computed:            true,
						},
						"datacenter": schema.StringAttribute{
							MarkdownDescription: "Datacenter location",
							Computed:            true,
						},
						"cpu": schema.StringAttribute{
							MarkdownDescription: "CPU model name",
							Computed:            true,
						},
						"cpu_benchmark": schema.Int64Attribute{
							MarkdownDescription: "CPU benchmark score",
							Computed:            true,
						},
						"memory_size": schema.Float64Attribute{
							MarkdownDescription: "Memory size in GB",
							Computed:            true,
						},
						"hdd_size": schema.Float64Attribute{
							MarkdownDescription: "Primary HDD size in GB",
							Computed:            true,
						},
						"hdd_text": schema.StringAttribute{
							MarkdownDescription: "Human-readable storage description",
							Computed:            true,
						},
						"hdd_count": schema.Int64Attribute{
							MarkdownDescription: "Number of primary HDDs",
							Computed:            true,
						},
						"price": schema.Float64Attribute{
							MarkdownDescription: "Monthly price (net)",
							Computed:            true,
						},
						"price_vat": schema.Float64Attribute{
							MarkdownDescription: "Monthly price (gross, including VAT)",
							Computed:            true,
						},
						"price_setup": schema.Float64Attribute{
							MarkdownDescription: "One-time setup price (net)",
							Computed:            true,
						},
						"price_setup_vat": schema.Float64Attribute{
							MarkdownDescription: "One-time setup price (gross, including VAT)",
							Computed:            true,
						},
						"price_hourly": schema.Float64Attribute{
							MarkdownDescription: "Hourly price (net)",
							Computed:            true,
						},
						"price_hourly_vat": schema.Float64Attribute{
							MarkdownDescription: "Hourly price (gross, including VAT)",
							Computed:            true,
						},
						"fixed_price": schema.BoolAttribute{
							MarkdownDescription: "Whether the price is fixed (won't be reduced further)",
							Computed:            true,
						},
						"next_reduce": schema.Int64Attribute{
							MarkdownDescription: "Seconds until next price reduction",
							Computed:            true,
						},
						"next_reduce_date": schema.StringAttribute{
							MarkdownDescription: "Timestamp of next price reduction",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *AuctionServersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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
func (d *AuctionServersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AuctionServersDataSourceModel

	// Get auction servers from API
	servers, err := d.client.Auction.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading auction servers",
			fmt.Sprintf("Could not read auction servers: %s", err.Error()),
		)
		return
	}

	// Map API response to Terraform state
	state.ID = types.StringValue("auction_servers")
	state.Servers = make([]AuctionServerModel, len(servers))

	for i, server := range servers {
		descriptions := make([]types.String, len(server.Description))
		for j, desc := range server.Description {
			descriptions[j] = types.StringValue(desc)
		}

		datacenter := types.StringNull()
		if server.Datacenter != nil {
			datacenter = types.StringValue(*server.Datacenter)
		}

		state.Servers[i] = AuctionServerModel{
			ID:             types.Int64Value(int64(server.ID)),
			Name:           types.StringValue(server.Name),
			Description:    descriptions,
			Traffic:        types.StringValue(server.Traffic),
			Datacenter:     datacenter,
			CPU:            types.StringValue(server.CPU),
			CPUBenchmark:   types.Int64Value(int64(server.CPUBenchmark)),
			MemorySize:     types.Float64Value(server.MemorySize),
			HDDSize:        types.Float64Value(server.HDDSize),
			HDDText:        types.StringValue(server.HDDText),
			HDDCount:       types.Int64Value(int64(server.HDDCount)),
			Price:          types.Float64Value(server.Price.Float64()),
			PriceVAT:       types.Float64Value(server.PriceVAT.Float64()),
			PriceSetup:     types.Float64Value(server.PriceSetup.Float64()),
			PriceSetupVAT:  types.Float64Value(server.PriceSetupVAT.Float64()),
			PriceHourly:    types.Float64Value(server.PriceHourly.Float64()),
			PriceHourlyVAT: types.Float64Value(server.PriceHourlyVAT.Float64()),
			FixedPrice:     types.BoolValue(server.FixedPrice),
			NextReduce:     types.Int64Value(server.NextReduce),
			NextReduceDate: types.StringValue(server.NextReduceDate),
		}
	}

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
