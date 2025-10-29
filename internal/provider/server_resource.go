// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

// Ensure the implementation satisfies the resource.Resource interface.
var _ resource.Resource = &ServerResource{}
var _ resource.ResourceWithImportState = &ServerResource{}

// NewServerResource is a helper function to simplify the provider implementation.
func NewServerResource() resource.Resource {
	return &ServerResource{}
}

// ServerResource is the resource implementation.
type ServerResource struct {
	client *hrobot.Client
}

// ServerResourceModel describes the resource data model.
type ServerResourceModel struct {
	TransactionID   types.String    `tfsdk:"transaction_id"`
	ServerType      types.String    `tfsdk:"server_type"`
	AuthorizedKeys  []types.String  `tfsdk:"authorized_keys"`
	Password        types.String    `tfsdk:"password"`
	Image           types.String    `tfsdk:"image"`
	Datacenter      types.String    `tfsdk:"datacenter"`
	Comment         types.String    `tfsdk:"comment"`
	PublicNet       *PublicNetModel `tfsdk:"public_net"`
	Status          types.String    `tfsdk:"status"`
	ServerID        types.Int64     `tfsdk:"server_id"`
	ServerName      types.String    `tfsdk:"server_name"`
	WaitForComplete types.Bool      `tfsdk:"wait_for_complete"`
}

// PublicNetModel describes the public network configuration.
type PublicNetModel struct {
	IPv4Enabled types.Bool   `tfsdk:"ipv4_enabled"`
	IPv4        types.String `tfsdk:"ipv4"`
	IPv6        types.String `tfsdk:"ipv6"`
}

// Metadata returns the resource type name.
func (r *ServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

// Schema defines the schema for the resource.
func (r *ServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Hetzner dedicated server (auction or product-based).",
		Attributes: map[string]schema.Attribute{
			"transaction_id": schema.StringAttribute{
				MarkdownDescription: "Transaction ID (computed)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"server_type": schema.StringAttribute{
				MarkdownDescription: "Server type: 'auction' for auction servers or product name like 'AX41-NVMe' for standard products",
				Required:            true,
			},
			"authorized_keys": schema.ListAttribute{
				MarkdownDescription: "SSH key fingerprints for authorization (use this OR password, not both)",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Root password (use this OR authorized_keys, not both)",
				Optional:            true,
				Sensitive:           true,
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "Image/distribution to install (default: 'Rescue system')",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("Rescue system"),
			},
			"datacenter": schema.StringAttribute{
				MarkdownDescription: "Datacenter location (required for product servers, not used for auction servers). Valid values: FSN1, HEL1, NBG1",
				Optional:            true,
			},
			"comment": schema.StringAttribute{
				MarkdownDescription: "Comment for the order (optional, may require manual provisioning)",
				Optional:            true,
			},
			"wait_for_complete": schema.BoolAttribute{
				MarkdownDescription: "Wait for the server order to complete before returning (default: true)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Transaction status",
				Computed:            true,
			},
			"server_id": schema.Int64Attribute{
				MarkdownDescription: "Server ID: For auction servers, this is the server number to purchase (required). For other servers, this is computed after provisioning.",
				Optional:            true,
				Computed:            true,
			},
			"server_name": schema.StringAttribute{
				MarkdownDescription: "Server name (required, can be updated)",
				Required:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"public_net": schema.SingleNestedBlock{
				MarkdownDescription: "Public network configuration",
				Attributes: map[string]schema.Attribute{
					"ipv4_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable primary IPv4 address (default: true)",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"ipv4": schema.StringAttribute{
						MarkdownDescription: "Primary IPv4 address (computed)",
						Computed:            true,
					},
					"ipv6": schema.StringAttribute{
						MarkdownDescription: "Primary IPv6 address (computed, always enabled)",
						Computed:            true,
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ServerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ServerResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverType := plan.ServerType.ValueString()

	// Validate server_type and server_id combination
	if serverType == "auction" || serverType == "Auction Server" {
		// For auction servers, server_id must be provided
		if plan.ServerID.IsNull() || plan.ServerID.IsUnknown() {
			resp.Diagnostics.AddError(
				"Missing server_id for auction server",
				"For server_type='auction', server_id must be provided (the server number from the auction)",
			)
			return
		}
	} else {
		// For product servers, server_id should not be provided
		if !plan.ServerID.IsNull() && !plan.ServerID.IsUnknown() {
			resp.Diagnostics.AddError(
				"Invalid server_id for product server",
				"For product-based servers, server_id should not be provided (it will be computed after provisioning)",
			)
			return
		}
		// For product servers, datacenter must be provided
		if plan.Datacenter.IsNull() || plan.Datacenter.IsUnknown() {
			resp.Diagnostics.AddError(
				"Missing datacenter for product server",
				"For product-based servers, datacenter must be provided (valid values: FSN1, HEL1, NBG1)",
			)
			return
		}
	}

	// Authorization method
	auth := hrobot.AuthorizationMethod{}
	if len(plan.AuthorizedKeys) > 0 {
		keys := make([]string, len(plan.AuthorizedKeys))
		for i, k := range plan.AuthorizedKeys {
			keys[i] = k.ValueString()
		}
		auth.Keys = keys
	} else if !plan.Password.IsNull() {
		auth.Password = plan.Password.ValueString()
	} else {
		resp.Diagnostics.AddError(
			"Missing authorization method",
			"Either authorized_keys or password must be provided",
		)
		return
	}

	// Addons
	var addons []string
	if plan.PublicNet != nil && !plan.PublicNet.IPv4Enabled.IsNull() && plan.PublicNet.IPv4Enabled.ValueBool() {
		addons = []string{"primary_ipv4"}
	}

	var transaction *hrobot.MarketTransaction
	var err error

	// Build and place order based on server type
	if serverType == "auction" {
		// Auction server order
		order := hrobot.MarketProductOrder{
			ProductID:    uint32(plan.ServerID.ValueInt64()),
			Auth:         auth,
			Distribution: plan.Image.ValueString(),
			Language:     "en",
			ServerName:   plan.ServerName.ValueString(),
			Addons:       addons,
			Test:         false,
		}
		if !plan.Comment.IsNull() {
			order.Comment = plan.Comment.ValueString()
		}

		transaction, err = r.client.Ordering.PlaceMarketOrder(ctx, order)
		if err != nil {
			// Check if server already exists when we get INVALID_INPUT error
			errMsg := err.Error()
			if serverType == "auction" && (strings.Contains(errMsg, "INVALID_INPUT") || strings.Contains(errMsg, "invalid input")) {
				// Try to fetch the server to see if it already exists
				serverID := hrobot.ServerID(plan.ServerID.ValueInt64())
				if existingServer, getErr := r.client.Server.Get(ctx, serverID); getErr == nil && existingServer != nil {
					resp.Diagnostics.AddError(
						"Server already exists",
						fmt.Sprintf("Server %d already exists. You can import it into Terraform state by running:\n\n  tofu import hrobot_server.auction %d",
							plan.ServerID.ValueInt64(),
							plan.ServerID.ValueInt64(),
						),
					)
					return
				}
			}

			resp.Diagnostics.AddError(
				"Error placing auction server order",
				fmt.Sprintf("Could not place auction server order: %s", err.Error()),
			)
			return
		}
	} else {
		// Product server order
		order := hrobot.ProductOrder{
			ProductID:    serverType,
			Auth:         auth,
			Distribution: plan.Image.ValueString(),
			Language:     "en",
			Location:     plan.Datacenter.ValueString(),
			ServerName:   plan.ServerName.ValueString(),
			Addons:       addons,
			Test:         false,
		}
		if !plan.Comment.IsNull() {
			order.Comment = plan.Comment.ValueString()
		}

		transaction, err = r.client.Ordering.PlaceProductOrder(ctx, order)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error placing product server order",
				fmt.Sprintf("Could not place product server order: %s", err.Error()),
			)
			return
		}
	}

	// Map response to resource model
	plan.TransactionID = types.StringValue(transaction.ID)
	plan.Status = types.StringValue(transaction.Status)

	if transaction.ServerNumber != nil {
		plan.ServerID = types.Int64Value(int64(*transaction.ServerNumber))
	}

	// Wait for completion if requested
	if plan.WaitForComplete.ValueBool() && transaction.Status != "ready" && transaction.Status != "cancelled" {
		finalTx, err := r.client.Ordering.WaitForMarketTransactionCompletion(ctx, transaction.ID, 30*time.Second)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error waiting for order completion",
				fmt.Sprintf("Order was placed but failed to complete: %s", err.Error()),
			)
			// Still save the state with what we have
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
			return
		}

		// Update with final state
		plan.Status = types.StringValue(finalTx.Status)
		if finalTx.ServerNumber != nil {
			plan.ServerID = types.Int64Value(int64(*finalTx.ServerNumber))
		}
	}

	// Set server name and fetch server details if server is provisioned
	if !plan.ServerID.IsNull() {
		serverID := hrobot.ServerID(plan.ServerID.ValueInt64())

		// Set the server name (required field)
		server, err := r.client.Server.SetName(ctx, serverID, plan.ServerName.ValueString())
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Failed to set server name",
				fmt.Sprintf("Server was provisioned but failed to set name: %s", err.Error()),
			)
		} else if server != nil {
			// Update server name from API response
			plan.ServerName = types.StringValue(server.ServerName)
		}

		// Fetch server details to populate public_net IPs
		server, err = r.client.Server.Get(ctx, serverID)
		if err == nil && server != nil {
			// Initialize public_net if not already set
			if plan.PublicNet == nil {
				plan.PublicNet = &PublicNetModel{
					IPv4Enabled: types.BoolValue(true),
				}
			}

			// Set IPv4 if enabled and available
			if plan.PublicNet.IPv4Enabled.ValueBool() && server.ServerIP != nil {
				plan.PublicNet.IPv4 = types.StringValue(server.ServerIP.String())
			} else {
				plan.PublicNet.IPv4 = types.StringNull()
			}

			// IPv6 is always enabled - fetch from subnets
			if len(server.Subnet) > 0 {
				for _, subnet := range server.Subnet {
					// Look for IPv6 subnet
					if subnet.IP.To4() == nil {
						plan.PublicNet.IPv6 = types.StringValue(subnet.IP.String() + "/" + subnet.Mask)
						break
					}
				}
			}
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ServerResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	txID := state.TransactionID.ValueString()

	// Check if this was imported from a server (transaction ID format: "server-XXXXX")
	isServerImport := len(txID) > 7 && txID[:7] == "server-"

	if !isServerImport {
		// Get current state from API for real transactions
		transaction, err := r.client.Ordering.GetMarketTransaction(ctx, txID)
		if err != nil {
			if hrobot.IsNotFoundError(err) {
				// Transaction no longer exists
				resp.State.RemoveResource(ctx)
				return
			}
			resp.Diagnostics.AddError(
				"Error reading market order transaction",
				fmt.Sprintf("Could not read transaction %s: %s", txID, err.Error()),
			)
			return
		}

		// Update state with latest values from API
		state.Status = types.StringValue(transaction.Status)
		if transaction.ServerNumber != nil {
			state.ServerID = types.Int64Value(int64(*transaction.ServerNumber))
		}
	}

	// For both imported and real orders, fetch server details if we have a server number
	if !state.ServerID.IsNull() {
		serverID := hrobot.ServerID(state.ServerID.ValueInt64())
		server, err := r.client.Server.Get(ctx, serverID)
		if err != nil {
			if hrobot.IsNotFoundError(err) {
				// Server no longer exists
				resp.State.RemoveResource(ctx)
				return
			}
			// Continue even if server fetch fails - might be temporary
		} else if server != nil {
			// Update state with latest server info
			state.ServerName = types.StringValue(server.ServerName)
			state.Status = types.StringValue(string(server.Status))

			// Update public_net block
			if state.PublicNet == nil {
				state.PublicNet = &PublicNetModel{
					IPv4Enabled: types.BoolValue(server.ServerIP != nil),
				}
			}

			// Set IPv4 if available
			if server.ServerIP != nil {
				state.PublicNet.IPv4 = types.StringValue(server.ServerIP.String())
			} else {
				state.PublicNet.IPv4 = types.StringNull()
			}

			// IPv6 is always enabled - fetch from subnets
			if len(server.Subnet) > 0 {
				for _, subnet := range server.Subnet {
					// Look for IPv6 subnet
					if subnet.IP.To4() == nil {
						state.PublicNet.IPv6 = types.StringValue(subnet.IP.String() + "/" + subnet.Mask)
						break
					}
				}
			}
		}
	}

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state.
func (r *ServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ServerResourceModel

	// Read Terraform plan and state data
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only server_name can be updated
	if !plan.ServerName.Equal(state.ServerName) && !plan.ServerName.IsNull() && !plan.ServerName.IsUnknown() {
		// Use state.ServerID since it contains the actual server number
		// plan.ServerID might be unknown if other fields are being added
		if state.ServerID.IsNull() {
			resp.Diagnostics.AddError(
				"Cannot update server name",
				"Server has not been provisioned yet. Wait for provisioning to complete.",
			)
			return
		}

		serverID := hrobot.ServerID(state.ServerID.ValueInt64())
		server, err := r.client.Server.SetName(ctx, serverID, plan.ServerName.ValueString())
		if err != nil {
			// Check if server doesn't exist
			if hrobot.IsNotFoundError(err) {
				resp.Diagnostics.AddError(
					"Server not found",
					fmt.Sprintf("Server %d does not exist. It may have been deleted or the import was incorrect. Please verify the server number and re-import if necessary.", state.ServerID.ValueInt64()),
				)
			} else {
				resp.Diagnostics.AddError(
					"Error updating server name",
					fmt.Sprintf("Could not update server name: %s", err.Error()),
				)
			}
			return
		}

		// Update with API response
		if server != nil {
			plan.ServerName = types.StringValue(server.ServerName)
		}
	}

	// Preserve server_id from state - it cannot change
	plan.ServerID = state.ServerID
	plan.TransactionID = state.TransactionID
	plan.Status = state.Status

	// Note: server_id cannot be changed as it identifies the server itself
	// The plan.ServerID should equal state.ServerID by the time we get here

	// For imported servers, preserve plan values for these fields if they're set in config
	// Otherwise preserve from state. These are metadata fields that don't affect the actual server.
	if !plan.Image.IsNull() && !plan.Image.IsUnknown() {
		// Keep the planned image value from config
	} else if !state.Image.IsNull() {
		plan.Image = state.Image
	}

	if len(plan.AuthorizedKeys) > 0 {
		// Keep the planned authorized_keys from config
	} else if len(state.AuthorizedKeys) > 0 {
		plan.AuthorizedKeys = state.AuthorizedKeys
	}

	// Refresh server details to get latest values
	if !state.ServerID.IsNull() {
		serverID := hrobot.ServerID(state.ServerID.ValueInt64())
		server, err := r.client.Server.Get(ctx, serverID)
		if err == nil && server != nil {
			plan.ServerName = types.StringValue(server.ServerName)

			// Update public_net block
			if plan.PublicNet == nil {
				plan.PublicNet = &PublicNetModel{
					IPv4Enabled: types.BoolValue(server.ServerIP != nil),
				}
			}

			// Set IPv4 if available
			if server.ServerIP != nil {
				plan.PublicNet.IPv4 = types.StringValue(server.ServerIP.String())
			} else {
				plan.PublicNet.IPv4 = types.StringNull()
			}

			// IPv6 is always enabled - fetch from subnets
			if len(server.Subnet) > 0 {
				for _, subnet := range server.Subnet {
					// Look for IPv6 subnet
					if subnet.IP.To4() == nil {
						plan.PublicNet.IPv6 = types.StringValue(subnet.IP.String() + "/" + subnet.Mask)
						break
					}
				}
			}
		}
	}

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state.
func (r *ServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ServerResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Cancel the server if it has been provisioned
	if !state.ServerID.IsNull() {
		serverID := hrobot.ServerID(state.ServerID.ValueInt64())

		// Request immediate cancellation
		cancellation := hrobot.Cancellation{
			ServerID:         serverID,
			CancellationDate: "now",
		}

		err := r.client.Server.RequestCancellation(ctx, cancellation)
		if err != nil {
			// Check if it's a "not found" error - server might already be cancelled
			if hrobot.IsNotFoundError(err) {
				// Server already gone, that's fine
				return
			}
			resp.Diagnostics.AddError(
				"Error cancelling server",
				fmt.Sprintf("Could not cancel server %d: %s", state.ServerID.ValueInt64(), err.Error()),
			)
			return
		}
	}
}

// ImportState imports an existing resource into Terraform.
func (r *ServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID can be either a transaction ID or a server number
	// We'll try to parse it as a number first to determine which one it is
	importID := req.ID

	// Try to get server info first (assume it's a server number)
	serverNum, parseErr := strconv.ParseInt(importID, 10, 64)
	if parseErr == nil {
		serverID := hrobot.ServerID(serverNum)
		server, err := r.client.Server.Get(ctx, serverID)

		if err == nil && server != nil {
			// Successfully found marketplace server - import using server info
			var state ServerResourceModel

			// Set basic server info from the actual server
			state.ServerID = types.Int64Value(int64(server.ServerNumber))
			state.ServerName = types.StringValue(server.ServerName)

			// Set to indicate this was imported directly from server (no transaction)
			state.TransactionID = types.StringValue(fmt.Sprintf("server-%d", server.ServerNumber))
			state.Status = types.StringValue(string(server.Status))
			state.ServerType = types.StringValue(server.Product)
			state.WaitForComplete = types.BoolValue(true)

			// Initialize public_net block
			state.PublicNet = &PublicNetModel{
				IPv4Enabled: types.BoolValue(server.ServerIP != nil),
			}

			// Set IPv4 if available
			if server.ServerIP != nil {
				state.PublicNet.IPv4 = types.StringValue(server.ServerIP.String())
			} else {
				state.PublicNet.IPv4 = types.StringNull()
			}

			// IPv6 is always enabled - fetch from subnets
			if len(server.Subnet) > 0 {
				for _, subnet := range server.Subnet {
					// Look for IPv6 subnet
					if subnet.IP.To4() == nil {
						state.PublicNet.IPv6 = types.StringValue(subnet.IP.String() + "/" + subnet.Mask)
						break
					}
				}
			}

			// Save to state
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}

	// If server lookup failed, try as transaction ID
	transaction, err := r.client.Ordering.GetMarketTransaction(ctx, importID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing server market order",
			fmt.Sprintf("Could not find server or transaction with ID '%s': %s", importID, err.Error()),
		)
		return
	}

	// Import using transaction info
	var state ServerResourceModel
	state.TransactionID = types.StringValue(transaction.ID)
	state.Status = types.StringValue(transaction.Status)

	if transaction.ServerNumber != nil {
		state.ServerID = types.Int64Value(int64(*transaction.ServerNumber))

		// Fetch server name
		serverID := hrobot.ServerID(*transaction.ServerNumber)
		server, err := r.client.Server.Get(ctx, serverID)
		if err == nil && server != nil {
			state.ServerName = types.StringValue(server.ServerName)
		}
	}

	// Set computed/unknown values
	state.ServerType = types.StringValue("auction")
	state.WaitForComplete = types.BoolValue(true)

	// Save to state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
