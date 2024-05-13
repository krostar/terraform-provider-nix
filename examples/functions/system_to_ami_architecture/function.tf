resource "nix_store_path" "awesome_host" {
  installable = provider::nix::flake_nixos_configuration(path.module, "awesomeHost", "formats.amazon").installable
}

resource "aws_ami" "awesome_host" {
  name = "awesomeHost"
  // ...
  architecture = provider::nix::system_to_ami_architecture(nix_store_path.awesome_host.system)
}
