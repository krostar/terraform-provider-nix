variable "store_path" {
  type = string
}

resource "nix_copy_store_path" "this" {
  store_path = var.store_path
  to         = "ssh-ng://some-remote-host"
}
