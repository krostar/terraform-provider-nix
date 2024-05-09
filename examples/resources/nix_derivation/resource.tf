resource "nix_derivation" "this" {
  installable = "${path.module}#nixosConfigurations.awesomeHost.config.formats.amazon"
}