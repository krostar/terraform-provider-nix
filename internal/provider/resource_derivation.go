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
	derivationResource struct {
		nix nix.Nix
	}
	derivationResourceModel struct {
		Installable types.String `tfsdk:"installable"`
		Output      types.String `tfsdk:"output_path"`
		Derivation  types.String `tfsdk:"drv_path"`
	}
)

func newResourceBuild() resource.Resource { return new(derivationResource) }

func (r *derivationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_derivation"
}

func (r *derivationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Build any installable and exposes its derivation and output.",
		Attributes: map[string]schema.Attribute{
			"installable": schema.StringAttribute{
				MarkdownDescription: "Nix installable (store path, nix packages, flake attribute, nix expressions, ...).",
				Required:            true,
			},
			"output_path": schema.StringAttribute{
				MarkdownDescription: "Path to the derivation output.",
				Computed:            true,
			},
			"drv_path": schema.StringAttribute{
				MarkdownDescription: "Path to the derivation file.",
				Computed:            true,
			},
		},
	}
}

func (r *derivationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *derivationResource) buildInstallable(ctx context.Context, model *derivationResourceModel, diags *diag.Diagnostics) {
	if diags.HasError() {
		return
	}

	storePath, err := r.nix.Build(ctx, model.Installable.ValueString())
	if err != nil {
		diags.AddError("Unable to build derivation", err.Error())
		return
	}

	model.Derivation = types.StringValue(storePath.Derivation)
	model.Output = types.StringValue(storePath.Output)
}

func (r *derivationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan derivationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	r.buildInstallable(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *derivationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state derivationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	{ // check if installable exists (and return its store path)
		ok, storePath, err := r.nix.IsBuilt(ctx, state.Installable.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to get build info", err.Error())
			return
		}
		if ok {
			state.Derivation = types.StringValue(storePath.Derivation)
			state.Output = types.StringValue(storePath.Output)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		}
	}

	// otherwise, rebuild it
	r.buildInstallable(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *derivationResource) Update(ctx context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	tflog.Info(ctx, "Update operation is unhandled in this provider")
}

func (r *derivationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state derivationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if err := r.nix.Delete(ctx, state.Output.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete", err.Error())
	}

	resp.Diagnostics.AddWarning(
		"Delete operation may not remove garbage",
		"See https://nixos.org/manual/nix/stable/command-ref/nix-collect-garbage and run nix-collect-garbage if needed",
	)
}
