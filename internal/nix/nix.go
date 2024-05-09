package nix

import "context"

// Nix exposes ways to interact with nix.
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

// StorePath defines the path on the filesystem, usually on /nix/store, of the derivation and its output.
type StorePath struct {
	Derivation string
	Output     string
}

// CopyRequest is the input parameter provided to the Copy method of the Nix interface.
type CopyRequest struct {
	Installable             string
	From                    *string
	To                      *string
	CheckSignature          *bool
	SubstituteOnDestination *bool
	SSHOptions              []string
}
