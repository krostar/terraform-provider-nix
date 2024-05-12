package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

type flakeNixosConfigurationFunction struct{}

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
				Name:        "path",
				Description: "Directory to the flake.nix file.",
			},
			function.StringParameter{
				Name:        "configuration",
				Description: "NixOS configuration to use in the flake's nixosConfigurations output.",
			},
			function.StringParameter{
				Name:        "derivation",
				Description: "Derivation to build.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (*flakeNixosConfigurationFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var flakePath, configuration, derivation string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &flakePath, &configuration, &derivation))
	installable := fmt.Sprintf("%s#'nixosConfigurations.%q.config.%s'", flakePath, configuration, derivation)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, installable))
}
