package pks

import (
	"bufio"
	"bytes"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

//TODO: do after director resource is working

//TODO: make this have ref to the director resource to make sure resources are ordered correctly
// maybe later so users not creating a director with tf can still use this
// must remember to add depends_on in the meantime
// introduce a director data source at the same time

//TODO: add mutexes around apply-changes
// this was the suggested approach to control concurrency here: https://github.com/terraform-providers/terraform-provider-aws/issues/483

func resourcePcfTile() *schema.Resource {
	return &schema.Resource{
		Create: resourcePcfTileCreate,
		Read:   resourcePcfTileRead,
		Update: resourcePcfTileUpdate,
		Delete: resourcePcfTileDelete,
		Importer: &schema.ResourceImporter{
			//TODO is this correct?
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"product_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Product that will be installed as a tile",
			},
			"tile_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "(not implemented) Version of the tile to be installed",
			},
			"stemcell_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "(not implemented) Version of the tile to be installed",
			},
			"tile_config": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validation.ValidateJsonString,
				DiffSuppressFunc: suppressEquivalentJsonDiffs,
				Description:      "JSON config file for director, IaaS, and security properties",
			},
		},
	}
}

func resourcePcfTileCreate(d *schema.ResourceData, m interface{}) error {

	return resourcePcfTileUpdate(d, m)
}

func resourcePcfTileRead(d *schema.ResourceData, m interface{}) error {
	return nil
}

func resourcePcfTileUpdate(d *schema.ResourceData, m interface{}) error {
	opsmanClient := m.(*Client)

	//TODO: makes more sense to have a product as a data source?
	//upload to the opsman
	tileFile, _ := os.Open(d.Get("tile_file").(string))
	defer tileFile.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("product", "file")
	io.Copy(part, tileFile)
	writer.Close()

	req, _ := http.NewRequest("POST", "https://"+opsmanClient.target+"/api/v0/available_products", bufio.NewReader(tileFile))
	req.Header["Authorization"] = []string{"Bearer " + opsmanClient.token}
	req.Header["Content-Type"] = []string{writer.FormDataContentType()}
	opsmanClient.httpClient.Do(req)
	req.Header.Del("Accept-Encoding")
	// stage product

	// configure product

	// apply changes on product
	return resourcePcfTileRead(d, m)
}

func resourcePcfTileDelete(d *schema.ResourceData, m interface{}) error {
	return nil
}
