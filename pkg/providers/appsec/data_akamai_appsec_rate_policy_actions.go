package appsec

import (
	"context"
	"fmt"
	"strconv"

	edge "github.com/akamai/AkamaiOPEN-edgegrid-golang/edgegrid"
	v2 "github.com/akamai/AkamaiOPEN-edgegrid-golang/v2/pkg/appsec"
	"github.com/akamai/terraform-provider-akamai/v2/pkg/akamai"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRatePolicyActions() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRatePolicyActionsRead,
		Schema: map[string]*schema.Schema{
			"config_id": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"version": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"policy_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"output_text": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Text Export representation",
			},
		},
	}
}

func dataSourceRatePolicyActionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := akamai.Meta(m)
	client := inst.Client(meta)
	logger := meta.Log("APPSEC", "resourceRatePolicyActionsRead")
	CorrelationID := "[APPSEC][resourceRatePolicyActions-" + meta.OperationID() + "]"

	getRatePolicyActions := v2.GetRatePolicyActionsRequest{}

	getRatePolicyActions.ConfigID = d.Get("config_id").(int)
	getRatePolicyActions.Version = d.Get("version").(int)
	getRatePolicyActions.PolicyID = d.Get("policy_id").(string)

	ratepolicyactions, err := client.GetRatePolicyActions(ctx, getRatePolicyActions)
	if err != nil {
		logger.Warnf("calling 'getRatePolicyActions': %s", err.Error())
	}

	for _, configval := range ratepolicyactions.RatePolicyActions {
		edge.PrintfCorrelation("[DEBUG]", CorrelationID, fmt.Sprintf("ratepolicyaction  configval %v\n", configval.ID))
		d.SetId(strconv.Itoa(configval.ID))
	}

	ots := OutputTemplates{}
	InitTemplates(ots)

	outputtext, err := RenderTemplates(ots, "ratePolicyActions", ratepolicyactions)
	edge.PrintfCorrelation("[DEBUG]", CorrelationID, fmt.Sprintf("ratePolicyActions outputtext   %v\n", outputtext))
	if err == nil {
		d.Set("output_text", outputtext)
	}

	return nil
}