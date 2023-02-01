package logstream

import (
	"context"
	"net/http"
	"strings"

	"github.com/auth0/go-auth0/management"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var validLogStreamTypes = []string{
	"eventbridge",
	"eventgrid",
	"http",
	"datadog",
	"splunk",
	"sumo",
	"mixpanel",
	"segment",
}

// NewResource will return a new auth0_log_stream resource.
func NewResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: createLogStream,
		ReadContext:   readLogStream,
		UpdateContext: updateLogStream,
		DeleteContext: deleteLogStream,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "With this resource, you can manage your Auth0 log streams.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the log stream.",
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(validLogStreamTypes, true),
				ForceNew:     true,
				Description: "Type of the log stream, which indicates the sink provider. " +
					"Options include: `" + strings.Join(validLogStreamTypes, "`, `") + "`.",
			},
			"status": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ValidateFunc: validation.StringInSlice([]string{
					"active",
					"paused",
					"suspended",
				}, false),
				Description: "The current status of the log stream. Options are \"active\", \"paused\", \"suspended\".",
			},
			"filters": {
				Type:     schema.TypeList,
				Optional: true,
				Description: "Only logs events matching these filters will be delivered by the stream." +
					" If omitted or empty, all events will be delivered.",
				Elem: &schema.Schema{
					Type: schema.TypeMap,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
			},
			"sink": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Required:    true,
				Description: "The sink configuration for the log stream.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"aws_account_id": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							RequiredWith: []string{"sink.0.aws_region"},
							Description:  "The AWS Account ID.",
						},
						"aws_region": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							RequiredWith: []string{"sink.0.aws_account_id"},
							Description:  "The AWS Region, e.g. \"us-east-2\").",
						},
						"aws_partner_event_source": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
							Description: "Name of the Partner Event Source to be used with AWS. " +
								"Generally generated by Auth0 and passed to AWS, so this should " +
								"be an output attribute.",
						},
						"azure_subscription_id": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							RequiredWith: []string{"sink.0.azure_resource_group", "sink.0.azure_region"},
							Description:  "The unique alphanumeric string that identifies your Azure subscription.",
						},
						"azure_resource_group": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							RequiredWith: []string{"sink.0.azure_subscription_id", "sink.0.azure_region"},
							Description: "The Azure EventGrid resource group which allows you to manage all " +
								"Azure assets within one subscription.",
						},
						"azure_region": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							RequiredWith: []string{"sink.0.azure_subscription_id", "sink.0.azure_resource_group"},
							Description:  "The Azure region code, e.g. \"ne\")",
						},
						"azure_partner_topic": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
							Description: "Name of the Partner Topic to be used with Azure. " +
								"Generally should not be specified.",
						},
						"http_content_format": {
							Type:         schema.TypeString,
							Optional:     true,
							RequiredWith: []string{"sink.0.http_endpoint", "sink.0.http_authorization", "sink.0.http_content_type"},
							Description: "The format of data sent over HTTP. Options are " +
								"\"JSONLINES\", \"JSONARRAY\" or \"JSONOBJECT\"",
							ValidateFunc: validation.StringInSlice([]string{
								"JSONLINES",
								"JSONARRAY",
								"JSONOBJECT",
							}, false),
						},
						"http_content_type": {
							Type:     schema.TypeString,
							Optional: true,
							Description: "The \"Content-Type\" header to send over HTTP. " +
								"Common value is \"application/json\".",
							RequiredWith: []string{"sink.0.http_endpoint", "sink.0.http_authorization", "sink.0.http_content_format"},
						},
						"http_endpoint": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "The HTTP endpoint to send streaming logs.",
							RequiredWith: []string{"sink.0.http_content_format", "sink.0.http_authorization", "sink.0.http_content_type"},
						},
						"http_authorization": {
							Type:         schema.TypeString,
							Optional:     true,
							Sensitive:    true,
							Description:  "Sent in the HTTP \"Authorization\" header with each request.",
							RequiredWith: []string{"sink.0.http_content_format", "sink.0.http_endpoint", "sink.0.http_content_type"},
						},
						"http_custom_headers": {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeMap,
								Elem: &schema.Schema{Type: schema.TypeString},
							},
							Optional:    true,
							Default:     nil,
							Description: "Additional HTTP headers to be included as part of the HTTP request.",
						},
						"datadog_region": {
							Type:         schema.TypeString,
							Optional:     true,
							RequiredWith: []string{"sink.0.datadog_api_key"},
							ValidateFunc: validation.StringInSlice(
								[]string{"us", "eu", "us3", "us5"},
								false,
							),
							Description: "The Datadog region. Options are [\"us\", \"eu\", \"us3\", \"us5\"].",
						},
						"datadog_api_key": {
							Type:         schema.TypeString,
							Optional:     true,
							Sensitive:    true,
							RequiredWith: []string{"sink.0.datadog_region"},
							Description:  "The Datadog API key.",
						},
						"splunk_domain": {
							Type:         schema.TypeString,
							Optional:     true,
							RequiredWith: []string{"sink.0.splunk_token", "sink.0.splunk_port", "sink.0.splunk_secure"},
							Description:  "The Splunk domain name.",
						},
						"splunk_token": {
							Type:         schema.TypeString,
							Optional:     true,
							Sensitive:    true,
							RequiredWith: []string{"sink.0.splunk_domain", "sink.0.splunk_port", "sink.0.splunk_secure"},
							Description:  "The Splunk access token.",
						},
						"splunk_port": {
							Type:         schema.TypeString,
							Optional:     true,
							RequiredWith: []string{"sink.0.splunk_domain", "sink.0.splunk_token", "sink.0.splunk_secure"},
							Description:  "The Splunk port.",
						},
						"splunk_secure": {
							Type:         schema.TypeBool,
							Optional:     true,
							Default:      nil,
							RequiredWith: []string{"sink.0.splunk_domain", "sink.0.splunk_port", "sink.0.splunk_token"},
							Description:  "This toggle should be turned off when using self-signed certificates.",
						},
						"sumo_source_address": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  nil,
							Description: "Generated URL for your defined HTTP source in " +
								"Sumo Logic for collecting streaming data from Auth0.",
						},
						"mixpanel_region": {
							Type:         schema.TypeString,
							Optional:     true,
							RequiredWith: []string{"sink.0.mixpanel_service_account_password", "sink.0.mixpanel_project_id", "sink.0.mixpanel_service_account_username"},
							Description: "The Mixpanel region. Options are [\"us\", \"eu\"]. " +
								"EU is required for customers with EU data residency requirements.",
						},
						"mixpanel_project_id": {
							Type:         schema.TypeString,
							Optional:     true,
							RequiredWith: []string{"sink.0.mixpanel_region", "sink.0.mixpanel_service_account_username", "sink.0.mixpanel_service_account_password"},
							Description:  "The Mixpanel project ID, found on the Project Settings page.",
						},
						"mixpanel_service_account_username": {
							Type:         schema.TypeString,
							Optional:     true,
							RequiredWith: []string{"sink.0.mixpanel_region", "sink.0.mixpanel_project_id", "sink.0.mixpanel_service_account_password"},
							Description:  "The Mixpanel Service Account username. Services Accounts can be created in the Project Settings page.",
						},
						"mixpanel_service_account_password": {
							Type:         schema.TypeString,
							Optional:     true,
							Sensitive:    true,
							RequiredWith: []string{"sink.0.mixpanel_region", "sink.0.mixpanel_project_id", "sink.0.mixpanel_service_account_username"},
							Description:  "The Mixpanel Service Account password.",
						},
						"segment_write_key": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "The [Segment Write Key](https://segment.com/docs/connections/find-writekey/).",
						},
					},
				},
			},
		},
	}
}

func createLogStream(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	api := m.(*management.Management)

	logStream := expandLogStream(d)
	if err := api.LogStream.Create(logStream); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(logStream.GetID())

	// The Management API only allows updating a log stream's status.
	// Therefore, if the status field was present in the configuration,
	// we perform an additional operation to modify it.
	status := d.Get("status").(string)
	if status != "" && status != logStream.GetStatus() {
		if err := api.LogStream.Update(logStream.GetID(), &management.LogStream{Status: &status}); err != nil {
			return diag.FromErr(err)
		}
	}

	return readLogStream(ctx, d, m)
}

func readLogStream(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	api := m.(*management.Management)

	logStream, err := api.LogStream.Read(d.Id())
	if err != nil {
		if mErr, ok := err.(management.Error); ok && mErr.Status() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	result := multierror.Append(
		d.Set("name", logStream.GetName()),
		d.Set("status", logStream.GetStatus()),
		d.Set("type", logStream.GetType()),
		d.Set("filters", logStream.Filters),
		d.Set("sink", flattenLogStreamSink(d, logStream.Sink)),
	)

	return diag.FromErr(result.ErrorOrNil())
}

func updateLogStream(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	api := m.(*management.Management)

	logStream := expandLogStream(d)
	if err := api.LogStream.Update(d.Id(), logStream); err != nil {
		return diag.FromErr(err)
	}

	return readLogStream(ctx, d, m)
}

func deleteLogStream(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	api := m.(*management.Management)

	if err := api.LogStream.Delete(d.Id()); err != nil {
		if mErr, ok := err.(management.Error); ok && mErr.Status() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
	}

	d.SetId("")
	return nil
}
