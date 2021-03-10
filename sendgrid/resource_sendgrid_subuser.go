/*
Provide a resource to manage a subuser.
Example Usage
```hcl
resource "sendgrid_subuser" "subuser" {
	username = "my-subuser"
	email    = "subuser@example.org"
	password = "Passw0rd!"
	ips      = [
		"127.0.0.1"
	]
}
```
Import
A subuser can be imported, e.g.
```hcl
$ terraform import sendgrid_subuser.subuser userName
```
*/
package sendgrid

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sendgrid "github.com/trois-six/terraform-provider-sendgrid/sdk"
)

var ErrSubUserNotFound = errors.New("subUser wasn't found")

func subUserNotFound(name string) error {
	return fmt.Errorf("%w: %s", ErrSubUserNotFound, name)
}

func resourceSendgridSubuser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSendgridSubuserCreate,
		ReadContext:   resourceSendgridSubuserRead,
		UpdateContext: resourceSendgridSubuserUpdate,
		DeleteContext: resourceSendgridSubuserDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Description: "The name of the subuser.",
				Required:    true,
			},
			"password": {
				Type:        schema.TypeString,
				Description: "The password the subuser will use when logging into SendGrid.",
				Sensitive:   true,
				Required:    true,
			},
			"email": {
				Type:        schema.TypeString,
				Description: "The email of the subuser.",
				Required:    true,
			},
			"ips": {
				Type:        schema.TypeSet,
				Description: "The IP addresses that should be assigned to this subuser.",
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"user_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"disabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"signup_session_token": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"authorization_token": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"credit_allocation_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceSendgridSubuserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*sendgrid.Client)

	username := d.Get("username").(string)
	password := d.Get("password").(string)
	email := d.Get("email").(string)

	ipsSet := d.Get("ips").(*schema.Set).List()
	ips := make([]string, 0)

	for _, ip := range ipsSet {
		ips = append(ips, ip.(string))
	}

	if err := resource.RetryContext(
		ctx,
		d.Timeout(schema.TimeoutCreate),
		retrySubUserCreateClient(c, username, email, password, ips),
	); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(username)

	if d.Get("disabled").(bool) {
		return resourceSendgridSubuserUpdate(ctx, d, m)
	}

	return resourceSendgridSubuserRead(ctx, d, m)
}

func retrySubUserCreateClient(
	c *sendgrid.Client,
	username string,
	email string,
	password string,
	ips []string,
) func() *resource.RetryError {
	return func() *resource.RetryError {
		_, requestErr := c.CreateSubuser(username, email, password, ips)

		if requestErr.Err != nil && requestErr.StatusCode == http.StatusTooManyRequests {
			return resource.RetryableError(ErrCreateRateLimit)
		}

		if requestErr.Err != nil {
			return resource.NonRetryableError(requestErr.Err)
		}

		return nil
	}
}

func resourceSendgridSubuserRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*sendgrid.Client)

	subUser, requestErr := c.ReadSubUser(d.Id())
	if requestErr.Err != nil {
		return diag.FromErr(requestErr.Err)
	}

	if len(subUser) == 0 {
		return diag.FromErr(subUserNotFound(d.Id()))
	}

	//nolint:errcheck
	d.Set("user_id", subUser[0].ID)
	//nolint:errcheck
	d.Set("disabled", subUser[0].Disabled)
	//nolint:errcheck
	d.Set("email", subUser[0].Email)

	return nil
}

func resourceSendgridSubuserUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*sendgrid.Client)

	if d.HasChange("disabled") {
		_, requestErr := c.UpdateSubuser(d.Id(), d.Get("disabled").(bool))
		if requestErr.Err != nil {
			return diag.FromErr(requestErr.Err)
		}
	}

	return resourceSendgridSubuserRead(ctx, d, m)
}

func retrySubUserDeleteClient(c *sendgrid.Client, username string) func() *resource.RetryError {
	return func() *resource.RetryError {
		_, requestErr := c.DeleteSubuser(username)

		if requestErr.Err != nil && requestErr.StatusCode == http.StatusTooManyRequests {
			return resource.RetryableError(ErrCreateRateLimit)
		}

		if requestErr.Err != nil {
			return resource.NonRetryableError(
				fmt.Errorf("error creating subuser: %w", requestErr.Err),
			)
		}

		return nil
	}
}

func resourceSendgridSubuserDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*sendgrid.Client)

	if err := resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), retrySubUserDeleteClient(
		c, d.Id())); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
