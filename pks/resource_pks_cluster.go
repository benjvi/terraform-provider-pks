package pks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"log"
	"net/http"
	"strings"
	"time"
)

func resourcePksTile() *schema.Resource {
	return &schema.Resource{
		Create: resourcePksTileCreate,
		Read:   resourcePksTileRead,
		// Update: resourcePksTileUpdate,
		Delete: resourcePksTileDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Cluster Name",
				ForceNew:    true,
			},

			"external_hostname": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Hostname that will be assigned to the Kubernetes API",
				ForceNew:    true,
			},

			"plan": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Plan used to create cluster, will determine master size, default worker set and other cluster settings",
				ForceNew:    true,
			},

			"num_nodes": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Number of worker nodes, overriding plan-specified default",
				ForceNew:    true,
			},

			"master_ips": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "IPs assigned to the master VMs",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},

			"uuid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique ID for the cluster, use this to lookup the cluster with BOSH",
			},

			"k8s_version": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"pks_version": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"last_action": {
				Type:     schema.TypeString,
				Computed: true,
				//ValidateFunc: validation.StringInSlice([]string{"CREATE", "UPDATE", "DELETE"}, true),
				Description: "Unique ID for the cluster, use this to lookup the cluster with BOSH",
			},

			"last_action_state": {
				Type:     schema.TypeString,
				Computed: true,
				//ValidateFunc: validation.StringInSlice([]string{"in progress", "succeeded", "failed"}, true),
				Description: "Unique ID for the cluster, use this to lookup the cluster with BOSH",
			},

			"last_action_description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique ID for the cluster, use this to lookup the cluster with BOSH",
			},
		},
	}
}

func resourcePksTileCreate(d *schema.ResourceData, m interface{}) error {
	pksClient := m.(*Client)

	// 1. setup request object
	name := d.Get("name").(string)

	params := ClusterParameters{
		KubernetesMasterHost: d.Get("external_hostname").(string),
		KubernetesMasterPort: 8443,
	}
	if workers, ok := d.GetOk("num_nodes"); ok {
		params.KubernetesWorkerInstances = int64(workers.(int))
	}

	clusterReq := ClusterRequest{
		Parameters: params,
		Name:       name,
		PlanName:   d.Get("plan").(string),
	}

	log.Printf("[DEBUG] PKS cluster create request configuration: %#v", clusterReq)

	// 2. Trigger creation
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(clusterReq)
	req, _ := http.NewRequest("POST", "https://"+pksClient.target+":9021/v1/clusters", b)
	req.Header["Authorization"] = []string{"Bearer " + pksClient.token}
	req.Header["Content-Type"] = []string{"application/json; charset=utf-8"}
	req.Header["Accept"] = []string{"application/json; charset=utf-8"}
	resp, err := pksClient.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST to API to create cluster failed: %q", err)
	}
	defer resp.Body.Close()
	// TODO: check for error codes

	err = waitForClusterAction(pksClient, name, "CREATE")
	if err != nil {
		return err
	}

	// 4. Set ID after success so terraform can continue to manage the resource
	d.SetId(name)

	return resourcePksTileRead(d, m)
}

func resourcePksTileRead(d *schema.ResourceData, m interface{}) error {
	pksClient := m.(*Client)
	// only use *terraform* ID (cluster name) to do the read, this is all that is guaranteed to be set
	// in particular, on import only ID is set
	name := d.Id()

	cr, err := getCluster(pksClient, name)
	if err != nil {
		return err
	}

	// TODO: check for 404 and delete ID

	d.Set("name", cr.Name)
	d.Set("external_hostname", cr.Parameters.KubernetesMasterHost)
	d.Set("plan", cr.PlanName)
	d.Set("num_nodes", cr.Parameters.KubernetesWorkerInstances)
	d.Set("uuid", cr.Uuid)
	d.Set("last_action", cr.Name)
	d.Set("last_action_state", cr.Name)
	d.Set("last_action_description", cr.Name)
	d.Set("master_ips", cr.KubernetesMasterIps)

	return nil
}

/*func resourcePksTileUpdate(d *schema.ResourceData, m interface{}) error {
	// TODO
	pksClient := m.(*Client)
	return resourcePksTileRead(d, m)
}*/

func resourcePksTileDelete(d *schema.ResourceData, m interface{}) error {
	pksClient := m.(*Client)
	name := d.Id()
	req, _ := http.NewRequest("DELETE", "https://"+pksClient.target+":9021/v1/clusters/"+name, nil)
	req.Header["Authorization"] = []string{"Bearer " + pksClient.token}
	req.Header["Accept"] = []string{"application/json; charset=utf-8"}

	resp, err := pksClient.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error deleting cluster from PKS API %q: %q", req.URL.String(), err.Error())
	}
	defer resp.Body.Close()

	// TODO: check for error codes

	return nil
}

type ClusterRequest struct {
	Name       string            `json:"name"`
	PlanName   string            `json:"plan_name"`
	Parameters ClusterParameters `json:"parameters"`
}

type ClusterParameters struct {
	KubernetesMasterHost      string `json:"kubernetes_master_host"`
	KubernetesMasterPort      int64  `json:"kubernetes_master_port"`
	KubernetesWorkerInstances int64  `json:"kubernetes_worker_instances"`
}

type ClusterResponse struct {
	Name                  string            `json:"name"`
	PlanName              string            `json:"plan_name"`
	LastAction            string            `json:"last_action"`
	LastActionState       string            `json:"last_action_state"`
	LastActionDescription string            `json:"last_action_description"`
	Uuid                  string            `json:"uuid"`
	K8sVersion            string            `json:"k8s_version"`
	PksVersion            string            `json:"pks_version"`
	KubernetesMasterIps   []string          `json:"kubernetes_master_ips"`
	Parameters            ClusterParameters `json:"parameters"`
}

func waitForClusterAction(client *Client, clusterName, action string) error {
	timeout := time.After(time.Duration(client.maxWaitMin) * time.Minute)
	tick := time.Tick(time.Duration(client.waitPollIntervalSec) * time.Second)

	// Keep trying until we're timed out or got a result or got an error
	for {
		select {
		case <-timeout:
			return fmt.Errorf("Timeout for action %q to succeedon cluster %q", action, clusterName)
		case <-tick:
			cr, err := getCluster(client, clusterName)
			if err != nil {
				return err
			}

			if !strings.EqualFold(cr.LastAction, action) {
				return fmt.Errorf("Found an unexpected action on our cluster: %q, status: %q (%q)", cr.LastAction,
					cr.LastActionState, cr.LastActionDescription)
			}

			if strings.EqualFold(cr.LastActionState, "in progress") {
				break
			} else if strings.EqualFold(cr.LastActionState, "failed") {
				return fmt.Errorf("Cluster creation failed with error: %q", cr.LastActionDescription)
			} else if strings.EqualFold(cr.LastActionState, "succeeded") {
				return nil
			} else {
				return fmt.Errorf("Unexpected cluster status: %q", cr.LastActionState)
			}
		}
	}
}

func getCluster(client *Client, clusterName string) (*ClusterResponse, error) {
	req, _ := http.NewRequest("GET", "https://"+client.target+":9021/v1/clusters/"+clusterName, nil)
	req.Header["Authorization"] = []string{"Bearer " + client.token}
	req.Header["Accept"] = []string{"application/json; charset=utf-8"}
	resp, err := client.httpClient.Do(req)
	if err != nil {
		// this doesn't catch 4xx/5xx !
		return nil, fmt.Errorf("error reading cluster from PKS API %q: %q", req.URL.String(), err.Error())
	}
	defer resp.Body.Close()

	// TODO: check for error codes

	var cr ClusterResponse
	err = json.NewDecoder(resp.Body).Decode(&cr)
	if err != nil {
		return nil, fmt.Errorf("error parsing cluster response from PKS API %q: %q", req.URL.String(), err.Error())
	}
	return &cr, nil
}
