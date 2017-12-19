package kong

import (
	"fmt"
	"github.com/dghubble/sling"
	"github.com/hashicorp/terraform/helper/schema"
	"net/http"
)

type TargetRequest struct {
	ID       string `json:"id,omitempty"`
	Target   string `json:"target,omitempty"`
	Weight   int    `json:"weight,omitempty"`
	Upstream string `json:"-"`
}

type TargetResponse struct {
	ID       string `json:"id,omitempty"`
	Target   string `json:"target,omitempty"`
	Weight   int    `json:"weight,omitempty"`
	Upstream string `json:"upstream_id,omitempty"`
}

type targetQuery struct {
	ID     string `url:"id,omitempty"`
	Target string `url:"target,omitempty"`
	Weight int    `url:"weight,omitempty"`
}

type Targets struct {
	Total int              `json:"total,omitempty"`
	Data  []TargetResponse `json:"data,omitempty"`
}

func resourceKongTarget() *schema.Resource {
	return &schema.Resource{
		Create: resourceKongTargetCreate,
		Read:   resourceKongTargetRead,
		Update: resourceKongTargetUpdate,
		Delete: resourceKongTargetDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"target": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The target address (ip or hostname) and port.",
			},

			"weight": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     100,
				Description: "The weight this target gets within the upstream loadbalancer.",
			},

			"upstream": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The unique identifier or the name of the upstream to which to add the target.",
			},
		},
	}
}

func resourceKongTargetCreate(d *schema.ResourceData, meta interface{}) error {
	sling := meta.(*sling.Sling)
	target := getTargetFromResourceData(d)
	createdTarget := new(TargetResponse)

	resp, err := sling.New().BodyJSON(target).Path("upstreams/" + target.Upstream + "/").Post("targets").ReceiveSuccess(createdTarget)
	if err != nil {
		return fmt.Errorf("error while creating target: " + err.Error())
	} else if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code received: " + resp.Status)
	}

	setTargetToResourceData(d, createdTarget)

	return nil
}

func resourceKongTargetRead(d *schema.ResourceData, meta interface{}) error {
	sling := meta.(*sling.Sling)
	target := getTargetFromResourceData(d)
	query := &targetQuery{ID: target.ID}
	targets := new(Targets)

	resp, err := sling.New().Path("upstreams/" + target.Upstream + "/targets/active/").QueryStruct(query).ReceiveSuccess(targets)
	if err != nil {
		return fmt.Errorf("error while reading target: " + err.Error())
	}
	if resp.StatusCode == http.StatusNotFound || targets.Total <= 0 || len(targets.Data) <= 0 {
		d.SetId("")
		return nil
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code received: " + resp.Status)
	}

	setTargetToResourceData(d, &targets.Data[0])
	return nil
}

// target is immutable, update = delete + add
func resourceKongTargetUpdate(d *schema.ResourceData, meta interface{}) error {
	sling := meta.(*sling.Sling)
	target := getTargetFromResourceData(d)
	createdTarget := new(TargetResponse)

	resp, err := sling.New().Delete("upstreams/" + target.Upstream + "/targets/" + target.ID).ReceiveSuccess(nil)
	if err != nil {
		return fmt.Errorf("error while updating target" + err.Error())
	} else if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code received: " + resp.Status)
	}

	resp, err = sling.New().BodyJSON(target).Path("upstreams/" + target.Upstream + "/").Post("targets").ReceiveSuccess(createdTarget)
	if err != nil {
		return fmt.Errorf("error while updating target" + err.Error())
	} else if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code received: " + resp.Status)
	}

	setTargetToResourceData(d, createdTarget)
	return nil
}

func resourceKongTargetDelete(d *schema.ResourceData, meta interface{}) error {
	sling := meta.(*sling.Sling)
	target := getTargetFromResourceData(d)

	resp, err := sling.New().Delete("upstreams/" + target.Upstream + "/targets/" + target.ID).ReceiveSuccess(nil)
	if err != nil {
		return fmt.Errorf("error while deleting target" + err.Error())
	} else if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code received: "+resp.Status, target, d)
	}
	return nil
}

func getTargetFromResourceData(d *schema.ResourceData) *TargetRequest {
	target := &TargetRequest{
		Target:   d.Get("target").(string),
		Upstream: d.Get("upstream").(string),
		Weight:   d.Get("weight").(int),
	}

	if id, ok := d.GetOk("id"); ok {
		target.ID = id.(string)
	}

	return target
}

func setTargetToResourceData(d *schema.ResourceData, target *TargetResponse) {
	d.SetId(target.ID)
	d.Set("target", target.Target)
	d.Set("weight", target.Weight)
	d.Set("upstream", target.Upstream)
}