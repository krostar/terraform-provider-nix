resource "nix_store_path" "this" {
  installable = provider::nix::flake_nixos_configuration(path.module, "awesomeHost", "formats.amazon")
}
