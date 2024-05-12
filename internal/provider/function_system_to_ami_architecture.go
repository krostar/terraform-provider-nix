package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

type systemToAMIArchitectureFunction struct {
	nixArchitectureToAMIArchitecture map[string]string
}

func newFunctionSystemToAMIArchitecture() function.Function {
	return &systemToAMIArchitectureFunction{
		// $(nix eval nixpkgs#lib.systems.doubles.all) for all nix systems
		// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/finding-an-ami.html -> 32-bit (i386), 64-bit (x86_64), or 64-bit ARM (arm64)
		nixArchitectureToAMIArchitecture: map[string]string{
			"aarch64": "arm64",
			"x86_64":  "x86_64",
			"i686":    "i386",
		},
	}
}

func (*systemToAMIArchitectureFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "system_to_ami_architecture"
}

func (*systemToAMIArchitectureFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Maps a nix system to a AMI architecture.",
		Description: "Returns an architecture usable in AMI configuration, corresponding to the system a nix derivation is built for.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "system",
				Description: "System from a nix derivation.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *systemToAMIArchitectureFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var nixSystem string
	if resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &nixSystem)); resp.Error != nil {
		return
	}

	nixArchitecture := strings.SplitN(nixSystem, "-", 2)[0]
	amiArchitecture, ok := f.nixArchitectureToAMIArchitecture[nixArchitecture]
	if !ok {
		resp.Error = function.NewFuncError(fmt.Sprintf("Unable to map nix architecture %s to an AMI architecture", nixArchitecture))
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, amiArchitecture))
}
