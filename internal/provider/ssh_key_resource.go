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
var _ resource.Resource = &SSHKeyResource{}
var _ resource.ResourceWithImportState = &SSHKeyResource{}

// NewSSHKeyResource is a helper function to simplify the provider implementation.
func NewSSHKeyResource() resource.Resource {
	return &SSHKeyResource{}
}

// SSHKeyResource is the resource implementation.
type SSHKeyResource struct {
	client *hrobot.Client
}

// SSHKeyResourceModel describes the resource data model.
type SSHKeyResourceModel struct {
	Fingerprint types.String `tfsdk:"fingerprint"`
	Name        types.String `tfsdk:"name"`
	PublicKey   types.String `tfsdk:"public_key"`
	Type        types.String `tfsdk:"type"`
	Size        types.Int64  `tfsdk:"size"`
}

// Metadata returns the resource type name.
func (r *SSHKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

// Schema defines the schema for the resource.
func (r *SSHKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an SSH public key in Hetzner Robot.",
		Attributes: map[string]schema.Attribute{
			"fingerprint": schema.StringAttribute{
				MarkdownDescription: "SSH key fingerprint (computed)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name for the SSH key",
				Required:            true,
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "SSH public key data in OpenSSH format",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "SSH key type (e.g., RSA, ED25519)",
				Computed:            true,
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "SSH key size in bits",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *SSHKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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
func (r *SSHKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SSHKeyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the SSH key via API
	key, err := r.client.Key.Create(ctx, plan.Name.ValueString(), plan.PublicKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating SSH key",
			fmt.Sprintf("Could not create SSH key: %s", err.Error()),
		)
		return
	}

	// Map response to resource model
	plan.Fingerprint = types.StringValue(key.Fingerprint)
	plan.Name = types.StringValue(key.Name)
	plan.Type = types.StringValue(key.Type)
	plan.Size = types.Int64Value(int64(key.Size))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *SSHKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SSHKeyResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state from API
	key, err := r.client.Key.Get(ctx, state.Fingerprint.ValueString())
	if err != nil {
		if hrobot.IsNotFoundError(err) {
			// SSH key was deleted outside of Terraform
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading SSH key",
			fmt.Sprintf("Could not read SSH key %s: %s", state.Fingerprint.ValueString(), err.Error()),
		)
		return
	}

	// Update state with latest values from API
	state.Name = types.StringValue(key.Name)
	state.Type = types.StringValue(key.Type)
	state.Size = types.Int64Value(int64(key.Size))

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state.
func (r *SSHKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SSHKeyResourceModel
	var state SSHKeyResourceModel

	// Read Terraform plan and state data into the models
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only the name can be updated (public_key requires replacement)
	if !plan.Name.Equal(state.Name) {
		key, err := r.client.Key.Rename(ctx, state.Fingerprint.ValueString(), plan.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating SSH key",
				fmt.Sprintf("Could not update SSH key %s: %s", state.Fingerprint.ValueString(), err.Error()),
			)
			return
		}

		// Update state with new name
		plan.Name = types.StringValue(key.Name)
	}

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state.
func (r *SSHKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SSHKeyResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the SSH key via API
	err := r.client.Key.Delete(ctx, state.Fingerprint.ValueString())
	if err != nil {
		if !hrobot.IsNotFoundError(err) {
			resp.Diagnostics.AddError(
				"Error deleting SSH key",
				fmt.Sprintf("Could not delete SSH key %s: %s", state.Fingerprint.ValueString(), err.Error()),
			)
			return
		}
	}
}

// ImportState imports an existing resource into Terraform.
func (r *SSHKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Use the fingerprint as the import ID
	resource.ImportStatePassthroughID(ctx, path.Root("fingerprint"), req, resp)
}
