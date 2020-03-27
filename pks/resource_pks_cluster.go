package pks

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"log"
)

func resourcePksCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourcePksClusterCreate,
		Read:   resourcePksClusterRead,
		Update: resourcePksClusterUpdate,
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
				Description: "Last action performed on the cluster through PKS, one of CREATE, UPDATE, DELETE",
			},

			"last_action_state": {
				Type:     schema.TypeString,
				Computed: true,
				//ValidateFunc: validation.StringInSlice([]string{"in progress", "succeeded", "failed"}, true),
				Description: "One of: in progress, succeeded, failed",
			},

			"last_action_description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "",
			},

			"net_profile_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Network profile used to create cluster",
				ForceNew:    true,
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
	}
	if workers, ok := d.GetOk("num_nodes"); ok {
		params.KubernetesWorkerInstances = int64(workers.(int))
	}

	clusterReq := ClusterRequest{
		Parameters: params,
		Name:       name,
		PlanName:   d.Get("plan").(string),
		NetworkProfileName:   d.Get("net_profile_name").(string),
	}

	log.Printf("[DEBUG] PKS cluster create request configuration: %#v", clusterReq)

	err := CreateCluster(pksClient, clusterReq)
	if err != nil {
		return err
	}

	err = WaitForClusterAction(pksClient, name, "CREATE")
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

	cr, exists, err := GetCluster(pksClient, name)
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
	d.Set("last_action", cr.LastAction)
	d.Set("last_action_state", cr.LastActionState)
	d.Set("last_action_description", cr.LastActionDescription)
	d.Set("master_ips", cr.KubernetesMasterIps)
	d.Set("net_profile_name", cr.NetworkProfileName)

	return nil
}

func resourcePksClusterUpdate(d *schema.ResourceData, m interface{}) error {
	pksClient := m.(*Client)
	name := d.Id()

	updateClusterReq := UpdateClusterParameters{}

	updatesFound := false
	if numNodes, ok := d.GetOk("num_nodes"); ok {
		updateClusterReq.KubernetesWorkerInstances = int64(numNodes.(int))
		updatesFound = true
	}

	if updatesFound {
		err := UpdateCluster(pksClient, name, updateClusterReq)
		if err != nil {
			return err
		}

		err = WaitForClusterAction(pksClient, name, "UPDATE")
		if err != nil {
			return err
		}
	}

	return resourcePksClusterRead(d, m)
}

func resourcePksClusterDelete(d *schema.ResourceData, m interface{}) error {
	pksClient := m.(*Client)
	name := d.Id()

	err := DeleteCluster(pksClient, name)
	if err != nil {
		return err
	}

	err = WaitForClusterAction(pksClient, name, "DELETE")
	if err != nil {
		return err
	}

	return nil
}
