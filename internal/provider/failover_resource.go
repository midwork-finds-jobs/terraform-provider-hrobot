// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

// Ensure the implementation satisfies the resource.Resource interface.
var _ resource.Resource = &FailoverResource{}
var _ resource.ResourceWithImportState = &FailoverResource{}

// NewFailoverResource is a helper function to simplify the provider implementation.
func NewFailoverResource() resource.Resource {
	return &FailoverResource{}
}

// FailoverResource is the resource implementation.
type FailoverResource struct {
	client *hrobot.Client
}

// FailoverResourceModel describes the resource data model.
type FailoverResourceModel struct {
	IP             types.String `tfsdk:"ip"`
	Netmask        types.String `tfsdk:"netmask"`
	ServerIP       types.String `tfsdk:"server_ip"`
	ServerIPv6Net  types.String `tfsdk:"server_ipv6_net"`
	ServerNumber   types.Int64  `tfsdk:"server_number"`
	ActiveServerIP types.String `tfsdk:"active_server_ip"`
}

// Metadata returns the resource type name.
func (r *FailoverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_failover_ip"
}

// Schema defines the schema for the resource.
func (r *FailoverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages routing of a failover IP address in Hetzner Robot. Use this resource to route a failover IP to a specific server.",
		Attributes: map[string]schema.Attribute{
			"ip": schema.StringAttribute{
				MarkdownDescription: "Failover IP address (IPv4 or IPv6)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"netmask": schema.StringAttribute{
				MarkdownDescription: "Failover netmask",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"server_ip": schema.StringAttribute{
				MarkdownDescription: "Main IP of the related server",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"server_ipv6_net": schema.StringAttribute{
				MarkdownDescription: "Main IPv6 net of the related server",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"server_number": schema.Int64Attribute{
				MarkdownDescription: "Server ID of the related server",
				Computed:            true,
			},
			"active_server_ip": schema.StringAttribute{
				MarkdownDescription: "Main IP address of the server where the failover IP should be routed to",
				Required:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *FailoverResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the resource and sets the initial Terraform state.
func (r *FailoverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FailoverResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Route the failover IP to the specified server
	failover, err := r.client.Failover.Update(ctx, plan.IP.ValueString(), plan.ActiveServerIP.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"error routing failover ip",
			fmt.Sprintf("could not route failover ip %s to %s: %s", plan.IP.ValueString(), plan.ActiveServerIP.ValueString(), err.Error()),
		)
		return
	}

	// Map response to resource model
	plan.IP = types.StringValue(failover.IP)
	plan.Netmask = types.StringValue(failover.Netmask)
	plan.ServerIP = types.StringValue(failover.ServerIP)
	plan.ServerIPv6Net = types.StringValue(failover.ServerIPv6Net)
	plan.ServerNumber = types.Int64Value(int64(failover.ServerNumber))
	if failover.ActiveServerIP != nil {
		plan.ActiveServerIP = types.StringValue(*failover.ActiveServerIP)
	} else {
		plan.ActiveServerIP = types.StringNull()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *FailoverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FailoverResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state from API
	failover, err := r.client.Failover.Get(ctx, state.IP.ValueString())
	if err != nil {
		if hrobot.IsNotFoundError(err) {
			// Failover IP was deleted outside of Terraform
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"error reading failover ip",
			fmt.Sprintf("could not read failover ip %s: %s", state.IP.ValueString(), err.Error()),
		)
		return
	}

	// Update state with latest values from API
	state.IP = types.StringValue(failover.IP)
	state.Netmask = types.StringValue(failover.Netmask)
	state.ServerIP = types.StringValue(failover.ServerIP)
	state.ServerIPv6Net = types.StringValue(failover.ServerIPv6Net)
	state.ServerNumber = types.Int64Value(int64(failover.ServerNumber))
	if failover.ActiveServerIP != nil {
		state.ActiveServerIP = types.StringValue(*failover.ActiveServerIP)
	} else {
		// If the failover is unrouted (active_server_ip is null), remove the resource
		// because it means the failover routing has been deleted
		resp.State.RemoveResource(ctx)
		return
	}

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state.
func (r *FailoverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FailoverResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the failover IP routing via API
	failover, err := r.client.Failover.Update(ctx, plan.IP.ValueString(), plan.ActiveServerIP.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating failover ip routing",
			fmt.Sprintf("could not update failover ip routing for %s to %s: %s", plan.IP.ValueString(), plan.ActiveServerIP.ValueString(), err.Error()),
		)
		return
	}

	// Update plan with the latest values
	plan.IP = types.StringValue(failover.IP)
	plan.Netmask = types.StringValue(failover.Netmask)
	plan.ServerIP = types.StringValue(failover.ServerIP)
	plan.ServerIPv6Net = types.StringValue(failover.ServerIPv6Net)
	plan.ServerNumber = types.Int64Value(int64(failover.ServerNumber))
	if failover.ActiveServerIP != nil {
		plan.ActiveServerIP = types.StringValue(*failover.ActiveServerIP)
	} else {
		plan.ActiveServerIP = types.StringNull()
	}

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state.
func (r *FailoverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FailoverResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the failover IP routing via API (unroutes the failover)
	err := r.client.Failover.Delete(ctx, state.IP.ValueString())
	if err != nil {
		if !hrobot.IsNotFoundError(err) {
			resp.Diagnostics.AddError(
				"error deleting failover ip routing",
				fmt.Sprintf("could not delete failover ip routing for %s: %s", state.IP.ValueString(), err.Error()),
			)
			return
		}
	}
}

// ImportState imports an existing resource into Terraform.
func (r *FailoverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the failover IP address
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ip"), req.ID)...)
}
