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
var _ resource.Resource = &RDNSResource{}
var _ resource.ResourceWithImportState = &RDNSResource{}

// NewRDNSResource is a helper function to simplify the provider implementation.
func NewRDNSResource() resource.Resource {
	return &RDNSResource{}
}

// RDNSResource is the resource implementation.
type RDNSResource struct {
	client *hrobot.Client
}

// RDNSResourceModel describes the resource data model.
type RDNSResourceModel struct {
	IP  types.String `tfsdk:"ip"`
	PTR types.String `tfsdk:"ptr"`
}

// Metadata returns the resource type name.
func (r *RDNSResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rdns"
}

// Schema defines the schema for the resource.
func (r *RDNSResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a reverse DNS (PTR) record for an IP address in Hetzner Robot.",
		Attributes: map[string]schema.Attribute{
			"ip": schema.StringAttribute{
				MarkdownDescription: "IP address for the reverse DNS entry",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ptr": schema.StringAttribute{
				MarkdownDescription: "PTR record (hostname) for the IP address",
				Required:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *RDNSResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *RDNSResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RDNSResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the reverse DNS entry via API
	// Use Update method which works for both create and update
	entry, err := r.client.RDNS.Update(ctx, plan.IP.ValueString(), plan.PTR.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating reverse dns entry",
			fmt.Sprintf("could not create reverse dns entry for %s: %s", plan.IP.ValueString(), err.Error()),
		)
		return
	}

	// Map response to resource model
	plan.IP = types.StringValue(entry.IP)
	plan.PTR = types.StringValue(entry.PTR)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *RDNSResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RDNSResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state from API
	entry, err := r.client.RDNS.Get(ctx, state.IP.ValueString())
	if err != nil {
		if hrobot.IsNotFoundError(err) {
			// Reverse DNS entry was deleted outside of Terraform
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"error reading reverse dns entry",
			fmt.Sprintf("could not read reverse dns entry for %s: %s", state.IP.ValueString(), err.Error()),
		)
		return
	}

	// Update state with latest values from API
	state.IP = types.StringValue(entry.IP)
	state.PTR = types.StringValue(entry.PTR)

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state.
func (r *RDNSResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RDNSResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the reverse DNS entry via API
	entry, err := r.client.RDNS.Update(ctx, plan.IP.ValueString(), plan.PTR.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating reverse dns entry",
			fmt.Sprintf("could not update reverse dns entry for %s: %s", plan.IP.ValueString(), err.Error()),
		)
		return
	}

	// Update plan with the latest values
	plan.IP = types.StringValue(entry.IP)
	plan.PTR = types.StringValue(entry.PTR)

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state.
func (r *RDNSResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RDNSResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the reverse DNS entry via API
	err := r.client.RDNS.Delete(ctx, state.IP.ValueString())
	if err != nil {
		if !hrobot.IsNotFoundError(err) {
			resp.Diagnostics.AddError(
				"error deleting reverse dns entry",
				fmt.Sprintf("could not delete reverse dns entry for %s: %s", state.IP.ValueString(), err.Error()),
			)
			return
		}
	}
}

// ImportState imports an existing resource into Terraform.
func (r *RDNSResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the IP address
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ip"), req.ID)...)
}
