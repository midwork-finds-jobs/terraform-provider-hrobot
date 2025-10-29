// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/midwork-finds-jobs/terraform-provider-hrobot/pkg/hrobot"
)

// Ensure the implementation satisfies the resource.Resource interface.
var _ resource.Resource = &VSwitchResource{}
var _ resource.ResourceWithImportState = &VSwitchResource{}

// NewVSwitchResource is a helper function to simplify the provider implementation.
func NewVSwitchResource() resource.Resource {
	return &VSwitchResource{}
}

// VSwitchResource is the resource implementation.
type VSwitchResource struct {
	client *hrobot.Client
}

// VSwitchResourceModel describes the resource data model.
type VSwitchResourceModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	VLAN      types.Int64  `tfsdk:"vlan"`
	Cancelled types.Bool   `tfsdk:"cancelled"`
	Servers   types.Set    `tfsdk:"servers"`
}

// Metadata returns the resource type name.
func (r *VSwitchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vswitch"
}

// Schema defines the schema for the resource.
func (r *VSwitchResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "manages a vswitch in hetzner robot.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "vswitch id (computed)",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the vswitch",
				Required:            true,
			},
			"vlan": schema.Int64Attribute{
				MarkdownDescription: "vlan id for the vswitch",
				Required:            true,
			},
			"cancelled": schema.BoolAttribute{
				MarkdownDescription: "whether the vswitch has been cancelled (computed)",
				Computed:            true,
			},
			"servers": schema.SetAttribute{
				MarkdownDescription: "list of server numbers or IPs to attach to this vswitch",
				ElementType:         types.Int64Type,
				Optional:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *VSwitchResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *VSwitchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VSwitchResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the vSwitch via API
	vswitch, err := r.client.VSwitch.Create(ctx, plan.Name.ValueString(), int(plan.VLAN.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating vswitch",
			fmt.Sprintf("could not create vswitch: %s", err.Error()),
		)
		return
	}

	// Map response to resource model
	plan.ID = types.Int64Value(int64(vswitch.ID))
	plan.Name = types.StringValue(vswitch.Name)
	plan.VLAN = types.Int64Value(int64(vswitch.VLAN))
	plan.Cancelled = types.BoolValue(vswitch.Cancelled)

	// Add servers to the vSwitch if specified
	if !plan.Servers.IsNull() && !plan.Servers.IsUnknown() {
		var serverNumbers []int64
		resp.Diagnostics.Append(plan.Servers.ElementsAs(ctx, &serverNumbers, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(serverNumbers) > 0 {
			// Convert server numbers to strings for the API
			servers := make([]string, len(serverNumbers))
			for i, num := range serverNumbers {
				servers[i] = strconv.FormatInt(num, 10)
			}

			// Add servers to the vSwitch
			err = r.client.VSwitch.AddServers(ctx, vswitch.ID, servers)
			if err != nil {
				resp.Diagnostics.AddError(
					"error adding servers to vswitch",
					fmt.Sprintf("could not add servers to vswitch %d: %s", vswitch.ID, err.Error()),
				)
				return
			}

			// Wait for servers to be added and become ready
			if err := r.client.VSwitch.WaitForVSwitchReady(ctx, vswitch.ID); err != nil {
				resp.Diagnostics.AddError(
					"vswitch not ready after adding servers",
					fmt.Sprintf("vswitch is busy after adding servers: %s", err.Error()),
				)
				return
			}
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *VSwitchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VSwitchResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state from API
	vswitch, err := r.client.VSwitch.Get(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		if hrobot.IsNotFoundError(err) {
			// vSwitch was deleted outside of Terraform
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"error reading vswitch",
			fmt.Sprintf("could not read vswitch %d: %s", state.ID.ValueInt64(), err.Error()),
		)
		return
	}

	// Update state with latest values from API
	state.Name = types.StringValue(vswitch.Name)
	state.VLAN = types.Int64Value(int64(vswitch.VLAN))
	state.Cancelled = types.BoolValue(vswitch.Cancelled)

	// Extract server numbers from the API response
	if len(vswitch.Servers) > 0 {
		serverNumbers := make([]int64, 0, len(vswitch.Servers))
		for _, server := range vswitch.Servers {
			serverNumbers = append(serverNumbers, int64(server.ServerNumber))
		}
		serverSet, diags := types.SetValueFrom(ctx, types.Int64Type, serverNumbers)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Servers = serverSet
	} else {
		// No servers, set to empty set
		serverSet, diags := types.SetValueFrom(ctx, types.Int64Type, []int64{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Servers = serverSet
	}

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state.
func (r *VSwitchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VSwitchResourceModel
	var state VSwitchResourceModel

	// Read Terraform plan and state data into the models
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the vSwitch if name or VLAN changed
	if !plan.Name.Equal(state.Name) || !plan.VLAN.Equal(state.VLAN) {
		err := r.client.VSwitch.Update(ctx, int(state.ID.ValueInt64()), plan.Name.ValueString(), int(plan.VLAN.ValueInt64()))
		if err != nil {
			resp.Diagnostics.AddError(
				"error updating vswitch",
				fmt.Sprintf("could not update vswitch %d: %s", state.ID.ValueInt64(), err.Error()),
			)
			return
		}
	}

	// Update servers if the servers list changed
	if !plan.Servers.Equal(state.Servers) {
		// Wait for vSwitch to be ready before making changes
		if err := r.client.VSwitch.WaitForVSwitchReady(ctx, int(state.ID.ValueInt64())); err != nil {
			resp.Diagnostics.AddError(
				"vswitch not ready",
				fmt.Sprintf("vswitch is busy, could not wait for it to be ready: %s", err.Error()),
			)
			return
		}

		var planServers, stateServers []int64

		// Get planned servers
		if !plan.Servers.IsNull() && !plan.Servers.IsUnknown() {
			resp.Diagnostics.Append(plan.Servers.ElementsAs(ctx, &planServers, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		// Get current state servers
		if !state.Servers.IsNull() && !state.Servers.IsUnknown() {
			resp.Diagnostics.Append(state.Servers.ElementsAs(ctx, &stateServers, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		// Find servers to add and remove
		toAdd := difference(planServers, stateServers)
		toRemove := difference(stateServers, planServers)

		// Remove servers that are no longer in the plan
		if len(toRemove) > 0 {
			removeServers := make([]string, len(toRemove))
			for i, num := range toRemove {
				removeServers[i] = strconv.FormatInt(num, 10)
			}
			err := r.client.VSwitch.RemoveServers(ctx, int(state.ID.ValueInt64()), removeServers)
			if err != nil {
				resp.Diagnostics.AddError(
					"error removing servers from vswitch",
					fmt.Sprintf("could not remove servers from vswitch %d: %s", state.ID.ValueInt64(), err.Error()),
				)
				return
			}

			// Wait for removal to complete
			if err := r.client.VSwitch.WaitForVSwitchReady(ctx, int(state.ID.ValueInt64())); err != nil {
				resp.Diagnostics.AddError(
					"vswitch not ready after removing servers",
					fmt.Sprintf("vswitch is busy after removing servers: %s", err.Error()),
				)
				return
			}
		}

		// Add new servers from the plan
		if len(toAdd) > 0 {
			addServers := make([]string, len(toAdd))
			for i, num := range toAdd {
				addServers[i] = strconv.FormatInt(num, 10)
			}
			err := r.client.VSwitch.AddServers(ctx, int(state.ID.ValueInt64()), addServers)
			if err != nil {
				resp.Diagnostics.AddError(
					"error adding servers to vswitch",
					fmt.Sprintf("could not add servers to vswitch %d: %s", state.ID.ValueInt64(), err.Error()),
				)
				return
			}

			// Wait for addition to complete
			if err := r.client.VSwitch.WaitForVSwitchReady(ctx, int(state.ID.ValueInt64())); err != nil {
				resp.Diagnostics.AddError(
					"vswitch not ready after adding servers",
					fmt.Sprintf("vswitch is busy after adding servers: %s", err.Error()),
				)
				return
			}
		}
	}

	// Read back the updated vSwitch to get the latest state
	vswitch, err := r.client.VSwitch.Get(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"error reading vswitch after update",
			fmt.Sprintf("could not read vswitch %d after update: %s", state.ID.ValueInt64(), err.Error()),
		)
		return
	}

	// Update plan with the latest values (preserve ID from state)
	plan.ID = state.ID
	plan.Name = types.StringValue(vswitch.Name)
	plan.VLAN = types.Int64Value(int64(vswitch.VLAN))
	plan.Cancelled = types.BoolValue(vswitch.Cancelled)

	// Extract server numbers from the API response
	if len(vswitch.Servers) > 0 {
		serverNumbers := make([]int64, 0, len(vswitch.Servers))
		for _, server := range vswitch.Servers {
			serverNumbers = append(serverNumbers, int64(server.ServerNumber))
		}
		serverSet, diags := types.SetValueFrom(ctx, types.Int64Type, serverNumbers)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Servers = serverSet
	} else {
		// No servers, set to empty set
		serverSet, diags := types.SetValueFrom(ctx, types.Int64Type, []int64{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Servers = serverSet
	}

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state.
func (r *VSwitchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VSwitchResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the vSwitch via API (cancel immediately with "now")
	err := r.client.VSwitch.Delete(ctx, int(state.ID.ValueInt64()), "now")
	if err != nil {
		if !hrobot.IsNotFoundError(err) {
			resp.Diagnostics.AddError(
				"error deleting vswitch",
				fmt.Sprintf("could not delete vswitch %d: %s", state.ID.ValueInt64(), err.Error()),
			)
			return
		}
	}
}

// ImportState imports an existing resource into Terraform.
func (r *VSwitchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID as an integer
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"invalid import id",
			fmt.Sprintf("could not parse import id '%s' as integer: %s", req.ID, err.Error()),
		)
		return
	}

	// Set the ID in state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// difference returns elements in a that are not in b.
func difference(a, b []int64) []int64 {
	mb := make(map[int64]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []int64
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}
