package pks

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"net/http"
	"testing"
)

func TestAccPksCluster_basic(t *testing.T) {
	rString := acctest.RandString(6)
	resourceName := "pks_cluster.test"
	clusterName := "tf_acc_basic_" + rString
	hostname := clusterName + ".example.com"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPksClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPksClusterBasicConfig(clusterName, hostname),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPksClusterExists(resourceName, clusterName),
					resource.TestCheckResourceAttr(resourceName, "name", clusterName),
					resource.TestCheckResourceAttr(resourceName, "external_hostname", hostname),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPksCluster_allFields(t *testing.T) {
	rString := acctest.RandString(6)
	resourceName := "pks_cluster.test"
	clusterName := "tf_acc_allfields_" + rString
	hostname := clusterName + ".example.com"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPksClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPksClusterAllFieldsConfig(clusterName, hostname),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPksClusterExists(resourceName, clusterName),
					resource.TestCheckResourceAttr(resourceName, "plan", "small"),
					resource.TestCheckResourceAttr(resourceName, "num_nodes", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccPksClusterAllFieldsConfig(clusterName, hostname),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPksClusterExists(resourceName, clusterName),
					resource.TestCheckResourceAttr(resourceName, "plan", "small"),
					resource.TestCheckResourceAttr(resourceName, "num_nodes", "1"),
				),
			},
		},
	})
}

func TestAccPksCluster_CreateAfterManualDestroy(t *testing.T) {
	rString := acctest.RandString(6)

	resourceName := "pks_cluster.test"
	clusterName := "tf_acc_recreate_" + rString
	hostname := clusterName + ".example.com"
	var initialUuid string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPksClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPksClusterAllFieldsConfig(clusterName, hostname),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPksClusterExists(resourceName, clusterName),
					testAccManuallyDeletePksCluster(resourceName, clusterName, &initialUuid),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccPksClusterAllFieldsConfig(clusterName, hostname),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPksClusterExists(resourceName, clusterName),
					func(state *terraform.State) error {
						if initialUuid == state.RootModule().Resources[resourceName].Primary.Attributes["uuid"] {
							// terraform should have noticed we deleted the cluster and triggered a recreate
							return fmt.Errorf("uuid is unchanged even after we thought we recreated the cluster ( %s )",
								initialUuid)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccPksCluster_import(t *testing.T) {
	rString := acctest.RandString(6)

	resourceName := "pks_cluster.test"
	clusterName := "tf_acc_import_" + rString
	hostname := clusterName + ".example.com"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPksClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPksClusterAllFieldsConfig(clusterName, hostname),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPksClusterExists(resourceName, clusterName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPksClusterExists(resourceName, clusterName),
				),
			},
		},
	})
}

func testAccCheckPksClusterExists(resourceName, clusterName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Cluster Name is set as ID")
		}

		client := testAccProvider.Meta().(*Client)
		req, _ := http.NewRequest("GET", "https://"+client.target+":9021/v1/clusters/"+rs.Primary.ID, nil)
		req.Header["Authorization"] = []string{"Bearer " + client.token}
		req.Header["Accept"] = []string{"application/json; charset=utf-8"}
		resp, err := client.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("error reading cluster from PKS API %q: %q", req.URL.String(), err.Error())
		}
		defer resp.Body.Close()

		var cr ClusterResponse
		err = json.NewDecoder(resp.Body).Decode(&cr)
		if err != nil {
			return fmt.Errorf("error parsing cluster response from PKS API %q: %q", req.URL.String(), err.Error())
		}

		if cr.Name != clusterName {
			return fmt.Errorf("Actual cluster name %q doesn't match expected %q", cr.Name, clusterName)
		}

		return nil
	}
}

func testAccCheckPksClusterDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pks_cluster" {
			continue
		}

		req, _ := http.NewRequest("GET", "https://"+client.target+":9021/v1/clusters/"+rs.Primary.ID, nil)
		req.Header["Authorization"] = []string{"Bearer " + client.token}
		req.Header["Accept"] = []string{"application/json; charset=utf-8"}
		resp, err := client.httpClient.Do(req)
		if err == nil {
			var cr ClusterResponse
			err = json.NewDecoder(resp.Body).Decode(&cr)
			if err != nil {
				return fmt.Errorf("error parsing cluster response from PKS API %q: %q", req.URL.String(), err.Error())
			}

			return fmt.Errorf("Cluster %q still exists: %#v", rs.Primary.ID, cr)
		}
		resp.Body.Close()
	}

	return nil
}

func testAccManuallyDeletePksCluster(resourceName, clusterName string, initialUuid *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		uuid := rs.Primary.Attributes["uuid"]
		initialUuid = &uuid

		client := testAccProvider.Meta().(*Client)
		req, _ := http.NewRequest("DELETE", "https://"+client.target+":9021/v1/clusters/"+clusterName, nil)
		req.Header["Authorization"] = []string{"Bearer " + client.token}
		req.Header["Accept"] = []string{"application/json; charset=utf-8"}

		resp, err := client.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("error deleting cluster from PKS API %q: %q", req.URL.String(), err.Error())
		}
		defer resp.Body.Close()

		return nil
	}
}

func testAccPksClusterBasicConfig(name, hostname string) string {
	return fmt.Sprintf(`
resource "pks_cluster" "test" {
  name = "%s"
  external_hostname = "%s"
  plan = "small"
}
`, name, hostname)
}

func testAccPksClusterAllFieldsConfig(name, hostname string) string {
	return fmt.Sprintf(`
resource "pks_cluster" "test" {
  name = "%s"
  external_hostname = "%s"
  plan = "small"
  num_nodes = 1
}
`, name, hostname)
}
