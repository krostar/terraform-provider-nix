package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/krostar/terraform-provider-nix/internal/nix"
)

type (
	dataSourceDerivation struct {
		nix nix.Nix
	}

	dataSourceDerivationModel struct {
		Installable    types.String `tfsdk:"installable"`
		OutputPath     types.String `tfsdk:"output_path"`
		DerivationPath types.String `tfsdk:"drv_path"`
		System         types.String `tfsdk:"system"`
	}
)

func newDataSourceDerivation() datasource.DataSource { return new(dataSourceDerivation) }

func (*dataSourceDerivation) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_derivation"
}

func (*dataSourceDerivation) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieve data about a nix derivation.",
		Attributes: map[string]schema.Attribute{
			"installable": schema.StringAttribute{
				MarkdownDescription: "Nix installable (store path, nix packages, flake attribute, nix expressions, ...).",
				Required:            true,
			},
			"output_path": schema.StringAttribute{
				MarkdownDescription: "Path to the derivation build output.",
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

func (d *dataSourceDerivation) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.nix = n
}

func (d *dataSourceDerivation) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model dataSourceDerivationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	derivation, err := d.nix.DescribeDerivation(ctx, model.Installable.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to describe derivation", err.Error())
		return
	}

	model.DerivationPath = types.StringValue(derivation.Path.Derivation)
	model.OutputPath = types.StringValue(derivation.Path.Output)
	model.System = types.StringValue(derivation.System)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
