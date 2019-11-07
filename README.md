Terraform Provider
==================

It's a terraform provider for PKS. At the moment the only completed resource is the `pks_cluster` resource for creating clusters.

Note, that this is not an officially supported provider. If you do encounter any issues raise an issue.

- Website: https://www.terraform.io
- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

Requirements
------------

- [Terraform](https://www.terraform.io/downloads.html) 0.12
- [Go](https://golang.org/doc/install) 1.13 (to build the provider plugin)


Using the Provider
----------------------

To use this provider in your Terraform environment, follow the instructions to [install it as a plugin.](https://www.terraform.io/docs/plugins/basics.html#installing-a-plugin) After placing it into your plugins directory,  run `terraform init` to initialize it.

Example Configuration
----------------------

A simple configuration to create a cluster:
```
provider "pks" {
  target = "${var.pks_api_dns_name}"
  token = "${var.token}"
}

resource "pks_cluster" "example" {
  name = "example1"
  external_hostname = "${var.k8s_api_dns_name}"
  plan = "small"
  num_nodes = 1
}

variable "k8s_api_dns_name" {
  type = "string"
}

variable "pks_api_dns_name" {
  type = "string"
}

variable "token" {
  type = "string"
}
```


Configuration Options
----------------------

Configuration options can be found :
* [here](/docs/provider_configuration.md) for the provider itself
* [here](/docs/resource_pks_cluster.md)for the `pks_cluster` resource

Developing the Provider
---------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (please check the [requirements](https://github.com/terraform-providers/terraform-provider-aws#requirements) before proceeding).

*Note:* This project uses [Go Modules](https://blog.golang.org/using-go-modules) making it safe to work with it outside of your existing [GOPATH](http://golang.org/doc/code.html#GOPATH). The instructions that follow assume a directory in your home directory outside of the standard GOPATH (i.e `$HOME/development/terraform-providers/`).

Clone repository to: `$HOME/development/terraform-providers/`

```sh
$ mkdir -p $HOME/development/terraform-providers/; cd $HOME/development/terraform-providers/
$ git clone git@github.com:terraform-providers/terraform-provider-aws
...
```

Enter the provider directory and run `make tools`. This will install some useful tools for the provider.

```sh
$ make tools
```

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

```sh
$ make build
...
$ $GOPATH/bin/terraform-provider-aws
...
```

Testing the Provider
---------------------------

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run. Please read [Running an Acceptance Test](https://github.com/terraform-providers/terraform-provider-aws/blob/master/.github/CONTRIBUTING.md#running-an-acceptance-test) in the contribution guidelines for more information on usage.

```sh
$ make testacc
```

Contributing
---------------------------

Issues on GitHub are intended to be related to bugs or feature requests with provider codebase. See https://www.terraform.io/docs/extend/community/index.html for a list of community resources to ask questions about Terraform.
~~~