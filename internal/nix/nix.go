package nix

import (
	"context"
	"encoding/json"
)

// Nix exposes ways to interact with nix.
type Nix interface {
	// EvaluateExpression evaluate a nix expression.
	EvaluateExpression(ctx context.Context, req EvaluateRequest) (json.RawMessage, error)

	// Build a derivation or fetch a store path.
	Build(ctx context.Context, installable string) (*StorePath, error)

	// DescribeDerivation queries information about a store paths.
	DescribeDerivation(ctx context.Context, installable string) (*Derivation, error)

	// GetStorePath returns a store path and whenever it is valid.
	GetStorePath(ctx context.Context, installable string) (bool, *StorePath, error)

	// CopyStorePath copies store path closures between two Nix stores.
	CopyStorePath(ctx context.Context, req CopyRequest) error

	// RemoteStorePathExists checks whether a nix store path exists.
	RemoteStorePathExists(ctx context.Context, req RemoteStorePathExistsRequest) (bool, error)
}

// StorePath defines the path on the filesystem, usually on /nix/store, of the derivation and its outputs.
type StorePath struct {
	Derivation string
	Output     string
}

// Derivation description.
type Derivation struct {
	Name   string
	Path   StorePath
	System string
}

// EvaluateRequest is the input parameter provided to the EvaluateExpression of the Nix interface.
type EvaluateRequest struct {
	Installable string
	Apply       *string
}

// CopyRequest is the input parameter provided to the CopyStorePath method of the Nix interface.
type CopyRequest struct {
	Installable             string
	From                    *string
	To                      *string
	CheckSignature          *bool
	SubstituteOnDestination *bool
	SSHOptions              []string
}

// RemoteStorePathExistsRequest is the input parameter provided to the RemoteStorePathExists method of the Nix interface.
type RemoteStorePathExistsRequest struct {
	Installable string
	Store       string
	SSHOptions  []string
}
