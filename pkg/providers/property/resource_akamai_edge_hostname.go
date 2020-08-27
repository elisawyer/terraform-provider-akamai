package property

import (
	"errors"
	"fmt"
	"github.com/akamai/terraform-provider-akamai/v2/pkg/akamai"
	"github.com/akamai/terraform-provider-akamai/v2/pkg/tools"
	"strings"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/jsonhooks-v1"
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceSecureEdgeHostName() *schema.Resource {
	return &schema.Resource{
		Create: resourceSecureEdgeHostNameCreate,
		Read:   resourceSecureEdgeHostNameRead,
		Delete: resourceSecureEdgeHostNameDelete,
		Exists: resourceSecureEdgeHostNameExists,
		Importer: &schema.ResourceImporter{
			State: resourceSecureEdgeHostNameImport,
		},
		Schema: akamaiSecureEdgeHostNameSchema,
	}
}

var akamaiSecureEdgeHostNameSchema = map[string]*schema.Schema{
	"product": {
		Type:     schema.TypeString,
		Required: true,
		ForceNew: true,
	},
	"contract": {
		Type:     schema.TypeString,
		Required: true,
		ForceNew: true,
	},
	"group": {
		Type:     schema.TypeString,
		Required: true,
		ForceNew: true,
	},
	"edge_hostname": {
		Type:     schema.TypeString,
		Required: true,
		ForceNew: true,
	},
	"ipv4": {
		Type:     schema.TypeBool,
		Optional: true,
		Default:  true,
		ForceNew: true,
	},
	"ipv6": {
		Type:     schema.TypeBool,
		Optional: true,
		Default:  false,
		ForceNew: true,
	},
	"ip_behavior": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"certificate": {
		Type:     schema.TypeInt,
		Optional: true,
		ForceNew: true,
	},
}

func resourceSecureEdgeHostNameCreate(d *schema.ResourceData, _ interface{}) error {
	akactx := akamai.ContextGet(inst.Name())
	logger := akactx.Log("PAPI", "resourceSecureEdgeHostNameCreate")
	d.Partial(true)
	CorrelationID := "[PAPI][resourceSecureEdgeHostNameCreate-" + akactx.OperationID() + "]"
	group, err := getGroup(d, CorrelationID, logger)
	if err != nil {
		return err
	}
	logger.Debug("  Edgehostnames GROUP = %v", group)
	contract, err := getContract(d, CorrelationID, logger)
	if err != nil {
		return err
	}
	logger.Debug("Edgehostnames CONTRACT = %v", contract)
	product, err := getProduct(d, contract, CorrelationID, logger)
	if err != nil {
		return err
	}
	if group == nil {
		return errors.New("group must be specified to create a new Edge Hostname")
	}
	if contract == nil {
		return errors.New("contract must be specified to create a new Edge Hostname")
	}
	if product == nil {
		return errors.New("product must be specified to create a new Edge Hostname")
	}

	edgeHostnames, err := papi.GetEdgeHostnames(contract, group, "")
	if err != nil {
		return err
	}
	edgeHostname, err := tools.GetStringValue("edge_hostname", d)
	if err != nil {
		return err
	}
	newHostname := edgeHostnames.NewEdgeHostname()
	newHostname.ProductID = product.ProductID
	newHostname.EdgeHostnameDomain = edgeHostname

	switch {
	case strings.HasSuffix(edgeHostname, ".edgesuite.net"):
		newHostname.DomainPrefix = strings.TrimSuffix(edgeHostname, ".edgesuite.net")
		newHostname.DomainSuffix = "edgesuite.net"
		newHostname.SecureNetwork = "STANDARD_TLS"
	case strings.HasSuffix(edgeHostname, ".edgekey.net"):
		newHostname.DomainPrefix = strings.TrimSuffix(edgeHostname, ".edgekey.net")
		newHostname.DomainSuffix = "edgekey.net"
		newHostname.SecureNetwork = "ENHANCED_TLS"
	case strings.HasSuffix(edgeHostname, ".akamaized.net"):
		newHostname.DomainPrefix = strings.TrimSuffix(edgeHostname, ".akamaized.net")
		newHostname.DomainSuffix = "akamaized.net"
		newHostname.SecureNetwork = "SHARED_CERT"
	}

	ipv4, err := tools.GetBoolValue("ipv4", d)
	if err != nil {
		return err
	}
	if ipv4 {
		newHostname.IPVersionBehavior = "IPV4"
	}
	ipv6, err := tools.GetBoolValue("ipv6", d)
	if err != nil {
		return err
	}
	if ipv6 {
		newHostname.IPVersionBehavior = "IPV6"
	}
	if ipv4 && ipv6 {
		newHostname.IPVersionBehavior = "IPV6_COMPLIANCE"
	}
	if err := d.Set("ip_behavior", newHostname.IPVersionBehavior); err != nil {
		return fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error())
	}

	certEnrollmentID, err := tools.GetIntValue("certificate", d)
	if err != nil {
		if !errors.Is(err, tools.ErrNotFound) {
			return err
		}
		if newHostname.SecureNetwork == "ENHANCED_TLS" {
			return errors.New("A certificate enrollment ID is required for Enhanced TLS (edgekey.net) edge hostnames")
		}
	}
	newHostname.CertEnrollmentId = certEnrollmentID
	newHostname.SlotNumber = certEnrollmentID

	hostname, err := edgeHostnames.FindEdgeHostname(newHostname)
	if err != nil {
		// TODO this error has to be ignored (for now) as FindEdgeHostname returns error if no hostnames were found
		logger.Debug("could not finc edge hostname: %s", err.Error())
	}
	if hostname != nil && hostname.EdgeHostnameID != "" {
		body, err := jsonhooks.Marshal(hostname)
		if err != nil {
			return err
		}
		logger.Debug("EHN Found = %s", body)

		if hostname.IPVersionBehavior != newHostname.IPVersionBehavior {
			return fmt.Errorf("existing edge hostname found with incompatible IP version (%s vs %s). You must use the same settings, or try a different edge hostname", hostname.IPVersionBehavior, newHostname.IPVersionBehavior)
		}

		logger.Debug("Existing edge hostname FOUND = %s", hostname.EdgeHostnameID)
		d.SetId(hostname.EdgeHostnameID)
		d.Partial(false)
		return nil
	}
	logger.Debug("Creating new edge hostname: %#v", newHostname)
	err = newHostname.Save("", CorrelationID)
	if err != nil {
		return err
	}
	d.SetId(newHostname.EdgeHostnameID)
	d.Partial(false)
	return nil
}

func resourceSecureEdgeHostNameDelete(d *schema.ResourceData, _ interface{}) error {
	akactx := akamai.ContextGet(inst.Name())
	logger := akactx.Log("PAPI", "resourceSecureEdgeHostNameDelete")
	logger.Debug("DELETING")
	d.SetId("")
	logger.Debug("DONE")
	return nil
}

func resourceSecureEdgeHostNameImport(d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	akactx := akamai.ContextGet(inst.Name())
	logger := akactx.Log("PAPI", "resourceSecureEdgeHostNameImport")
	resourceID := d.Id()
	propertyID := resourceID

	if !strings.HasPrefix(resourceID, "prp_") {
		keys := []papi.SearchKey{
			papi.SearchByPropertyName,
			papi.SearchByHostname,
			papi.SearchByEdgeHostname,
		}
		for _, searchKey := range keys {
			results, err := papi.Search(searchKey, resourceID, "")
			if err != nil {
				// TODO determine why is this error ignored
				logger.Debug("searching by key: %s: %w", searchKey, err)
				continue
			}

			if results != nil && len(results.Versions.Items) > 0 {
				propertyID = results.Versions.Items[0].PropertyID
				break
			}
		}
	}

	property := papi.NewProperty(papi.NewProperties())
	property.PropertyID = propertyID
	err := property.GetProperty("")
	if err != nil {
		return nil, err
	}

	if err := d.Set("account", property.AccountID); err != nil {
		return nil, fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error())
	}
	if err := d.Set("contract", property.ContractID); err != nil {
		return nil, fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error())
	}
	if err := d.Set("group", property.GroupID); err != nil {
		return nil, fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error())
	}
	if err := d.Set("name", property.PropertyName); err != nil {
		return nil, fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error())
	}
	if err := d.Set("version", property.LatestVersion); err != nil {
		return nil, fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error())
	}
	d.SetId(property.PropertyID)

	return []*schema.ResourceData{d}, nil
}

func resourceSecureEdgeHostNameExists(d *schema.ResourceData, _ interface{}) (bool, error) {
	akactx := akamai.ContextGet(inst.Name())
	logger := akactx.Log("PAPI", "resourceSecureEdgeHostNameCreate")
	CorrelationID := "[PAPI][resourceSecureEdgeHostNameCreate-" + akactx.OperationID() + "]"
	group, err := getGroup(d, CorrelationID, logger)
	if err != nil {
		return false, err
	}
	logger.Debug("Figuring out edgehostnames GROUP = %v", group)
	contract, err := getContract(d, CorrelationID, logger)
	if err != nil {
		return false, err
	}
	logger.Debug("Figuring out edgehostnames CONTRACT = %v", contract)
	property := papi.NewProperty(papi.NewProperties())
	property.Group = group
	property.Contract = contract

	logger.Debug("Figuring out edgehostnames %v", d.Id())
	edgeHostnames := papi.NewEdgeHostnames()
	logger.Debug("NewEdgeHostnames empty struct  %s", edgeHostnames.ContractID)
	err = edgeHostnames.GetEdgeHostnames(property.Contract, property.Group, d.Id(), CorrelationID)
	if err != nil {
		return false, err
	}
	// FIXME: this logic seems to be flawed - 'true' is returned whenever GetEdgeHostnames did not return an error (even if no hostnames were present in response)
	logger.Debug("Edgehostname EXISTS in contract")
	return true, nil
}

func resourceSecureEdgeHostNameRead(d *schema.ResourceData, _ interface{}) error {
	akactx := akamai.ContextGet(inst.Name())
	logger := akactx.Log("PAPI", "resourceSecureEdgeHostNameCreate")
	CorrelationID := "[PAPI][resourceSecureEdgeHostNameCreate-" + akactx.OperationID() + "]"
	d.Partial(true)

	group, err := getGroup(d, CorrelationID, logger)
	if err != nil {
		return err
	}
	logger.Debug("Figuring out edgehostnames GROUP = %v", group)
	contract, err := getContract(d, CorrelationID, logger)
	if err != nil {
		return err
	}
	logger.Debug("Figuring out edgehostnames CONTRACT = %v", contract)
	property := papi.NewProperty(papi.NewProperties())
	property.Group = group
	property.Contract = contract
	logger.Debug("Figuring out edgehostnames %v", d.Id())
	edgeHostnames := papi.NewEdgeHostnames()
	logger.Debug("NewEdgeHostnames empty struct %v", edgeHostnames.ContractID)
	err = edgeHostnames.GetEdgeHostnames(property.Contract, property.Group, "", CorrelationID)
	if err != nil {
		return err
	}
	logger.Debug("EdgeHostnames exist in contract")

	if len(edgeHostnames.EdgeHostnames.Items) == 0 {
		return fmt.Errorf("no default edge hostname found")
	}
	logger.Debug("Edgehostnames Default host %v", edgeHostnames.EdgeHostnames.Items[0])
	defaultEdgeHostname := edgeHostnames.EdgeHostnames.Items[0]

	var found bool
	var edgeHostnameID string
	edgeHostname, err := tools.GetStringValue("edge_hostname", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return err
	}
	if edgeHostname != "" {
		for _, hostname := range edgeHostnames.EdgeHostnames.Items {
			if hostname.EdgeHostnameDomain == edgeHostname {
				found = true
				defaultEdgeHostname = hostname
				edgeHostnameID = hostname.EdgeHostnameID
			}
		}
		logger.Debug("Found EdgeHostname %v", found)
		logger.Debug("Default EdgeHostname %v", defaultEdgeHostname)
	}

	if err := d.Set("contract", contract); err != nil {
		return fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error())
	}
	if err := d.Set("group", group); err != nil {
		return fmt.Errorf("%w: %s", tools.ErrValueSet, err.Error())
	}
	d.SetId(edgeHostnameID)
	return nil
}