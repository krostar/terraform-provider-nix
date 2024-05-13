package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type (
	flakeNixosConfigurationFunction      struct{}
	flakeNixosConfigurationFunctionModel struct {
		Installable   types.String `tfsdk:"installable"`
		Flake         types.String `tfsdk:"flake"`
		Configuration types.String `tfsdk:"configuration"`
		Attribute     types.String `tfsdk:"attribute"`
	}
)

func newFunctionFlakeNixosConfiguration() function.Function {
	return new(flakeNixosConfigurationFunction)
}

func (*flakeNixosConfigurationFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "flake_nixos_configuration"
}

func (*flakeNixosConfigurationFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Construct an installable from a flake path, nixos configuration, and derivation",
		Description: "Returns something like .#nixosConfigurations.awesomeHost.config.system.build.toplevel where . = flake path ; awesomeHost = nixos configuration ; system.build.toplevel = derivation.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "flake",
				Description: "Path or registry identifying the flake",
			},
			function.StringParameter{
				Name:        "configuration",
				Description: "NixOS configuration to use from the flake's nixosConfigurations set.",
			},
			function.StringParameter{
				Name:        "attribute",
				Description: "Configuration attribute to select.",
			},
		},
		Return: function.ObjectReturn{
			AttributeTypes: map[string]attr.Type{
				"installable":   types.StringType,
				"flake":         types.StringType,
				"configuration": types.StringType,
				"attribute":     types.StringType,
			},
		},
	}
}

func (*flakeNixosConfigurationFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var flake, configuration, attribute string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &flake, &configuration, &attribute))

	output := flakeNixosConfigurationFunctionModel{
		Installable:   types.StringValue(fmt.Sprintf("%s#'nixosConfigurations.%q.config.%s'", flake, configuration, attribute)),
		Flake:         types.StringValue(flake),
		Configuration: types.StringValue(configuration),
		Attribute:     types.StringValue(attribute),
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, output))
}
