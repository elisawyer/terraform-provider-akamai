package property

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/akamai/terraform-provider-akamai/v2/pkg/tools"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/edgegrid"
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
	"github.com/akamai/terraform-provider-akamai/v2/pkg/akamai"
	"github.com/akamai/terraform-provider-akamai/v2/pkg/config"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type (
	provider struct {
		*schema.Provider
	}
)

var (
	once sync.Once

	inst *provider
)

// Subprovider returns a core sub provider
func Subprovider() akamai.Subprovider {
	once.Do(func() {
		inst = &provider{Provider: Provider()}
	})

	return inst
}

// Provider returns the Akamai terraform.Resource provider.
func Provider() *schema.Provider {
	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"papi_section": {
				Optional:   true,
				Type:       schema.TypeString,
				Default:    "default",
				Deprecated: akamai.NoticeDeprecatedUseAlias("papi_section"),
			},
			"property_section": {
				Optional:   true,
				Type:       schema.TypeString,
				Default:    "default",
				Deprecated: akamai.NoticeDeprecatedUseAlias("property_section"),
			},
			"property": {
				Optional: true,
				Type:     schema.TypeSet,
				Elem:     config.Options("property"),
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"akamai_contract":       dataSourcePropertyContract(),
			"akamai_cp_code":        dataSourceCPCode(),
			"akamai_group":          dataSourcePropertyGroups(),
			"akamai_property_rules": dataPropertyRules(),
			"akamai_property":       dataSourceAkamaiProperty(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"akamai_cp_code":             resourceCPCode(),
			"akamai_edge_hostname":       resourceSecureEdgeHostName(),
			"akamai_property":            resourceProperty(),
			"akamai_property_rules":      resourcePropertyRules(),
			"akamai_property_variables":  resourcePropertyVariables(),
			"akamai_property_activation": resourcePropertyActivation(),
		},
	}
	return provider
}

func getPAPIV1Service(d tools.ResourceDataFetcher) (*edgegrid.Config, error) {
	var papiConfig edgegrid.Config
	var err error
	property, err := tools.GetSetValue("property", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return nil, err
	}
	if err == nil {
		log.Printf("[DEBUG] Setting property config via HCL")
		cfg := property.List()[0].(map[string]interface{})

		host, ok := cfg["host"].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s, %q", tools.ErrInvalidType, "host", "string")
		}
		accessToken, ok := cfg["access_token"].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s, %q", tools.ErrInvalidType, "access_token", "string")
		}
		clientToken, ok := cfg["client_token"].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s, %q", tools.ErrInvalidType, "client_token", "string")
		}
		clientSecret, ok := cfg["client_secret"].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s, %q", tools.ErrInvalidType, "client_secret", "string")
		}
		maxBody, ok := cfg["max_body"].(int)
		if !ok {
			return nil, fmt.Errorf("%w: %s, %q", tools.ErrInvalidType, "max_body", "int")
		}
		papiConfig = edgegrid.Config{
			Host:         host,
			AccessToken:  accessToken,
			ClientToken:  clientToken,
			ClientSecret: clientSecret,
			MaxBody:      maxBody,
		}

		papi.Init(papiConfig)
		return &papiConfig, nil
	}

	edgerc, err := tools.GetStringValue("edgerc", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return nil, err
	}

	section, err := tools.GetStringValue("property_section", d, "papi_section", "config_section")
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return nil, err
	}

	papiConfig, err = edgegrid.Init(edgerc, section)
	if err != nil {
		return nil, err
	}

	papi.Init(papiConfig)
	return &papiConfig, nil
}

func (p *provider) Name() string {
	return "property"
}

// ProviderVersion update version string anytime provider adds new features
const ProviderVersion string = "v0.8.3"

func (p *provider) Version() string {
	return ProviderVersion
}

func (p *provider) Schema() map[string]*schema.Schema {
	return p.Provider.Schema
}

func (p *provider) Resources() map[string]*schema.Resource {
	return p.Provider.ResourcesMap
}

func (p *provider) DataSources() map[string]*schema.Resource {
	return p.Provider.DataSourcesMap
}

func (p *provider) Configure(ctx context.Context, log hclog.Logger, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	log.Named(p.Name()).Debug("START Configure")

	cfg, err := getPAPIV1Service(d)
	if err != nil {
		return nil, nil
	}

	return cfg, nil
}