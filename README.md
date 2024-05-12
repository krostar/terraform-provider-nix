# Terraform Provider NixFlake

**This is a proof of concept, nothing is properly tested, things could change in a non-compatible way.**

## FAQ

### What does this provider provide ?

This module exposes two resources:

- `nix_store_path`: build a nix installable and get built store paths
- `nix_store_path_copy`: perform a copy a of nix store path from one store to another

two data sources:

- `nix_derivation`: retrieve nix derivation information
- `nix_eval`: retrieve value from nix

two functions:

- `derivation_system_to_ami_architecture`: maps nix system to ami architecture
- `flake_nixos_configuration`: construct a flake based nixos configuration installable name 

### How can I use this provider ?

The following example uses flakes, but you don't have to.

#### flake

For demonstration purpose:
- `mkdir demo && cd demo` -> create a demo directory 
- `git init` initialize an empty git repository
-  `git add` a file named `flake.nix` with the following content:

```nix
# ./flake.nix
{
  inputs = {
    nixos.url = "github:NixOS/nixpkgs/nixos-unstable";
    nixos-generators = {
      url = "github:nix-community/nixos-generators";
      inputs.nixpkgs.follows = "nixos";
    };
  };

  outputs = {
    nixos,
    nixos-generators,
    ...
  }: {
    nixosConfigurations = let
      modules = [
        nixos-generators.nixosModules.all-formats
        {
          formatConfigs.amazon.amazonImage.sizeMB = 4 * 1024;
          system.stateVersion = "24.05";
          nixpkgs.hostPlatform = "aarch64-linux";
        }
      ];
    in {
      awesomeHost = nixos.lib.nixosSystem {
        modules = [{services.nginx.enable = true;}] ++ modules;
      };
      anotherHost = nixos.lib.nixosSystem {inherit modules;};
    };
  };
}
```

This nix flake exposes two output attributes which are super simples nixosConfiguration, with awesomeHost containing a default nginx server.
```sh
$ nix flake show
terraform-provider-nix-demo
└───nixosConfigurations
    └───awesomeHost: NixOS configuration
    └───anotherHost: NixOS configuration
```

and this flake attributes can be built in different flavors, like `nixosConfigurations.awesomeHost.config.formats.amazon`
which can be used to generate a vhd file, used to create amazon ami.

That's it for the nix/flake part, we could build this derivation manually, but let's ask terraform to do it for us.

#### terraform

In the same directory, create a `main.tf` file with the following content:

```terraform
# ./main.tf
terraform {
  required_providers {
    nix = {
      source = "krostar/nix"
    }
  }
}

provider "nix" {}

resource "nix_store_path" "awesome_host" {
  installable = flake_nixos_configuration(path.module, "awesomeHost", "formats.amazon")
}

data "nix_derivation" "awesome_host" {
  installable = nix_store_path.awesome_host.drv_path
}

data "nix_derivation" "another_host" {
  installable = flake_nixos_configuration(path.module, "anotherHost", "formats.amazon")
}

output "from_resource" {
  value = nix_store_path.awesome_host
}

output "from_data_another_host" {
  value = data.nix_derivation.another_host
}

output "from_data_awesome_host" {
  value = data.nix_derivation.awesome_host
}
```

In this file:
- the `provider "nix"` terraform provider is defined and configured, it will be used to glue nix and terraform together
- the `data "nix_derivation"` allow us to retrieve some information about nix derivation
- the `resource "nix_store_path" "awesome_host"` allow us to build the derivation, and retrieve information about it
- and then output all data and resource information

#### Actions!

Run the following commands to initialize the project:

```sh
$ nix flake lock
$ terraform init
```

You should now be able to ask terraform for a plan:

```sh
$ terraform plan
data.nix_derivation.another_host: Reading...
data.nix_derivation.another_host: Still reading... [10s elapsed]
data.nix_derivation.another_host: Still reading... [20s elapsed]
data.nix_derivation.another_host: Read complete after 28s

Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
  + create
 <= read (data resources)

Terraform will perform the following actions:

  # data.nix_derivation.awesome_host will be read during apply
  # (config refers to values not yet known)
 <= data "nix_derivation" "awesome_host" {
      + drv_path    = (known after apply)
      + installable = (known after apply)
      + output_path = (known after apply)
      + system      = (known after apply)
    }

  # nix_store_path.awesome_host will be created
  + resource "nix_store_path" "awesome_host" {
      + drv_path    = (known after apply)
      + installable = ".#'nixosConfigurations.\"awesomeHost\".config.formats.amazon'"
      + output_path = (known after apply)
      + system      = (known after apply)
    }

Plan: 1 to add, 0 to change, 0 to destroy.

Changes to Outputs:
  + from_data_another_host = {
      + drv_path    = "/nix/store/zkkcwad2dcm9zl45q4va1fi8bsfmzi2m-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv"
      + installable = ".#'nixosConfigurations.\"anotherHost\".config.formats.amazon'"
      + output_path = "/nix/store/nwrdplz0mzyi3fzlndvf5ixmn0s9jf1m-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd"
      + system      = "aarch64-linux"
    }
  + from_data_awesome_host = {
      + drv_path    = (known after apply)
      + installable = (known after apply)
      + output_path = (known after apply)
      + system      = (known after apply)
    }
  + from_resource          = {
      + drv_path    = (known after apply)
      + installable = ".#'nixosConfigurations.\"awesomeHost\".config.formats.amazon'"
      + output_path = (known after apply)
      + system      = (known after apply)
    }

───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

Note: You didn't use the -out option to save this plan, so Terraform can't guarantee to take exactly these actions if you run "terraform apply" now.
```

here you see that the `resource "nix_store_path" "awesome_host"` will be created (installable will be built) if applied, and associated data are yet to be known.
Meanwhile, the `data "nix_derivation" "another_host"` is already capable of giving the derivation path (`drv_path`) and build output path `output_path`, without building the installable.
This data is already accessible without building the derivation because nix derivation path and output path are computed based on nix inputs, see [how nix store path works](https://nixos.org/guides/nix-pills/18-nix-store-paths.html).
Because the derivation is not built, the nix store do not contain the derivation nor the build output.

This is the main difference between `nix_derivation` **data** and `nix_store_path` **resource**. The former ask nix for the derivation information
(which evaluate the nix expression behind the flake installable, but does not build it), and the latter that builds it.

Let's apply this plan:

```sh
$ terraform apply
data.nix_derivation.another_host: Reading...
data.nix_derivation.another_host: Still reading... [10s elapsed]
data.nix_derivation.another_host: Still reading... [20s elapsed]
data.nix_derivation.another_host: Read complete after 26s

Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
  + create
 <= read (data resources)

Terraform will perform the following actions:

  # data.nix_derivation.awesome_host will be read during apply
  # (config refers to values not yet known)
 <= data "nix_derivation" "awesome_host" {
      + drv_path    = (known after apply)
      + installable = (known after apply)
      + output_path = (known after apply)
      + system      = (known after apply)
    }

  # nix_store_path.awesome_host will be created
  + resource "nix_store_path" "awesome_host" {
      + drv_path    = (known after apply)
      + installable = ".#'nixosConfigurations.\"awesomeHost\".config.formats.amazon'"
      + output_path = (known after apply)
      + system      = (known after apply)
    }

Plan: 1 to add, 0 to change, 0 to destroy.

Changes to Outputs:
  ~ from_data_another_host = {
      ~ drv_path    = "/nix/store/jqjw6gm06aifn20qa5fgbhbpkv97ps3k-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv" -> "/nix/store/zkkcwad2dcm9zl45q4va1fi8bsfmzi2m-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv"
      ~ output_path = "/nix/store/pri55fj97v6mm8p1x3wnz6lxbxkdvy63-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd" -> "/nix/store/nwrdplz0mzyi3fzlndvf5ixmn0s9jf1m-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd"
        # (2 unchanged attributes hidden)
    }
  + from_data_awesome_host = {
      + drv_path    = (known after apply)
      + installable = (known after apply)
      + output_path = (known after apply)
      + system      = (known after apply)
    }
  + from_resource          = {
      + drv_path    = (known after apply)
      + installable = ".#'nixosConfigurations.\"awesomeHost\".config.formats.amazon'"
      + output_path = (known after apply)
      + system      = (known after apply)
    }

Do you want to perform these actions?
  Terraform will perform the actions described above.
  Only 'yes' will be accepted to approve.

  Enter a value: yes

nix_store_path.awesome_host: Creating...
nix_store_path.awesome_host: Still creating... [10s elapsed]
nix_store_path.awesome_host: Still creating... [20s elapsed]
nix_store_path.awesome_host: Still creating... [30s elapsed]
nix_store_path.awesome_host: Still creating... [40s elapsed]
nix_store_path.awesome_host: Still creating... [50s elapsed]
nix_store_path.awesome_host: Still creating... [1m0s elapsed]
nix_store_path.awesome_host: Still creating... [1m10s elapsed]
nix_store_path.awesome_host: Still creating... [1m20s elapsed]
nix_store_path.awesome_host: Still creating... [1m30s elapsed]
nix_store_path.awesome_host: Still creating... [1m40s elapsed]
nix_store_path.awesome_host: Still creating... [1m50s elapsed]
nix_store_path.awesome_host: Still creating... [2m0s elapsed]
nix_store_path.awesome_host: Still creating... [2m10s elapsed]
nix_store_path.awesome_host: Still creating... [2m20s elapsed]
nix_store_path.awesome_host: Still creating... [2m30s elapsed]
nix_store_path.awesome_host: Still creating... [2m40s elapsed]
nix_store_path.awesome_host: Still creating... [2m50s elapsed]
nix_store_path.awesome_host: Still creating... [3m0s elapsed]
nix_store_path.awesome_host: Still creating... [3m10s elapsed]
nix_store_path.awesome_host: Still creating... [3m20s elapsed]
nix_store_path.awesome_host: Still creating... [3m30s elapsed]
nix_store_path.awesome_host: Creation complete after 3m36s
data.nix_derivation.awesome_host: Reading...
data.nix_derivation.awesome_host: Read complete after 0s

Apply complete! Resources: 1 added, 0 changed, 0 destroyed.

Outputs:

from_data_another_host = {
  "drv_path" = "/nix/store/zkkcwad2dcm9zl45q4va1fi8bsfmzi2m-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv"
  "installable" = ".#'nixosConfigurations.\"anotherHost\".config.formats.amazon'"
  "output_path" = "/nix/store/nwrdplz0mzyi3fzlndvf5ixmn0s9jf1m-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd"
  "system" = "aarch64-linux"
}
from_data_awesome_host = {
  "drv_path" = "/nix/store/g1y6hxdqg0gj906x9hljwdji3mb77vcd-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv"
  "installable" = "/nix/store/g1y6hxdqg0gj906x9hljwdji3mb77vcd-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv"
  "output_path" = "/nix/store/g9yfnibjgh633iw4d226cgzs8z2q30yh-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd"
  "system" = "aarch64-linux"
}
from_resource = {
  "drv_path" = "/nix/store/g1y6hxdqg0gj906x9hljwdji3mb77vcd-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv"
  "installable" = ".#'nixosConfigurations.\"awesomeHost\".config.formats.amazon'"
  "output_path" = "/nix/store/g9yfnibjgh633iw4d226cgzs8z2q30yh-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd"
  "system" = "aarch64-linux"
}
```

As expected, `drv_path` and `output_path` for `from_data_awesome_host` and `from_resource` are equal, we are indeed building the same derivation.
The real difference is that because the derivation is actually built, we can provide other modules with its output.

Note: If we manually build the `from_data_another_host` derivation via nix build, the store path will also exists.
The issue is that it's up to the nix store to keep this store path alive, if garbage collection happens between nix build and terraform apply then the store path becomes invalid.
That is not the case with the resource definition, if the store path is missing, it will be built again.

If we run terraform apply again:
```sh
$ terraform apply
nix_store_path.awesome_host: Refreshing state...
data.nix_derivation.another_host: Reading...
data.nix_derivation.another_host: Read complete after 0s
data.nix_derivation.awesome_host: Reading...
data.nix_derivation.awesome_host: Read complete after 0s

No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration and found no differences, so no changes are needed.

Apply complete! Resources: 0 added, 0 changed, 0 destroyed.

Outputs:

from_data_another_host = {
  "drv_path" = "/nix/store/zkkcwad2dcm9zl45q4va1fi8bsfmzi2m-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv"
  "installable" = ".#'nixosConfigurations.\"anotherHost\".config.formats.amazon'"
  "output_path" = "/nix/store/nwrdplz0mzyi3fzlndvf5ixmn0s9jf1m-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd"
  "system" = "aarch64-linux"
}
from_data_awesome_host = {
  "drv_path" = "/nix/store/g1y6hxdqg0gj906x9hljwdji3mb77vcd-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv"
  "installable" = "/nix/store/g1y6hxdqg0gj906x9hljwdji3mb77vcd-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv"
  "output_path" = "/nix/store/g9yfnibjgh633iw4d226cgzs8z2q30yh-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd"
  "system" = "aarch64-linux"
}
from_resource = {
  "drv_path" = "/nix/store/g1y6hxdqg0gj906x9hljwdji3mb77vcd-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv"
  "installable" = ".#'nixosConfigurations.\"awesomeHost\".config.formats.amazon'"
  "output_path" = "/nix/store/g9yfnibjgh633iw4d226cgzs8z2q30yh-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd"
  "system" = "aarch64-linux"
}
```

As expected, nothing changed so terraform has nothing to do.

Lets now perform a terraform destroy:

```sh
$ terraform destroy
nix_store_path.awesome_host: Refreshing state...
data.nix_derivation.another_host: Reading...
data.nix_derivation.another_host: Read complete after 0s
data.nix_derivation.awesome_host: Reading...
data.nix_derivation.awesome_host: Read complete after 1s

Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
  - destroy

Terraform will perform the following actions:

  # nix_store_path.awesome_host will be destroyed
  - resource "nix_store_path" "awesome_host" {
      - drv_path    = "/nix/store/g1y6hxdqg0gj906x9hljwdji3mb77vcd-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv" -> null
      - installable = ".#'nixosConfigurations.\"awesomeHost\".config.formats.amazon'" -> null
      - output_path = "/nix/store/g9yfnibjgh633iw4d226cgzs8z2q30yh-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd" -> null
      - system      = "aarch64-linux" -> null
    }

Plan: 0 to add, 0 to change, 1 to destroy.

Changes to Outputs:
  - from_data_another_host = {
      - drv_path    = "/nix/store/zkkcwad2dcm9zl45q4va1fi8bsfmzi2m-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv"
      - installable = ".#'nixosConfigurations.\"anotherHost\".config.formats.amazon'"
      - output_path = "/nix/store/nwrdplz0mzyi3fzlndvf5ixmn0s9jf1m-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd"
      - system      = "aarch64-linux"
    } -> null
  - from_data_awesome_host = {
      - drv_path    = "/nix/store/g1y6hxdqg0gj906x9hljwdji3mb77vcd-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv"
      - installable = "/nix/store/g1y6hxdqg0gj906x9hljwdji3mb77vcd-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv"
      - output_path = "/nix/store/g9yfnibjgh633iw4d226cgzs8z2q30yh-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd"
      - system      = "aarch64-linux"
    } -> null
  - from_resource          = {
      - drv_path    = "/nix/store/g1y6hxdqg0gj906x9hljwdji3mb77vcd-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv"
      - installable = ".#'nixosConfigurations.\"awesomeHost\".config.formats.amazon'"
      - output_path = "/nix/store/g9yfnibjgh633iw4d226cgzs8z2q30yh-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd"
      - system      = "aarch64-linux"
    } -> null

Do you really want to destroy all resources?
  Terraform will destroy all your managed infrastructure, as shown above.
  There is no undo. Only 'yes' will be accepted to confirm.

  Enter a value: yes

nix_store_path.awesome_host: Destroying...
nix_store_path.awesome_host: Destruction complete after 0s
╷
│ Warning: Delete operation is no-op for this provider.
│
│ Delete operation may have consequences out of the scope of this plan. Use nix-collect-garbage if needed.
╵

Destroy complete! Resources: 1 destroyed.
```

Nice!
You may have seen this notice above:
```
╷
│ Warning: Delete operation is no-op for this provider.
│
│ Delete operation may have consequences out of the scope of this plan. Use nix-collect-garbage if needed.
╵
```

As stated, cleaning store-paths may have consequences outside of terraform.
Cleaning garbage (unused built store-path and unused build dependencies) is out of scope of this terraform provider.
Consider running nix-collect-garbage manually, or set nix to automatically clean garbage when needed.

### How does it work ?

This provider executes nix commands via the `os/shell` package, through bash.
This means you need to have `bash` and `nix` in your `PATH` to run it.

For reproducibility reasons, consider providing bash, terraform, and nix through a nix shell.

Here are all the commands ran by this provider:
```sh
nix build
nix copy
nix derivation show
nix eval
nix path-info
```

### "nix_store_path: Refreshing state..." is super slow!

`resource "nix_store_path"` creation requires to actually call `nix build` with the provided installable.

After creation (when applying an already created resource), the provider checks with `nix path-info` if the store path is still valid:
- if it is, it returns the derivation description already built
- if it's not, it `nix build` the installable again

This is required to ensure the output exists, even in case of nix garbage collection,
this also means the store path can change due to modification outside terraform (like changing something nix-side),
see *Note: Objects have changed outside of Terraform* below.

Once an installable is built, considering nothing changed nix-side, rebuilding the same derivation should be near instant.

`data "nix_derivation"` requires to evaluate the nix expressions to evaluate nix code to build the derivation.

### Note: Objects have changed outside of Terraform

Once a terraform plan has been applied, changing anything nix-side imply that the derivation path will change and
terraform will notice it.

```sh
$ terraform plan
nix_store_path.awesome_host: Refreshing state...
data.nix_derivation.another_host: Reading...
data.nix_derivation.another_host: Read complete after 0s
data.nix_derivation.awesome_host: Reading...
data.nix_derivation.awesome_host: Read complete after 0s

Changes to Outputs:
  ~ from_data_another_host = {
      ~ drv_path    = "/nix/store/zkkcwad2dcm9zl45q4va1fi8bsfmzi2m-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv" -> "/nix/store/g1y6hxdqg0gj906x9hljwdji3mb77vcd-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd.drv"
      ~ output_path = "/nix/store/nwrdplz0mzyi3fzlndvf5ixmn0s9jf1m-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd" -> "/nix/store/g9yfnibjgh633iw4d226cgzs8z2q30yh-nixos-amazon-image-24.05.20240511.062ca2a-aarch64-linux.vhd"
        # (2 unchanged attributes hidden)
    }

You can apply this plan to save these new output values to the Terraform state, without changing any real infrastructure.

───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

Note: You didn't use the -out option to save this plan, so Terraform can't guarantee to take exactly these actions if you run "terraform apply" now.
```

Changing something nix-side may imply recreating / updating some existing infrastructure.
It's up to you to decide how changes nix-side should impact your infrastructure.

### How does this combine with other modules ?

Use the `nix_store_path` **resource** to do something in other module, like deploying a nixos system to amazon:

```terraform
provider "nix" {}

resource "nix_store_path" "awesome_host_vhd" {
  installable = provider::nix::flake_nixos_configuration(path.module, "awesomeHost", "formats.amazon")
}

resource "aws_s3_bucket" "nixos_ami" {}

resource "aws_s3_object" "awesome_host_vhd" {
  bucket = aws_s3_bucket.nixos_ami.id
  key = nix_store_path.awesome_host_vhd.output_path
  source = nix_store_path.awesome_host_vhd.output
}

resource "aws_ebs_snapshot_import" "awesome_host" {
  disk_container {
    format = "VHD"
    user_bucket {
      s3_bucket = aws_s3_bucket.nixos_ami.id
      s3_key    = aws_s3_object.awesome_host_vhd.id
    }
  }
}

resource "aws_ami" "awesome_host" {
  ebs_block_device {
    device_name = "/dev/xvda"
    snapshot_id = aws_ebs_snapshot_import.awesome_host.id
  }
  architecture = provider::nix::system_to_ami_architecture(nix_store_path.awesome_host_vhd.system)
}

resource "aws_instance" "awesome_host" {
  ami                    = aws_ami.awesome_host.id
  instance_type          = "t2.micro"
}
```

it's shortened but you get the overall idea.
