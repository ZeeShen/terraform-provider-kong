package kong

import (
	"fmt"
	"github.com/dghubble/sling"
	"github.com/hashicorp/terraform/helper/schema"
	"net/http"
)

type Upstream struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Slots int    `json:"slots,omitempty"`
}

func resourceKongUpstream() *schema.Resource {
	return &schema.Resource{
		Create: resourceKongUpstreamCreate,
		Read:   resourceKongUpstreamRead,
		Update: resourceKongUpstreamUpdate,
		Delete: resourceKongUpstreamDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The upstream name.",
			},

			"slots": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     100,
				ForceNew:    true,
				Description: "The number of slots in the loadbalancer algorithm.",
			},
		},
	}
}

func resourceKongUpstreamCreate(d *schema.ResourceData, meta interface{}) error {
	sling := meta.(*sling.Sling)
	upstream := getUpstreamFromResourceData(d)
	createdUpstream := new(Upstream)

	resp, err := sling.New().BodyJSON(upstream).Post("upstreams/").ReceiveSuccess(createdUpstream)
	if err != nil {
		return fmt.Errorf("error while creating upstream: " + err.Error())
	}
	if resp.StatusCode == http.StatusConflict {
		return fmt.Errorf("409 Conflict - use terraform import to manage this upstream.")
	} else if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code received: " + resp.Status)
	}

	setUpstreamToResourceData(d, createdUpstream)
	return nil
}

func resourceKongUpstreamRead(d *schema.ResourceData, meta interface{}) error {
	sling := meta.(*sling.Sling)
	id := d.Get("id").(string)
	upstream := new(Upstream)

	resp, err := sling.New().Path("upstreams/").Get(id).ReceiveSuccess(upstream)
	if err != nil {
		return fmt.Errorf("error while reading upstream: " + err.Error())
	}
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code received: " + resp.Status)
	}

	setUpstreamToResourceData(d, upstream)
	return nil
}

func resourceKongUpstreamUpdate(d *schema.ResourceData, meta interface{}) error {
	sling := meta.(*sling.Sling)
	upstream := getUpstreamFromResourceData(d)
	updatedUpstream := new(Upstream)

	resp, err := sling.New().BodyJSON(upstream).Patch("upstreams/").Path(upstream.ID).ReceiveSuccess(updatedUpstream)
	if err != nil {
		return fmt.Errorf("error while updating upstream" + err.Error())
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code received: " + resp.Status)
	}

	setUpstreamToResourceData(d, updatedUpstream)
	return nil
}

func resourceKongUpstreamDelete(d *schema.ResourceData, meta interface{}) error {
	sling := meta.(*sling.Sling)

	id := d.Get("id").(string)

	resp, err := sling.New().Delete("upstreams/").Path(id).ReceiveSuccess(nil)
	if err != nil {
		return fmt.Errorf("error while deleting upstream" + err.Error())
	} else if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code received: " + resp.Status)
	}

	return nil
}

func getUpstreamFromResourceData(d *schema.ResourceData) *Upstream {
	upstream := &Upstream{
		Name:  d.Get("name").(string),
		Slots: d.Get("slots").(int),
	}

	if id, ok := d.GetOk("id"); ok {
		upstream.ID = id.(string)
	}
	return upstream
}

func setUpstreamToResourceData(d *schema.ResourceData, upstream *Upstream) {
	d.SetId(upstream.ID)
	d.Set("name", upstream.Name)
	d.Set("slots", upstream.Slots)
}
