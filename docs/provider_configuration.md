# PKS Provider

For provisioning Kubernetes clusters through the PKS API. The provider needs to be configured with the proper credentials before it can be used.

## Example Usage

```hcl
# Configure the PKS provider.
provider "pks" {
  token  = "${var.token}"
  target = "${var.target}"
}

# Create a cluster
resource "pks_cluster" "example" {
  # ...
}
```

## Argument Reference

One of either `token`, or `client_id` + `client_secret` must be specified to authenticate with PKS:

* `token` - (Optional) A Bearer token used to login to PKS. This can be retrieved from the PKS UAA with the following curl command: `BEARER_TOKEN="$(curl -s https://${PKS_ADDRESS}:8443/oauth/token -k -XPOST -H 'Accept: application/json;charset=utf-8' -u "client_id:client_secret" -H 'Content-Type: application/x-www-form-urlencoded;charset=utf-8' -d 'grant_type=client_credentials' | jq -r .access_token)`, using client credentials such as the "UAA Management Admin Client" credential in the PKS Tile. The token can also be passed to the provider with the `PKS_TOKEN` shell environment variable. 
* `client_id` - Can also be passed to the provider with the `PKS_CLIENT_ID` shell environment variable. 
* `client_secret` - Can also be passed to the provider with the `PKS_CLIENT_SECRET` shell environment variable. 

The following additional arguments are supported:

* `target` - (Required) Hostname of the PKS API to connect to. Can also be passed to the provider with the `PKS_TARGET` shell environment variable. 
* `skip_ssl_validation` - (Optional) Default `false`. Can also be passed to the provider with the `PKS_SKIP_SSL_VALIDATION` shell environment variable. 
* `max_wait_min` - (Optional) Length of time (in minutes) that the provider will wait for PKS operations to complete. Default: 20. Can also be passed to the provider with the `PKS_MAX_WAIT_MIN` shell environment variable. 
* `wait_poll_interval_sec` - (Optional) Frequency of polling (in seconds) while waiting for PKS operations to complete. Default: 10. Can also be passed to the provider with the `PKS_WAIT_POLL_INTERVAL_SEC` shell environment variable. 