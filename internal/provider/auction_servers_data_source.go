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
	Filters *AuctionFiltersModel `tfsdk:"filters"`
}

// AuctionFiltersModel describes the optional filter criteria for auction servers.
type AuctionFiltersModel struct {
	Datacenter []types.String `tfsdk:"datacenter"`
	MinRAM     types.Int64    `tfsdk:"min_ram"`
	MinHDD     types.Float64  `tfsdk:"min_hdd"`
	MaxPrice   types.Float64  `tfsdk:"max_price"`
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
		MarkdownDescription: "Fetches servers currently available on the Hetzner auction/server market.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Placeholder identifier (always set to 'auction_servers')",
				Computed:            true,
			},
			"filters": schema.SingleNestedAttribute{
				MarkdownDescription: "Optional filters to narrow down results.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"datacenter": schema.ListAttribute{
						MarkdownDescription: "List of datacenter names to include (e.g., [\"FSN1-DC14\", \"NBG1-DC3\"]).",
						Optional:            true,
						ElementType:         types.StringType,
					},
					"min_ram": schema.Int64Attribute{
						MarkdownDescription: "Minimum RAM in MB (e.g., 32768 for 32GB).",
						Optional:            true,
					},
					"min_hdd": schema.Float64Attribute{
						MarkdownDescription: "Minimum disk size in GB (e.g., 2000 for 2TB).",
						Optional:            true,
					},
					"max_price": schema.Float64Attribute{
						MarkdownDescription: "Maximum monthly price in euros (net, e.g., 50.00).",
						Optional:            true,
					},
				},
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

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get auction servers from API
	servers, err := d.client.Auction.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading auction servers",
			fmt.Sprintf("Could not read auction servers: %s", err.Error()),
		)
		return
	}

	// Apply filters if provided
	if state.Filters != nil {
		f := state.Filters
		var filtered []hrobot.AuctionServer
		for _, s := range servers {
			// Datacenter filter
			if len(f.Datacenter) > 0 {
				if s.Datacenter == nil {
					continue
				}
				matched := false
				for _, dc := range f.Datacenter {
					if *s.Datacenter == dc.ValueString() {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}
			// min_ram filter (MB): MemorySize is in GB
			if !f.MinRAM.IsNull() && !f.MinRAM.IsUnknown() {
				minRAMGB := float64(f.MinRAM.ValueInt64()) / 1024.0
				if s.MemorySize < minRAMGB {
					continue
				}
			}
			// min_hdd filter (GB)
			if !f.MinHDD.IsNull() && !f.MinHDD.IsUnknown() {
				if s.HDDSize < f.MinHDD.ValueFloat64() {
					continue
				}
			}
			// max_price filter (euros net)
			if !f.MaxPrice.IsNull() && !f.MaxPrice.IsUnknown() {
				if s.Price.Float64() > f.MaxPrice.ValueFloat64() {
					continue
				}
			}
			filtered = append(filtered, s)
		}
		servers = filtered
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
