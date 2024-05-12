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
	dataSourceEval      struct{ nix nix.Nix }
	dataSourceEvalModel struct {
		Installable types.String `tfsdk:"installable"`
		Apply       types.String `tfsdk:"apply"`
		Output      types.String `tfsdk:"output"`
	}
)

func newDataSourceEval() datasource.DataSource { return new(dataSourceEval) }

func (*dataSourceEval) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_eval"
}

func (*dataSourceEval) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Evaluate nix expressions.",
		Attributes: map[string]schema.Attribute{
			"installable": schema.StringAttribute{
				MarkdownDescription: "Nix installable (store path, nix packages, flake attribute, nix expressions, ...).",
				Required:            true,
			},
			"apply": schema.BoolAttribute{
				MarkdownDescription: "Nix function to apply on expression result.",
				Optional:            true,
			},
			"output": schema.StringAttribute{
				MarkdownDescription: "Expression result json encoded",
				Computed:            true,
			},
		},
	}
}

func (d *dataSourceEval) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dataSourceEval) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model dataSourceEvalModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := d.nix.EvaluateExpression(ctx, nix.EvaluateRequest{
		Installable: model.Installable.ValueString(),
		Apply:       model.Apply.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to evaluate expression", err.Error())
		return
	}

	model.Output = types.StringValue(string(raw))
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
