data "nix_eval" "this" {
  installable = provider::nix::flake_nixos_configuration(path.module, "awesomeHost", "services.openssh.ports").installable
  apply       = "builtins.head"
}
