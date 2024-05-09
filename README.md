# Terraform Provider NixFlake

**This is a proof of concept, nothing is properly tested, things could change in a non-compatible way.**

## FAQ

### What does this provider provide ?

This module exposes two resources:

- `nix_derivation`: build a nix installable and get built paths
- `nix_copy_store_path`: copy a nix store path from one store to another

and one data source:

- `nix_store_path`: retrive nix store path information

### How can I use this provider ?

The following example uses flakes, but you don't have to.

#### flake

Create a file named `flake.nix` with the following demo content:

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

In the same directory as the `flake.nix` file above, lets create a `main.tf` file with the following content:

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

resource "nix_derivation" "awesome_host" {
  installable = "${path.module}#nixosConfigurations.awesomeHost.config.formats.amazon"
}

data "nix_store_path" "awesome_host" {
  installable = nix_derivation.awesome_host.output_path
}

data "nix_store_path" "another_host" {
  installable = "${path.module}#nixosConfigurations.anotherHost.config.formats.amazon"
}

output "from_resource" {
  value = nix_derivation.awesome_host
}

output "from_data_another_host" {
  value = data.nix_store_path.another_host
}

output "from_data_awesome_host" {
  value = data.nix_store_path.awesome_host
}
```

In this file:
- the `provider "nix"` terraform provider is defined and configured, it will be used to glue nix and terraform together
- the `data "nix_store_path"` allow us to retrieve some information about nix store paths
- the `resource "nix_derivation" "awesome_host"` allow us to build the derivation, and retrieve information about it
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
data.nix_store_path.another_host: Reading...
data.nix_store_path.another_host: Still reading... [10s elapsed]
data.nix_store_path.another_host: Still reading... [20s elapsed]
data.nix_store_path.another_host: Read complete after 20s

Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
  + create
 <= read (data resources)

Terraform will perform the following actions:

  # data.nix_store_path.awesome_host will be read during apply
  # (config refers to values not yet known)
 <= data "nix_store_path" "awesome_host" {
      + drv_path    = (known after apply)
      + installable = (known after apply)
      + output_path = (known after apply)
      + valid       = (known after apply)
    }

  # nix_derivation.awesome_host will be created
  + resource "nix_derivation" "awesome_host" {
      + drv_path    = (known after apply)
      + installable = ".#nixosConfigurations.awesomeHost.config.formats.amazon"
      + output_path = (known after apply)
    }

Plan: 1 to add, 0 to change, 0 to destroy.

Changes to Outputs:
  + from_data_another_host = {
      + drv_path    = "/nix/store/fvsifqg9picsl78132wqql5lm3dam6wb-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv"
      + installable = ".#nixosConfigurations.anotherHost.config.formats.amazon"
      + output_path = "/nix/store/7ajhwdh23iw4c4ipcz3mzyq3xg1w5r38-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
      + valid       = false
    }
  + from_data_awesome_host = {
      + drv_path    = (known after apply)
      + installable = (known after apply)
      + output_path = (known after apply)
      + valid       = (known after apply)
    }
  + from_resource          = {
      + drv_path    = (known after apply)
      + installable = ".#nixosConfigurations.awesomeHost.config.formats.amazon"
      + output_path = (known after apply)
    }

────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

Note: You didn't use the -out option to save this plan, so Terraform can't guarantee to take exactly these actions if you run "terraform apply" now.
```

here you see that the `resource "nix_derivation" "awesome_host"` will be created (installable will be built) if applied, and associated data are yet to be known.
Meanwhile, the `data "nix_store_path" "another_host"` is already capable of giving the derivation path (`drv_path`) and build output path `output_path`, without building the installable.
This data is already accessible without building the derivation because nix derivation path and output path are computed based on nix inputs, see [how nix store path works](https://nixos.org/guides/nix-pills/18-nix-store-paths.html).
Because the derivation is not built, the nix store do not contain the derivation nor the build output, that is why it is considered invalid (`valid = false`).

This is the main difference between `nix_store_path` **data** and `nix_derivation` **resource**. The former ask nix for the derivation information
(which evaluate the nix expression behind the flake installable, but does not build it), and the latter that builds it.

Let's apply this plan:

```sh
$ terraform apply
data.nix_store_path.another_host: Reading...
data.nix_store_path.another_host: Still reading... [10s elapsed]
data.nix_store_path.another_host: Still reading... [20s elapsed]
data.nix_store_path.another_host: Read complete after 21s

Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
  + create
 <= read (data resources)

Terraform will perform the following actions:

  # data.nix_store_path.awesome_host will be read during apply
  # (config refers to values not yet known)
 <= data "nix_store_path" "awesome_host" {
      + drv_path    = (known after apply)
      + installable = (known after apply)
      + output_path = (known after apply)
      + valid       = (known after apply)
    }

  # nix_derivation.awesome_host will be created
  + resource "nix_derivation" "awesome_host" {
      + drv_path    = (known after apply)
      + installable = ".#nixosConfigurations.awesomeHost.config.formats.amazon"
      + output_path = (known after apply)
    }

Plan: 1 to add, 0 to change, 0 to destroy.

Changes to Outputs:
  + from_data_another_host = {
      + drv_path    = "/nix/store/fvsifqg9picsl78132wqql5lm3dam6wb-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv"
      + installable = ".#nixosConfigurations.anotherHost.config.formats.amazon"
      + output_path = "/nix/store/7ajhwdh23iw4c4ipcz3mzyq3xg1w5r38-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
      + valid       = false
    }
  + from_data_awesome_host = {
      + drv_path    = (known after apply)
      + installable = (known after apply)
      + output_path = (known after apply)
      + valid       = (known after apply)
    }
  + from_resource          = {
      + drv_path    = (known after apply)
      + installable = ".#nixosConfigurations.awesomeHost.config.formats.amazon"
      + output_path = (known after apply)
    }

Do you want to perform these actions?
  Terraform will perform the actions described above.
  Only 'yes' will be accepted to approve.

  Enter a value: yes

nix_derivation.awesome_host: Creating...
nix_derivation.awesome_host: Still creating... [10s elapsed]
nix_derivation.awesome_host: Still creating... [20s elapsed]
nix_derivation.awesome_host: Creation complete after 21s
data.nix_store_path.awesome_host: Reading...
data.nix_store_path.awesome_host: Read complete after 0s

Apply complete! Resources: 1 added, 0 changed, 0 destroyed.

Outputs:

from_data_another_host = {
  "drv_path" = "/nix/store/fvsifqg9picsl78132wqql5lm3dam6wb-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv"
  "installable" = ".#nixosConfigurations.anotherHost.config.formats.amazon"
  "output_path" = "/nix/store/7ajhwdh23iw4c4ipcz3mzyq3xg1w5r38-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
  "valid" = false
}
from_data_awesome_host = {
  "drv_path" = "/nix/store/5ikkddmwwd05hirqyj86mlvrinmydj3v-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv"
  "installable" = "/nix/store/z0i2hszffgz5fbv4am26yij7pik8czkd-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
  "output_path" = "/nix/store/z0i2hszffgz5fbv4am26yij7pik8czkd-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
  "valid" = true
}
from_resource = {
  "drv_path" = "/nix/store/5ikkddmwwd05hirqyj86mlvrinmydj3v-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv"
  "installable" = ".#nixosConfigurations.awesomeHost.config.formats.amazon"
  "output_path" = "/nix/store/z0i2hszffgz5fbv4am26yij7pik8czkd-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
}
```

As expected, `drv_path` and `output_path` for `from_data_awesome_host` and `from_resource` are equal, we are indeed building the same derivation.
The real difference is that because the derivation is actually built it is `valid`, we can provide other modules with its output.

Note: If we manually build the `from_data_another_host` derivation via nix build, `valid` would also be true.
The issue is that it's up to the nix store to keep this store path alive, if garbage collection happens between nix build and terraform apply then the store path becomes invalid.
That is not the case with the resource definition, if the store path is missing, it will be built again.

If we run terraform apply again:
```sh
data.nix_store_path.another_host: Reading...
nix_derivation.awesome_host: Refreshing state...
data.nix_store_path.another_host: Still reading... [10s elapsed]
data.nix_store_path.another_host: Still reading... [20s elapsed]
data.nix_store_path.another_host: Still reading... [30s elapsed]
data.nix_store_path.another_host: Still reading... [40s elapsed]
data.nix_store_path.another_host: Still reading... [50s elapsed]
data.nix_store_path.another_host: Read complete after 51s
data.nix_store_path.awesome_host: Reading...
data.nix_store_path.awesome_host: Read complete after 0s

No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration and found no differences, so no changes are needed.

Apply complete! Resources: 0 added, 0 changed, 0 destroyed.

Outputs:

from_data_another_host = {
  "drv_path" = "/nix/store/fvsifqg9picsl78132wqql5lm3dam6wb-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv"
  "installable" = ".#nixosConfigurations.anotherHost.config.formats.amazon"
  "output_path" = "/nix/store/7ajhwdh23iw4c4ipcz3mzyq3xg1w5r38-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
  "valid" = false
}
from_data_awesome_host = {
  "drv_path" = "/nix/store/5ikkddmwwd05hirqyj86mlvrinmydj3v-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv"
  "installable" = "/nix/store/z0i2hszffgz5fbv4am26yij7pik8czkd-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
  "output_path" = "/nix/store/z0i2hszffgz5fbv4am26yij7pik8czkd-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
  "valid" = true
}
from_resource = {
  "drv_path" = "/nix/store/5ikkddmwwd05hirqyj86mlvrinmydj3v-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv"
  "installable" = ".#nixosConfigurations.awesomeHost.config.formats.amazon"
  "output_path" = "/nix/store/z0i2hszffgz5fbv4am26yij7pik8czkd-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
}
```

As expected, nothing changed so terraform has nothing to do.

Lets now perform a terraform destroy:

```sh
$ terraform destroy
data.nix_store_path.another_host: Reading...
nix_derivation.awesome_host: Refreshing state...
data.nix_store_path.another_host: Still reading... [10s elapsed]
data.nix_store_path.another_host: Still reading... [20s elapsed]
data.nix_store_path.another_host: Still reading... [30s elapsed]
data.nix_store_path.awesome_host: Reading...
data.nix_store_path.awesome_host: Read complete after 0s
data.nix_store_path.another_host: Still reading... [40s elapsed]
data.nix_store_path.another_host: Still reading... [50s elapsed]
data.nix_store_path.another_host: Read complete after 50s

Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
  - destroy

Terraform will perform the following actions:

  # nix_derivation.awesome_host will be destroyed
  - resource "nix_derivation" "awesome_host" {
      - drv_path    = "/nix/store/5ikkddmwwd05hirqyj86mlvrinmydj3v-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv" -> null
      - installable = ".#nixosConfigurations.awesomeHost.config.formats.amazon" -> null
      - output_path = "/nix/store/z0i2hszffgz5fbv4am26yij7pik8czkd-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd" -> null
    }

Plan: 0 to add, 0 to change, 1 to destroy.

Changes to Outputs:
  - from_data_another_host = {
      - drv_path    = "/nix/store/fvsifqg9picsl78132wqql5lm3dam6wb-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv"
      - installable = ".#nixosConfigurations.anotherHost.config.formats.amazon"
      - output_path = "/nix/store/7ajhwdh23iw4c4ipcz3mzyq3xg1w5r38-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
      - valid       = false
    } -> null
  - from_data_awesome_host = {
      - drv_path    = "/nix/store/5ikkddmwwd05hirqyj86mlvrinmydj3v-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv"
      - installable = "/nix/store/z0i2hszffgz5fbv4am26yij7pik8czkd-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
      - output_path = "/nix/store/z0i2hszffgz5fbv4am26yij7pik8czkd-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
      - valid       = true
    } -> null
  - from_resource          = {
      - drv_path    = "/nix/store/5ikkddmwwd05hirqyj86mlvrinmydj3v-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv"
      - installable = ".#nixosConfigurations.awesomeHost.config.formats.amazon"
      - output_path = "/nix/store/z0i2hszffgz5fbv4am26yij7pik8czkd-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
    } -> null

Do you really want to destroy all resources?
  Terraform will destroy all your managed infrastructure, as shown above.
  There is no undo. Only 'yes' will be accepted to confirm.

  Enter a value: yes

nix_derivation.awesome_host: Destroying...
nix_derivation.awesome_host: Destruction complete after 1s
╷
│ Warning: Delete operation may not remove garbage
│
│ See https://nixos.org/manual/nix/stable/command-ref/nix-collect-garbage and run nix-collect-garbage if needed
╵

Destroy complete! Resources: 1 destroyed.
```

Nice!
If we ask nix for the store path info:

```sh
nix path-info .#nixosConfigurations.awesomeHost.config.formats.amazon
this derivation will be built:
  /nix/store/5ikkddmwwd05hirqyj86mlvrinmydj3v-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv
error: path '/nix/store/z0i2hszffgz5fbv4am26yij7pik8czkd-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd' is not valid
```

as expected its missing.

You may have seen this notice above:
```
╷
│ Warning: Delete operation may not remove garbage
│
│ See https://nixos.org/manual/nix/stable/command-ref/nix-collect-garbage and run nix-collect-garbage if needed
╵
```

While we successfully removed the store path, all the build dependencies are still in the nix store.
Actually, rebuilding the derivation may even be near-instant, as nix would probably have all build dependencies still in the store.

Cleaning garbage (unused dependencies) is out of scope of this terraform provider, as it may have other consequences.
Consider running nix-collect-garbage manually, or set nix to automatically clean garbage when needed.

### How does it work ?

This provider executes nix commands via the `os/shell` package, through bash.
This means you need to have `bash` and `nix` in your `PATH` to run it.

For reproducibility reasons, consider providing bash, terraform, and nix through a nix shell.

Here are all the commands ran for the example above (not in order, and not considering the number of executions):
```sh
nix build --no-update-lock-file --no-write-lock-file --no-link --json .#nixosConfigurations.awesomeHost.config.formats.amazon
nix derivation show --no-update-lock-file --no-write-lock-file .#nixosConfigurations.anotherHost.config.formats.amazon
nix path-info --no-update-lock-file --no-write-lock-file --json .#nixosConfigurations.anotherHost.config.formats.amazon
nix path-info --no-update-lock-file --no-write-lock-file --json .#nixosConfigurations.awesomeHost.config.formats.amazon
nix path-info --no-update-lock-file --no-write-lock-file --json /nix/store/z0i2hszffgz5fbv4am26yij7pik8czkd-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd
nix store delete --no-update-lock-file --no-write-lock-file /nix/store/z0i2hszffgz5fbv4am26yij7pik8czkd-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd
```

### "nix_derivation.*: Refreshing state..." is super slow!

`resource "nix_derivation"` creation requires to actually call `nix build` with the provided installable.

After creation (when applying an already created resource), the provider checks with `nix path-info` if the store path is still valid:
- if it is, it returns the store path
- if it's not, it `nix build` the installable again

This is required to ensure the output exists, even in case of nix garbage collection, this also means the store path can change due to modification outside terraform (like changing something nix-side), see *Note: Objects have changed outside of Terraform* below.

Once an installable is built, considering nothing changed nix-side, rebuilding the same derivation should be near instant.

`data "nix_derivation"` requires to evaluate the nix expressions to compute the derivation, without actually building it.
It may or not be faster in some occasion. 

### Note: Objects have changed outside of Terraform

Once a terraform plan has been applied, changing anything nix-side imply that the derivation path will change and
terraform will notice it.

```sh
$ terraform plan
nix_derivation.this: Refreshing state...


Note: Objects have changed outside of Terraform

Terraform detected the following changes made outside of Terraform since the last "terraform apply" which may have affected this plan:

  # nix_derivation.this has changed
  ~ resource "nix_derivation" "this" {
      ~ output      = "/nix/store/z0i2hszffgz5fbv4am26yij7pik8czkd-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd" -> "/nix/store/7ajhwdh23iw4c4ipcz3mzyq3xg1w5r38-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
      ~ path        = "/nix/store/5ikkddmwwd05hirqyj86mlvrinmydj3v-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv" -> "/nix/store/fvsifqg9picsl78132wqql5lm3dam6wb-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv"
        # (1 unchanged attribute hidden)
    }


Unless you have made equivalent changes to your configuration, or ignored the relevant attributes using ignore_changes, the following plan may include actions to undo or respond to
these changes.

──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

Changes to Outputs:
  ~ resource_awesome_host = {
      ~ output      = "/nix/store/z0i2hszffgz5fbv4am26yij7pik8czkd-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd" -> "/nix/store/7ajhwdh23iw4c4ipcz3mzyq3xg1w5r38-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd"
      ~ path        = "/nix/store/5ikkddmwwd05hirqyj86mlvrinmydj3v-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv" -> "/nix/store/fvsifqg9picsl78132wqql5lm3dam6wb-nixos-amazon-image-24.05.20240505.25865a4-aarch64-linux.vhd.drv"
        # (1 unchanged attribute hidden)
    }

You can apply this plan to save these new output values to the Terraform state, without changing any real infrastructure.

──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

Note: You didn't use the -out option to save this plan, so Terraform can't guarantee to take exactly these actions if you run "terraform apply" now
```

It's up to you to decide how changes nix-side may impact the infrastructure.
Changing something nix-side may imply recreating / updating some existing infrastructure.


### How does this combine with other modules ?

Use the `nix derivation` **resource** to do something in other module, like deploying a nixos system to amazon:

```terraform
provider "nix" {}

resource "nix_derivation" "awesome_host_vhd" {
  installable = "${path.module}#nixosConfigurations.awesomeHost.config.formats.amazon"
}

resource "aws_s3_bucket" "nixos_ami" {}

resource "aws_s3_object" "awesome_host_vhd" {
  bucket = aws_s3_bucket.nixos_ami.id
  key = nix_derivation.awesome_host_vhd.output_path
  source = nix_derivation.awesome_host.output
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
}

resource "aws_instance" "awesome_host" {
  ami                    = aws_ami.awesome_host.id
  instance_type          = "t2.micro"
}
```

it's shortened but you get the overall idea.
