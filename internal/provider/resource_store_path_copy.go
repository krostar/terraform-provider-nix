package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/krostar/terraform-provider-nix/internal/nix"
)

type (
	resourceStorePathCopy      struct{ nix nix.Nix }
	resourceStorePathCopyModel struct {
		StorePath               types.String `tfsdk:"store_path"`
		From                    types.String `tfsdk:"from"`
		To                      types.String `tfsdk:"to"`
		CheckSignature          types.Bool   `tfsdk:"check_sigs"`
		SubstituteOnDestination types.Bool   `tfsdk:"substitute_on_destination"`
		SSHOptions              types.List   `tfsdk:"ssh_options"`
	}
)

func newResourceStorePathCopy() resource.Resource { return new(resourceStorePathCopy) }

func (*resourceStorePathCopy) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_store_path_copy"
}

func (*resourceStorePathCopy) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Copy store path closures between two Nix stores.",
		Attributes: map[string]schema.Attribute{
			"store_path": schema.StringAttribute{
				MarkdownDescription: "Store path to copy.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"from": schema.StringAttribute{
				MarkdownDescription: "URL of the source Nix store (see [nix stores](https://nixos.org/manual/nix/stable/command-ref/new-cli/nix3-help-stores) for possible values).",
				Optional:            true,
			},
			"to": schema.StringAttribute{
				MarkdownDescription: "URL of the destination Nix store (see [nix stores](https://nixos.org/manual/nix/stable/command-ref/new-cli/nix3-help-stores) for possible values).",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"check_sigs": schema.BoolAttribute{
				MarkdownDescription: "Whenever paths should be signed by trusted keys.",
				Optional:            true,
			},
			"substitute_on_destination": schema.BoolAttribute{
				MarkdownDescription: "Whether to try substitutes on the destination store (only supported by SSH stores). This causes the remote machine to try to substitute missing store paths, which may be faster if the link between the local and remote machines is slower than the link between the remote machine and its substitutes.",
				Optional:            true,
			},
			"ssh_options": schema.ListAttribute{
				MarkdownDescription: "SSH connection options (like `-o StrictHostKeyChecking=no`, see [man ssh_config](https://linux.die.net/man/5/ssh_config) for possible values).",
				ElementType:         types.StringType,
				Optional:            true,
			},
		},
	}
}

func (r *resourceStorePathCopy) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	n, ok := req.ProviderData.(nix.Nix)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected configure data type",
			fmt.Sprintf("Expected nix implementation, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.nix = n
}

func (r *resourceStorePathCopy) copyInstallable(ctx context.Context, model *resourceStorePathCopyModel, diags *diag.Diagnostics) {
	if diags.HasError() {
		return
	}

	var sshOptions []string
	diags.Append(model.SSHOptions.ElementsAs(ctx, &sshOptions, false)...)

	if err := r.nix.CopyStorePath(ctx, nix.CopyRequest{
		Installable:             model.StorePath.ValueString(),
		From:                    model.From.ValueStringPointer(),
		To:                      model.To.ValueStringPointer(),
		CheckSignature:          model.CheckSignature.ValueBoolPointer(),
		SubstituteOnDestination: model.SubstituteOnDestination.ValueBoolPointer(),
		SSHOptions:              sshOptions,
	}); err != nil {
		diags.AddError("Unable to copy", err.Error())
		return
	}
}

func (r *resourceStorePathCopy) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceStorePathCopyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if r.copyInstallable(ctx, &plan, &resp.Diagnostics); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *resourceStorePathCopy) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceStorePathCopyModel
	if resp.Diagnostics.Append(req.State.Get(ctx, &state)...); resp.Diagnostics.HasError() {
		return
	}

	var sshOptions []string
	resp.Diagnostics.Append(state.SSHOptions.ElementsAs(ctx, &sshOptions, false)...)

	exists, err := r.nix.RemoteStorePathExists(ctx, nix.RemoteStorePathExistsRequest{
		Installable: state.StorePath.ValueString(),
		Store:       state.To.ValueString(),
		SSHOptions:  sshOptions,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to check if remote store path exists", err.Error())
		return
	}

	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *resourceStorePathCopy) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state resourceStorePathCopyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &state)...)

	if r.copyInstallable(ctx, &state, &resp.Diagnostics); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (*resourceStorePathCopy) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Delete operation is a no-op for the nix provider.",
		"Delete operation may have consequences out of the scope of this plan. Use nix-collect-garbage on the remote store if needed.",
	)
}
