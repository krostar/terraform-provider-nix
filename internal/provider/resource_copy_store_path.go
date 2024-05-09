package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/krostar/terraform-provider-nix/internal/nix"
)

type (
	copyStorePathResource struct {
		nix nix.Nix
	}
	copyStorePathResourceModel struct {
		StorePath               types.String `tfsdk:"store_path"`
		From                    types.String `tfsdk:"from"`
		To                      types.String `tfsdk:"to"`
		CheckSignature          types.Bool   `tfsdk:"check_sigs"`
		SubstituteOnDestination types.Bool   `tfsdk:"substitute_on_destination"`
		SSHOptions              types.List   `tfsdk:"ssh_options"`
	}
)

func newResourceCopyStorePath() resource.Resource { return new(copyStorePathResource) }

func (*copyStorePathResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_copy_store_path"
}

func (*copyStorePathResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Copies store path closures between two Nix stores.",
		Attributes: map[string]schema.Attribute{
			"store_path": schema.StringAttribute{
				MarkdownDescription: "Store path to copy.",
				Required:            true,
			},
			"from": schema.StringAttribute{
				MarkdownDescription: "URL of the source Nix store (see [nix stores](https://nixos.org/manual/nix/stable/command-ref/new-cli/nix3-help-stores) for possible values).",
				Optional:            true,
			},
			"to": schema.StringAttribute{
				MarkdownDescription: "URL of the destination Nix store (see [nix stores](https://nixos.org/manual/nix/stable/command-ref/new-cli/nix3-help-stores) for possible values).",
				Required:            true,
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

func (r *copyStorePathResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	n, ok := req.ProviderData.(nix.Nix)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource configure data type",
			fmt.Sprintf("Expected nix implementation, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.nix = n
}

func (r *copyStorePathResource) copyInstallable(ctx context.Context, model *copyStorePathResourceModel, diags *diag.Diagnostics) {
	if diags.HasError() {
		return
	}

	var sshOptions []string
	diags.Append(model.SSHOptions.ElementsAs(ctx, &sshOptions, false)...)

	if err := r.nix.Copy(ctx, nix.CopyRequest{
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

func (r *copyStorePathResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan copyStorePathResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	r.copyInstallable(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (*copyStorePathResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {
}

func (*copyStorePathResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}

func (*copyStorePathResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete operation is a no-op for nix provider, use nix-collect-garbage if needed")
}
