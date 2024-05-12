resource "nix_store_path" "this" {
  installable = "${path.module}#nixosConfigurations.awesomeHost.config.formats.amazon"
}

resource "nix_store_path_copy" "this" {
  store_path = nix_store_path.this.output_path
  to         = "ssh-ng://some-remote-host"
}
