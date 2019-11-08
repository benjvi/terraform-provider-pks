package pks

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"io/ioutil"
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
				Config: testAccPksClusterAllFieldsConfig(clusterName, hostname, 1),
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
				Config: testAccPksClusterAllFieldsConfig(clusterName, hostname, 1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPksClusterExists(resourceName, clusterName),
					resource.TestCheckResourceAttr(resourceName, "plan", "small"),
					resource.TestCheckResourceAttr(resourceName, "num_nodes", "1"),
				),
			},
		},
	})
}

func TestAccPksCluster_update(t *testing.T) {
	rString := acctest.RandString(6)
	resourceName := "pks_cluster.test"
	clusterName := "tf_acc_update_" + rString
	hostname := clusterName + ".example.com"
	var initialUuid *string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPksClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPksClusterAllFieldsConfig(clusterName, hostname, 1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPksClusterExists(resourceName, clusterName),
					resource.TestCheckResourceAttr(resourceName, "plan", "small"),
					resource.TestCheckResourceAttr(resourceName, "num_nodes", "1"),
					func(state *terraform.State) error {
						uuidVal := state.RootModule().Resources[resourceName].Primary.Attributes["uuid"]
						_, err := uuid.Parse(uuidVal)
						if err != nil {
							// terraform should have noticed we deleted the cluster and triggered a recreate
							return fmt.Errorf("uuid value %q failed uuid parsing",
								uuidVal)
						}
						return nil
					},
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("Not found: %s", resourceName)
						}
						uuidState := rs.Primary.Attributes["uuid"]
						initialUuid = &uuidState

						_, err := uuid.Parse(*initialUuid)
						if err != nil {
							return fmt.Errorf("initialUuid value %q failed uuid parsing",
								*initialUuid)
						}
						return nil
					},
				),
			},
			{
				Config: testAccPksClusterAllFieldsConfig(clusterName, hostname, 2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPksClusterExists(resourceName, clusterName),
					resource.TestCheckResourceAttr(resourceName, "plan", "small"),
					resource.TestCheckResourceAttr(resourceName, "num_nodes", "2"),
					// uuid is unchanged, means this is an in-place update
					func(state *terraform.State) error {
						uuidVal := state.RootModule().Resources[resourceName].Primary.Attributes["uuid"]
						if uuidVal != *initialUuid {
							return fmt.Errorf("uuid changed from %q to %q indicated an unwanted recreation", *initialUuid, uuidVal)
						}
						return nil
					},
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
	var initialUuid *string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPksClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPksClusterAllFieldsConfig(clusterName, hostname, 1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPksClusterExists(resourceName, clusterName),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("Not found: %s", resourceName)
						}
						uuidState := rs.Primary.Attributes["uuid"]
						initialUuid = &uuidState

						_, err := uuid.Parse(*initialUuid)
						if err != nil {
							return fmt.Errorf("initialUuid value %q failed uuid parsing",
								*initialUuid)
						}
						return nil
					},
					testAccManuallyDeletePksCluster(clusterName),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccPksClusterAllFieldsConfig(clusterName, hostname, 1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPksClusterExists(resourceName, clusterName),
					func(state *terraform.State) error {
						if *initialUuid == state.RootModule().Resources[resourceName].Primary.Attributes["uuid"] {
							// terraform should have noticed we deleted the cluster and triggered a recreate
							return fmt.Errorf("uuid is unchanged even after we thought we recreated the cluster ( %s )",
								*initialUuid)
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
				Config: testAccPksClusterAllFieldsConfig(clusterName, hostname, 1),
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
		cr, _, err := GetCluster(client, clusterName)
		if err != nil {
			return err
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

		req, _ := http.NewRequest("GET", "https://"+client.hostname+":9021/v1/clusters/"+rs.Primary.ID, nil)
		req.Header["Authorization"] = []string{"Bearer " + client.token}
		req.Header["Accept"] = []string{"application/json; charset=utf-8"}
		resp, err := client.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("Error checking cluster %q is destroyed: %q", rs.Primary.ID, err.Error())
		}
		if resp.StatusCode == 404 {
			resp.Body.Close()
			return nil
		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("unexpected response %q after cluster %q destruction: %q", resp.Status, rs.Primary.ID, body)
		}
	}
	return nil
}

func testAccManuallyDeletePksCluster(clusterName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Client)
		err := DeleteCluster(client, clusterName)
		if err != nil {
			return err
		}

		err = WaitForClusterAction(client, clusterName, "DELETE")
		if err != nil {
			return err
		}

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

func testAccPksClusterAllFieldsConfig(name, hostname string, nodes int) string {
	return fmt.Sprintf(`
resource "pks_cluster" "test" {
  name = "%s"
  external_hostname = "%s"
  plan = "small"
  num_nodes = %d
}
`, name, hostname, nodes)
}
