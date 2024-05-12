data "nix_eval" "this" {
  installable = "${flake_nixos_configuration(path.module, "awesomeHost", "formats.amazon")}.config.services.openssh.port"
  apply       = "builtins.head"
}
