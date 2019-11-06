package pks

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"os"
	"testing"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"pks": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

type preCheckFunc = func(*testing.T)

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("PKS_TARGET"); v == "" {
		t.Fatal("PKS_TARGET must be set for acceptance tests")
	}

	if v := os.Getenv("PKS_TOKEN"); v == "" {
		t.Fatal("PKS_TOKEN must be set for acceptance tests")
	}
}

func generateRandomResourceName() string {
	return acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
}
