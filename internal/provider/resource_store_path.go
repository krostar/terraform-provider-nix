package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/sync/errgroup"

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

	if r.buildInstallable(ctx, &plan, &resp.Diagnostics); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *resourceStorePath) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceStorePathModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	derivation, err := r.nix.DescribeDerivation(ctx, state.Installable.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to describe derivation", err.Error())
		return
	}

	wp, ctx := errgroup.WithContext(ctx)
	drvExists, outputExists := false, false

	wp.Go(func() error {
		var err error
		drvExists, _, err = r.nix.GetStorePath(ctx, derivation.Path.Derivation)
		return err
	})

	wp.Go(func() error {
		var err error
		outputExists, _, err = r.nix.GetStorePath(ctx, derivation.Path.Output)
		return err
	})

	if err := wp.Wait(); err != nil {
		resp.Diagnostics.AddError("Unable to get build info", err.Error())
		return
	}

	if exists := drvExists && outputExists; !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Derivation = types.StringValue(derivation.Path.Derivation)
	state.Output = types.StringValue(derivation.Path.Output)
	state.System = types.StringValue(derivation.System)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceStorePath) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state resourceStorePathModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &state)...)

	if r.buildInstallable(ctx, &state, &resp.Diagnostics); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (*resourceStorePath) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Delete operation is a no-op for the nix provider.",
		"Delete operation may have consequences out of the scope of this plan. Use nix-collect-garbage if needed.",
	)
}
