package pks

import (
	"crypto/tls"
	"fmt"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform-plugin-sdk/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"net/http"
)

type Client struct {
	hostname, token, clientId, clientSecret, username, password string
	httpClient                                                  *http.Client
	maxWaitMin, waitPollIntervalSec                             int64
}

func Provider() terraform.ResourceProvider {
	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"hostname": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PKS_HOSTNAME", nil),
			},

			"token": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"client_id", "client_secret"},
				DefaultFunc:   schema.EnvDefaultFunc("PKS_TOKEN", nil),
				Description:   "Use generated token from UAA in lieu of normal auth",
			},

			"client_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"token"},
				DefaultFunc:   schema.EnvDefaultFunc("PKS_CLIENT_ID", nil),
			},

			"client_secret": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"token"},
				DefaultFunc:   schema.EnvDefaultFunc("PKS_CLIENT_SECRET", nil),
			},

			/* TODO (check whats the supported service client auth flow for pks api
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
			"pks_network_profile": resourcePksNetworkProfile(),
			"pks_sink": resourcePksSink(),
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

	hostname := d.Get("hostname").(string)

	// make sure we have a token via one of the auth methods
	clientId, clientIdOk := d.GetOk("client_id")
	clientSecret, clientSecretOk := d.GetOk("client_secret")
	token, tokenOk := d.GetOk("token")
	var clientToken string
	var err error
	if clientIdOk && clientSecretOk {
		clientToken, err = ClientLogin(c, hostname, clientId.(string), clientSecret.(string))
		if err != nil {
			return nil, err
		}
	} else if tokenOk {
		clientToken = token.(string)
	} else {
		return nil, fmt.Errorf("no valid combination of auth attributes found, set `token` OR both `client_id` and `client_secret`")
	}

	om := &Client{
		hostname:            hostname,
		token:               clientToken,
		httpClient:          c,
		maxWaitMin:          int64(d.Get("max_wait_min").(int)),
		waitPollIntervalSec: int64(d.Get("wait_poll_interval_sec").(int)),
	}

	return om, nil
}
