package pks

import (
	"crypto/tls"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform-plugin-sdk/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"net/http"
)

type Client struct {
	target, token, clientId, clientSecret, username, password string
	skipSslValidation                                         bool
	httpClient                                                *http.Client
	maxWaitMin, waitPollIntervalSec                           int64
}

func Provider() terraform.ResourceProvider {
	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"target": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("PKS_TARGET", nil),
			},

			"token": {
				Type:     schema.TypeString,
				Optional: true,
				//ConflictsWith: []string{"username", "password", "client_id", "client_secret"},
				DefaultFunc: schema.EnvDefaultFunc("PKS_TOKEN", nil),
				Description: "Use generated token from UAA in lieu of normal auth",
			},
			/* TODO (check whats the supported service client auth flow for pks api
			"client_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ConflictsWith: []string{"username", "password", "token"},
			},

			"client_secret": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ConflictsWith: []string{"username", "password", "token"},
			},

			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ConflictsWith: []string{"client_id", "client_secret", "token"},
			},

			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ConflictsWith: []string{"client_id", "client_secret", "token"},
			},*/

			"skip_ssl_validation": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PKS_SKIP_SSL_VALIDATION", false),
			},

			"max_wait_min": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Max length of time (in minutes) the provider will wait for async operations - like cluster creation - to finish",
				DefaultFunc: schema.EnvDefaultFunc("PKS_MAX_WAIT_MIN", 20),
			},

			"wait_poll_interval_sec": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Frequency of polling (in seconds) while waiting for async operations like cluster creation",
				DefaultFunc: schema.EnvDefaultFunc("PKS_WAIT_POLL_INTERVAL_SEC", 10),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"pks_cluster": resourcePksCluster(),
			/* TODO
			"pks_network_profile": resourcePcfTile(),
			"pks_network_profile": resourcePcfTile(),
			*/
		},
		ConfigureFunc: providerConfigure,
	}
	return provider
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	c := cleanhttp.DefaultClient()
	if d.Get("skip_ssl_validation").(bool) {
		tr := &http.Transport{
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
			DisableCompression: true,
		}
		c.Transport = logging.NewTransport("pks", tr)
	} else {
		c.Transport = logging.NewTransport("pks", c.Transport)
	}
	om := &Client{
		target:              d.Get("target").(string),
		token:               d.Get("token").(string),
		skipSslValidation:   d.Get("skip_ssl_validation").(bool),
		httpClient:          c,
		maxWaitMin:          int64(d.Get("max_wait_min").(int)),
		waitPollIntervalSec: int64(d.Get("wait_poll_interval_sec").(int)),
	}
	return om, nil
}
