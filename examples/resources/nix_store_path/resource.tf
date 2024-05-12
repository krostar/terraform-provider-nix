resource "nix_store_path" "this" {
  installable = "${path.module}#nixosConfigurations.awesomeHost.config.formats.amazon"
}
