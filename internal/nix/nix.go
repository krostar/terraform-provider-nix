package nix

import "context"

type Nix interface {
	// IsBuilt checks whenever a store path is valid and returns it.
	IsBuilt(ctx context.Context, installable string) (bool, *StorePath, error)

	// Derivation query information about store paths.
	Derivation(ctx context.Context, installable string) (*StorePath, error)

	// Build a derivation or fetch a store path.
	Build(ctx context.Context, installable string) (*StorePath, error)

	// Copy store path closures between two Nix stores.
	Copy(ctx context.Context, req CopyRequest) error

	// Delete paths from the Nix store.
	Delete(ctx context.Context, installable string) error
}

type StorePath struct {
	Derivation string
	Output     string
}

type CopyRequest struct {
	Installable             string
	From                    *string
	To                      *string
	CheckSignature          *bool
	SubstituteOnDestination *bool
	SSHOptions              []string
}
