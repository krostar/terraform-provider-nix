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
	dataSourceStorePath struct {
		nix nix.Nix
	}

	dataSourceStorePathModel struct {
		Installable    types.String `tfsdk:"installable"`
		OutputPath     types.String `tfsdk:"output_path"`
		DerivationPath types.String `tfsdk:"drv_path"`
		Valid          types.Bool   `tfsdk:"valid"`
	}
)

func newDataSourceStorePath() datasource.DataSource { return new(dataSourceStorePath) }

func (*dataSourceStorePath) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_store_path"
}

func (*dataSourceStorePath) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieve data about a nix store path.",
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
			"valid": schema.BoolAttribute{
				MarkdownDescription: "Whenever the derivation output is usable.",
				Computed:            true,
			},
		},
	}
}

func (d *dataSourceStorePath) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	n, ok := req.ProviderData.(nix.Nix)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource configure data type",
			fmt.Sprintf("Expected nix implementation, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.nix = n
}

func (d *dataSourceStorePath) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model dataSourceStorePathModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	valid, storePath, err := d.nix.IsBuilt(ctx, model.Installable.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check if installable is built", err.Error())
		return
	}
	if !valid {
		storePath, err = d.nix.Derivation(ctx, model.Installable.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to evaluate derivation", err.Error())
			return
		}
	}

	model.DerivationPath = types.StringValue(storePath.Derivation)
	model.OutputPath = types.StringValue(storePath.Output)
	model.Valid = types.BoolValue(valid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
