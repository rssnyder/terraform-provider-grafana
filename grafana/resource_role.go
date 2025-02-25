package grafana

import (
	"context"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func ResourceRole() *schema.Resource {
	return &schema.Resource{
		Description: `
**Note:** This resource is available only with Grafana Enterprise 8.+.

* [Official documentation](https://grafana.com/docs/grafana/latest/enterprise/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/access_control/)
`,
		CreateContext: CreateRole,
		UpdateContext: UpdateRole,
		ReadContext:   ReadRole,
		DeleteContext: DeleteRole,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"uid": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
			"version": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"global": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
			"permissions": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:     schema.TypeString,
							Required: true,
						},
						"scope": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func CreateRole(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)

	role := gapi.Role{
		UID:         d.Get("uid").(string),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Version:     int64(d.Get("version").(int)),
		Global:      d.Get("global").(bool),
		Permissions: permissions(d),
	}
	r, err := client.NewRole(role)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("uid", r.UID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(r.UID)
	return nil
}

func permissions(d *schema.ResourceData) []gapi.Permission {
	p, ok := d.GetOk("permissions")
	if !ok {
		return nil
	}

	perms := make([]gapi.Permission, 0)
	for _, permission := range p.(*schema.Set).List() {
		p := permission.(map[string]interface{})
		perms = append(perms, gapi.Permission{
			Action: p["action"].(string),
			Scope:  p["scope"].(string),
		})
	}

	return perms
}

func ReadRole(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)
	uid := d.Id()
	r, err := client.GetRole(uid)

	if err != nil {
		if strings.Contains(err.Error(), "role not found") {
			log.Printf("[WARN] removing role %s from state because it no longer exists in grafana", uid)
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}
	err = d.Set("version", r.Version)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("name", r.Name)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("uid", r.UID)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("description", r.Description)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("global", r.Global)
	if err != nil {
		return diag.FromErr(err)
	}
	perms := make([]interface{}, 0)
	for _, p := range r.Permissions {
		pMap := map[string]interface{}{
			"action": p.Action,
			"scope":  p.Scope,
		}
		perms = append(perms, pMap)
	}
	err = d.Set("permissions", perms)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(r.UID)
	return nil
}

func UpdateRole(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)

	if d.HasChange("version") || d.HasChange("name") || d.HasChange("description") || d.HasChange("permissions") {
		desc := ""
		// If description is defined, use the value from the config
		if v, ok := d.GetOk("description"); !ok {
			desc = v.(string)
		}
		r := gapi.Role{
			UID:         d.Id(),
			Name:        d.Get("name").(string),
			Description: desc,
			Version:     int64(d.Get("version").(int)),
			Permissions: permissions(d),
		}
		if err := client.UpdateRole(r); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func DeleteRole(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)
	uid := d.Id()
	g := d.Get("global").(bool)

	if err := client.DeleteRole(uid, g); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
