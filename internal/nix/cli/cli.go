package nixcli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/krostar/terraform-provider-nix/internal/nix"
)

// CLI is a nix implementation using the nix command line interface.
type CLI struct{}

// New creates a new nix implementation backed by the nix command line interface.
func New() *CLI {
	return &CLI{}
}

func (CLI) runNixCmd(ctx context.Context, additionalEnv []string, subcommand string, args ...string) (io.Reader, error) {
	command := strings.Join(append([]string{
		"nix",
		subcommand,
		"--no-update-lock-file",
		"--no-write-lock-file",
	}, args...), " ")
	cmd := exec.CommandContext(ctx, "bash", "-c", command) //nolint: gosec // even if some commands uses variables from callers, it is only the nix cli arguments, not the command executed.
	cmd.Env = append(os.Environ(), additionalEnv...)

	var stdOut bytes.Buffer
	cmd.Stdout = &stdOut
	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("unable to execute command %q: %v (stderr = %s)", err, command, stdErr.String())
	}

	return &stdOut, nil
}

// EvaluateExpression evaluates nix expression and returns a json message.
func (s CLI) EvaluateExpression(ctx context.Context, req nix.EvaluateRequest) (json.RawMessage, error) {
	args := []string{req.Installable, "--json"}
	if req.Apply != nil {
		args = append(args, "--apply "+*req.Apply)
	}

	raw, err := s.runNixCmd(ctx, nil, "eval", args...)
	if err != nil {
		return nil, err
	}

	var msg json.RawMessage
	if err := json.NewDecoder(raw).Decode(&msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// Build builds a nix installable using "nix build", and returns the associated store path.
func (s CLI) Build(ctx context.Context, installable string) (*nix.StorePath, error) {
	stdout, err := s.runNixCmd(ctx, nil, "build", "--no-link", "--json", installable)
	if err != nil {
		return nil, err
	}

	var derivation cmdBuildDerivationOutputDerivation
	{
		var derivations cmdBuildDerivationOutput
		switch err := json.NewDecoder(stdout).Decode(&derivations); {
		case err == nil && len(derivations) == 1:
			for _, d := range derivations {
				derivation = d
			}
		case err == nil && len(derivations) == 0:
			return nil, fmt.Errorf("no derivation built for installable %q", installable)
		case err == nil && len(derivations) > 1:
			return nil, fmt.Errorf("unhandled: found more than one derivation for installable %q", installable)
		default:
			return nil, fmt.Errorf("unable to decode shell to stdout: %v", err)
		}
	}

	return &nix.StorePath{
		Derivation: derivation.DrvPath,
		Output:     derivation.outputPath(),
	}, nil
}

// DescribeDerivation returns the derivation store path based on an installable, using "nix derivation show" command.
func (s CLI) DescribeDerivation(ctx context.Context, installable string) (*nix.Derivation, error) {
	stdout, err := s.runNixCmd(ctx, nil, "derivation show", installable)
	if err != nil {
		return nil, err
	}

	var (
		derivation     cmdDerivationShowOutputDerivation
		derivationPath string
	)
	{
		var derivations cmdDerivationShowOutput
		switch err := json.NewDecoder(stdout).Decode(&derivations); {
		case err == nil && len(derivations) == 1:
			for key, value := range derivations {
				derivationPath = key
				derivation = value
			}

		case err == nil && len(derivations) == 0:
			return nil, fmt.Errorf("no store path provided for installable %q", installable)
		case err == nil && len(derivations) > 1:
			return nil, fmt.Errorf("found more than one store paths for installable %q", installable)
		default:
			return nil, fmt.Errorf("unable to decode command output: %v", err)
		}
	}

	return &nix.Derivation{
		Name: derivation.Name,
		Path: nix.StorePath{
			Derivation: derivationPath,
			Output:     derivation.outputPath(),
		},
		System: derivation.System,
	}, nil
}

// GetStorePath uses the "nix path-info" command to check nix store paths.
// It returns true if the store paths are built.
func (s CLI) GetStorePath(ctx context.Context, installable string) (bool, *nix.StorePath, error) {
	stdout, err := s.runNixCmd(ctx, nil, "path-info", "--json", installable)
	if err != nil {
		return false, nil, err
	}

	var storePath cmdPathInfoOutputStorePath
	{
		var pathInfo cmdPathInfoOutput
		switch err := json.NewDecoder(stdout).Decode(&pathInfo); {
		case err == nil && len(pathInfo) == 1:
			storePath = pathInfo[0]
		case err == nil && len(pathInfo) == 0:
			return false, nil, fmt.Errorf("no store path provided for installable %q", installable)
		case err == nil && len(pathInfo) > 1:
			return false, nil, fmt.Errorf("found more than one store paths for installable %q", installable)
		default:
			return false, nil, fmt.Errorf("unable to decode command output: %v", err)
		}
	}

	return storePath.Valid, &nix.StorePath{
		Derivation: storePath.Deriver,
		Output:     storePath.Path,
	}, nil
}

// CopyStorePath copies the provided nix store path from the provided nix store to the provided nix store.
func (s CLI) CopyStorePath(ctx context.Context, req nix.CopyRequest) error {
	args := []string{req.Installable}
	if req.From != nil {
		args = append(args, "--from "+*req.From)
	}
	if req.To != nil {
		args = append(args, "--to "+*req.To)
	}
	if req.CheckSignature != nil && !*req.CheckSignature {
		args = append(args, "--no-check-sigs")
	}
	if req.SubstituteOnDestination != nil && *req.SubstituteOnDestination {
		args = append(args, "--substitute-on-destination")
	}

	var env []string
	if len(req.SSHOptions) > 0 {
		env = []string{"NIX_SSHOPTS=" + strings.Join(req.SSHOptions, " ")}
	}

	_, err := s.runNixCmd(ctx, env, "copy", args...)
	return err
}
