package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/krostar/terraform-provider-nix/internal/nix"
)

type (
	resourceStorePath      struct{ nix nix.Nix }
	resourceStorePathModel struct {
		Installable types.String `tfsdk:"installable"`
		Output      types.String `tfsdk:"output_path"`
		Derivation  types.String `tfsdk:"drv_path"`
		System      types.String `tfsdk:"system"`
	}
)

func newResourceStorePath() resource.Resource { return new(resourceStorePath) }

func (*resourceStorePath) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_store_path"
}

func (*resourceStorePath) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Build any installable and exposes its store paths.",
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
			"system": schema.StringAttribute{
				MarkdownDescription: "System for which the derivation is built.",
				Computed:            true,
			},
		},
	}
}

func (r *resourceStorePath) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *resourceStorePath) buildInstallable(ctx context.Context, model *resourceStorePathModel, diags *diag.Diagnostics) {
	if diags.HasError() {
		return
	}

	storePath, err := r.nix.Build(ctx, model.Installable.ValueString())
	if err != nil {
		diags.AddError("Unable to build derivation", err.Error())
		return
	}

	derivation, err := r.nix.DescribeDerivation(ctx, storePath.Derivation)
	if err != nil {
		diags.AddError("Unable to describe derivation", err.Error())
		return
	}

	model.Derivation = types.StringValue(storePath.Derivation)
	model.Output = types.StringValue(storePath.Output)
	model.System = types.StringValue(derivation.System)
}

func (r *resourceStorePath) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceStorePathModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	r.buildInstallable(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *resourceStorePath) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceStorePathModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	{ // check if installable exists
		exists, storePath, err := r.nix.GetStorePath(ctx, state.Installable.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to get build info", err.Error())
			return
		}
		if exists {
			derivation, err := r.nix.DescribeDerivation(ctx, storePath.Derivation)
			if err != nil {
				resp.Diagnostics.AddError("Unable to describe derivation", err.Error())
				return
			}

			state.Derivation = types.StringValue(storePath.Derivation)
			state.Output = types.StringValue(storePath.Output)
			state.System = types.StringValue(derivation.System)
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

func (*resourceStorePath) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update should not happen.", "Update does not really make this for this provider, don't know what to do.")
}

func (*resourceStorePath) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Delete operation is no-op for this provider.",
		"Delete operation may have consequences out of the scope of this plan. Use nix-collect-garbage if needed.",
	)
}
