package pks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

func resourcePksCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourcePksClusterCreate,
		Read:   resourcePksClusterRead,
		// Update: resourcePksClusterUpdate,
		Delete: resourcePksClusterDelete,
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

func resourcePksClusterCreate(d *schema.ResourceData, m interface{}) error {
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

	if resp.StatusCode > 299 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("cluster creation returned unexpected status %q with response: %q", resp.Status, body)
	}

	err = waitForClusterAction(pksClient, name, "CREATE")
	if err != nil {
		return err
	}

	// 4. Set ID after success so terraform can continue to manage the resource
	d.SetId(name)

	return resourcePksClusterRead(d, m)
}

func resourcePksClusterRead(d *schema.ResourceData, m interface{}) error {
	pksClient := m.(*Client)
	// only use *terraform* ID (cluster name) to do the read, this is all that is guaranteed to be set
	// in particular, on import only ID is set
	name := d.Id()

	cr, exists, err := getCluster(pksClient, name)
	if err != nil {
		return err
	}

	if !exists {
		// resource doesn't exist, so we have to remove it from the state
		d.SetId("")
		return nil
	}

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

/*func resourcePksClusterUpdate(d *schema.ResourceData, m interface{}) error {
	// TODO
	pksClient := m.(*Client)
	return resourcePksClusterRead(d, m)
}*/

func resourcePksClusterDelete(d *schema.ResourceData, m interface{}) error {
	pksClient := m.(*Client)
	name := d.Id()

	err := deleteCluster(pksClient, name)
	if err != nil {
		return err
	}

	err = waitForClusterAction(pksClient, name, "DELETE")
	if err != nil {
		return err
	}

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

	// may take a few moments for our action to be registered in PKS
	maxPollingRetries := 3
	pollingRetries := 0

	// Keep trying until we're timed out or got a result or got an error
	for {
		select {
		case <-timeout:
			return fmt.Errorf("Timed out waiting for action %q to succeed on cluster %q", action, clusterName)
		case <-tick:
			cr, exists, err := getCluster(client, clusterName)
			if err != nil {
				return err
			}

			// checking if cluster exists
			if strings.EqualFold("DELETE", action) && !exists {
				// delete action completed ok
				return nil
			} else if !exists {
				if pollingRetries < maxPollingRetries {
					pollingRetries = pollingRetries + 1
					break
				} else {
					return fmt.Errorf("Cluster %q not found while waiting for action %q", clusterName, action)
				}
			}

			// checking the action is what we expected
			if !strings.EqualFold(cr.LastAction, action) {
				if pollingRetries < maxPollingRetries {
					pollingRetries = pollingRetries + 1
					break
				} else {
					return fmt.Errorf("Found an unexpected action on our cluster: %q, status: %q (%q)", cr.LastAction,
						cr.LastActionState, cr.LastActionDescription)
				}
			}

			// check the status of our action
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

func getCluster(client *Client, clusterName string) (*ClusterResponse, bool, error) {
	req, _ := http.NewRequest("GET", "https://"+client.target+":9021/v1/clusters/"+clusterName, nil)
	req.Header["Authorization"] = []string{"Bearer " + client.token}
	req.Header["Accept"] = []string{"application/json; charset=utf-8"}
	resp, err := client.httpClient.Do(req)
	if err != nil {
		// this doesn't catch 4xx/5xx !
		return nil, false, fmt.Errorf("error reading cluster from PKS API %q: %q", req.URL.String(), err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, false, nil
	} else if resp.StatusCode > 299 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("cluster read returned unexpected status %q with response: %q", resp.Status, body)
	}

	var cr ClusterResponse
	err = json.NewDecoder(resp.Body).Decode(&cr)
	if err != nil {
		return nil, false, fmt.Errorf("error parsing cluster response from PKS API %q: %q", req.URL.String(), err.Error())
	}
	return &cr, true, nil
}

func deleteCluster(client *Client, clusterName string) error {
	req, _ := http.NewRequest("DELETE", "https://"+client.target+":9021/v1/clusters/"+clusterName, nil)
	req.Header["Authorization"] = []string{"Bearer " + client.token}
	req.Header["Accept"] = []string{"application/json; charset=utf-8"}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error deleting cluster from PKS API %q: %q", req.URL.String(), err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// cluster was already deleted
		return nil
	} else if resp.StatusCode > 299 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Cluster delete returned unexpected status %q with response: %q", resp.Status, body)
	}

	return nil
}
