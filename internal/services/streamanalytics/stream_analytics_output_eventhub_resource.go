package streamanalytics

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema"
	"github.com/hashicorp/go-azure-sdk/resource-manager/streamanalytics/2021-10-01-preview/outputs"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/streamanalytics/migration"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

func resourceStreamAnalyticsOutputEventHub() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceStreamAnalyticsOutputEventHubCreateUpdate,
		Read:   resourceStreamAnalyticsOutputEventHubRead,
		Update: resourceStreamAnalyticsOutputEventHubCreateUpdate,
		Delete: resourceStreamAnalyticsOutputEventHubDelete,

		Importer: pluginsdk.ImporterValidatingResourceIdThen(func(id string) error {
			_, err := outputs.ParseOutputID(id)
			return err
		}, importStreamAnalyticsOutput(outputs.EventHubOutputDataSource{})),

		SchemaVersion: 1,
		StateUpgraders: pluginsdk.StateUpgrades(map[int]pluginsdk.StateUpgrade{
			0: migration.StreamAnalyticsOutputEventHubV0ToV1{},
		}),

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(30 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(30 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*pluginsdk.Schema{
			"name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"stream_analytics_job_name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"resource_group_name": commonschema.ResourceGroupName(),

			"eventhub_name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"servicebus_namespace": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"shared_access_policy_key": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				Sensitive:    true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"shared_access_policy_name": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"property_columns": {
				Type:     pluginsdk.TypeList,
				Optional: true,
				Elem: &pluginsdk.Schema{
					Type:         pluginsdk.TypeString,
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},

			"partition_key": {
				Type:     pluginsdk.TypeString,
				Optional: true,
			},

			"authentication_mode": {
				Type:     pluginsdk.TypeString,
				Optional: true,
				Default:  string(outputs.AuthenticationModeConnectionString),
				ValidateFunc: validation.StringInSlice([]string{
					string(outputs.AuthenticationModeMsi),
					string(outputs.AuthenticationModeConnectionString),
				}, false),
			},

			"serialization": schemaStreamAnalyticsOutputSerialization(),
		},
	}
}

func resourceStreamAnalyticsOutputEventHubCreateUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).StreamAnalytics.OutputsClient
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id := outputs.NewOutputID(subscriptionId, d.Get("resource_group_name").(string), d.Get("stream_analytics_job_name").(string), d.Get("name").(string))
	if d.IsNewResource() {
		existing, err := client.Get(ctx, id)
		if err != nil {
			if !response.WasNotFound(existing.HttpResponse) {
				return fmt.Errorf("checking for presence of existing %s: %+v", id, err)
			}
		}

		if !response.WasNotFound(existing.HttpResponse) {
			return tf.ImportAsExistsError("azurerm_stream_analytics_output_eventhub", id.ID())
		}
	}

	eventHubName := d.Get("eventhub_name").(string)
	serviceBusNamespace := d.Get("servicebus_namespace").(string)
	sharedAccessPolicyKey := d.Get("shared_access_policy_key").(string)
	sharedAccessPolicyName := d.Get("shared_access_policy_name").(string)
	propertyColumns := d.Get("property_columns").([]interface{})
	partitionKey := d.Get("partition_key").(string)

	serializationRaw := d.Get("serialization").([]interface{})
	serialization, err := expandStreamAnalyticsOutputSerialization(serializationRaw)
	if err != nil {
		return fmt.Errorf("expanding `serialization`: %+v", err)
	}

	eventHubOutputDataSourceProps := &outputs.EventHubOutputDataSourceProperties{
		PartitionKey:        utils.String(partitionKey),
		PropertyColumns:     utils.ExpandStringSlice(propertyColumns),
		EventHubName:        utils.String(eventHubName),
		ServiceBusNamespace: utils.String(serviceBusNamespace),
		AuthenticationMode:  utils.ToPtr(outputs.AuthenticationMode(d.Get("authentication_mode").(string))),
	}

	if sharedAccessPolicyKey != "" {
		eventHubOutputDataSourceProps.SharedAccessPolicyKey = &sharedAccessPolicyKey
	}

	if sharedAccessPolicyName != "" {
		eventHubOutputDataSourceProps.SharedAccessPolicyName = &sharedAccessPolicyName
	}

	var dataSource outputs.OutputDataSource = outputs.EventHubOutputDataSource{
		Properties: eventHubOutputDataSourceProps,
	}
	props := outputs.Output{
		Name: utils.String(id.OutputName),
		Properties: &outputs.OutputProperties{
			Datasource:    pointer.To(dataSource),
			Serialization: pointer.To(serialization),
		},
	}

	var createOpts outputs.CreateOrReplaceOperationOptions
	var updateOpts outputs.UpdateOperationOptions
	if d.IsNewResource() {
		if _, err := client.CreateOrReplace(ctx, id, props, createOpts); err != nil {
			return fmt.Errorf("creating %s: %+v", id, err)
		}

		d.SetId(id.ID())
	} else if _, err := client.Update(ctx, id, props, updateOpts); err != nil {
		return fmt.Errorf("updating %s: %+v", id, err)
	}

	return resourceStreamAnalyticsOutputEventHubRead(d, meta)
}

func resourceStreamAnalyticsOutputEventHubRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).StreamAnalytics.OutputsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := outputs.ParseOutputID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, *id)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			log.Printf("[DEBUG] %s was not found - removing from state!", id)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("retrieving %s: %+v", id, err)
	}

	d.Set("name", id.OutputName)
	d.Set("stream_analytics_job_name", id.StreamingJobName)
	d.Set("resource_group_name", id.ResourceGroupName)

	if model := resp.Model; model != nil {
		if props := model.Properties; props != nil {
			if ds := props.Datasource; ds != nil {
				if output, ok := (*ds).(outputs.EventHubOutputDataSource); ok {
					if outputProps := output.Properties; outputProps != nil {
						eventHubName := ""
						if v := outputProps.EventHubName; v != nil {
							eventHubName = *v
						}
						d.Set("eventhub_name", eventHubName)

						serviceBusNamespace := ""
						if v := outputProps.ServiceBusNamespace; v != nil {
							serviceBusNamespace = *v
						}
						d.Set("servicebus_namespace", serviceBusNamespace)

						sharedAccessPolicyName := ""
						if v := outputProps.SharedAccessPolicyName; v != nil {
							sharedAccessPolicyName = *v
						}
						d.Set("shared_access_policy_name", sharedAccessPolicyName)

						partitionKey := ""
						if v := outputProps.PartitionKey; v != nil {
							partitionKey = *v
						}
						d.Set("partition_key", partitionKey)

						authMode := ""
						if v := outputProps.AuthenticationMode; v != nil {
							authMode = string(*v)
						}
						d.Set("authentication_mode", authMode)

						var propertyColumns []string
						if v := outputProps.PropertyColumns; v != nil {
							propertyColumns = *v
						}
						d.Set("property_columns", propertyColumns)
					}
				}
			}

			if err := d.Set("serialization", flattenStreamAnalyticsOutputSerialization(props.Serialization)); err != nil {
				return fmt.Errorf("setting `serialization`: %+v", err)
			}
		}
	}
	return nil
}

func resourceStreamAnalyticsOutputEventHubDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).StreamAnalytics.OutputsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := outputs.ParseOutputID(d.Id())
	if err != nil {
		return err
	}

	if resp, err := client.Delete(ctx, *id); err != nil {
		if !response.WasNotFound(resp.HttpResponse) {
			return fmt.Errorf("deleting %s: %+v", id, err)
		}
	}

	return nil
}
