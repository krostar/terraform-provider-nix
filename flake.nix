{
  inputs = {
    nixpkgs-unstable.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = {nixpkgs-unstable, ...}: let
    supportedSystems = ["aarch64-linux" "aarch64-darwin" "x86_64-linux" "x86_64-darwin"];
    forEachSupportedSystems = f: builtins.listToAttrs (builtins.map (system: (nixpkgs-unstable.lib.nameValuePair system (f system))) supportedSystems);
    pkgsForSystem = system: {
      nixpkgs ? nixpkgs-unstable,
      overlays ? [],
    }: (import nixpkgs {
      inherit system overlays;
      config.allowUnfree = true; # terraform requires this
    });
  in {
    devShells = forEachSupportedSystems (system: let
      pkgs = pkgsForSystem system {
        overlays = [
          (final: prev: {
            go_1_22 = prev.go_1_22.overrideAttrs (old: rec {
              version = "1.22.3";
              src = prev.fetchurl {
                url = "https://go.dev/dl/go${version}.src.tar.gz";
                hash = "sha256-gGSO80+QMZPXKlnA3/AZ9fmK4MmqE63gsOy/+ZGnb2g=";
              };
            });
          })
        ];
      };
    in {
      default = pkgs.mkShell {
        nativeBuildInputs = with pkgs; [
          act
          alejandra
          deadnix
          gci
          git
          go_1_22
          gofumpt
          golangci-lint
          gotools
          govulncheck
          shellcheck
          shfmt
          statix
          terraform
          terraform-docs
          tflint
          yamllint
        ];
      };
    });
    formatter = forEachSupportedSystems (system: let pkgs = pkgsForSystem system {}; in pkgs.alejandra);
  };
}
