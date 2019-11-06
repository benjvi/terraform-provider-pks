package main

import (
	"github.com/benjvi/terraform-provider-pcf-ops-manager/pcf_ops_manager"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider {
			return pcf_ops_manager.Provider()
		},
	})
}
