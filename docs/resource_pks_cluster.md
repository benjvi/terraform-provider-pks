# pks_cluster

Creates Kubernetes clusters using the PKS Cluster API. 

This resource waits for any actions taken on the cluster to be completed, allowing additional resources to be created that depend on completed cluster creation.

__CAUTION__: Updates have not been implemented yet, so any modifications to a cluster will cause recreation!

## Example Usage

```hcl
resource "pks_cluster" "example" {
  name = "example1"
  external_hostname = "example1-api.example.com"
  plan = "small"
  num_nodes = 1
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name to assign to the cluster in PKS.
* `external_hostname` - (Required) The hostname that will be used for accessing the Kubernetes cluster API.
* `plan` - (Required) Plan used to create cluster, will determine master size, default worker set and other cluster settings.
* `num_nodes` - (Optional) Number of worker nodes, overriding the default specified by the plan.

## Attributes Reference

The following attributes are exported:

* `master_ips` - IPs assigned to the Kubernetes master VMs.
* `uuid` - Unique ID for the cluster, use this to lookup the cluster with BOSH.
* `k8s_version`
* `pks_version`
* `last_action` - Last action performed on the cluster through PKS, one of "CREATE", "UPDATE", "DELETE".
* `last_action_state` - One of: "in progress", "succeeded", "failed".
* `last_action_description` - Any errors from the last action will be shown here.

## Import

Use the cluster name to import an existing cluster, e.g.

```
$ terraform import pks_cluster.example example_cluster_name
```