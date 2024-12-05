package apphostingschema

import (
	"fmt"
	"regexp"
	"slices"
)

var (
	// FullyQualifiedConnector is the regular expression for a full VPC Connector name.
	FullyQualifiedConnector = regexp.MustCompile(`^projects/[^/]+/locations/[^/]+/connectors/[^/]+$`)

	// ConnectorID is the regular expression for a VPC Connector ID.
	ConnectorID = regexp.MustCompile(`^[^/]+$`)

	// NetworkAddress is the regular expression for an IP address.
	NetworkAddress = regexp.MustCompile(`([0-9]{1,3}\.){3}[0-9]{1,3}`)

	// ValidEgress is the list of valid egress settings.
	ValidEgress = []string{"ALL_TRAFFIC", "PRIVATE_RANGES_ONLY"}
)

// ValidateVpcAccess validates the form of a vpcAccess struct.
func ValidateVpcAccess(vpcAccess *VpcAccess) error {
	if vpcAccess == nil {
		return nil
	}
	if vpcAccess.Egress != "" && !slices.Contains(ValidEgress, vpcAccess.Egress) {
		return fmt.Errorf("egress must be one of %v, got: %q", ValidEgress, vpcAccess.Egress)
	}
	if vpcAccess.Connector != "" && !FullyQualifiedConnector.MatchString(vpcAccess.Connector) && !ConnectorID.MatchString(vpcAccess.Connector) {
		return fmt.Errorf("connector must be fully qualified or an ID, got: %q", vpcAccess.Connector)
	}
	if vpcAccess.Connector == "" && len(vpcAccess.NetworkInterfaces) == 0 {
		return fmt.Errorf("one of connector or networkInterfaces must be set")
	}
	if vpcAccess.Connector != "" && len(vpcAccess.NetworkInterfaces) > 0 {
		return fmt.Errorf("connector and networkInterfaces cannot be set at the same time")
	}
	for _, ni := range vpcAccess.NetworkInterfaces {
		if ni.Network == "" && ni.Subnetwork == "" {
			return fmt.Errorf("at least one of network or subnetwork is required")
		}
		if ni.Network != "" && !NetworkAddress.MatchString(ni.Network) {
			return fmt.Errorf("network must be a network address, got: %q", ni.Network)
		}
		if ni.Subnetwork != "" && !NetworkAddress.MatchString(ni.Subnetwork) {
			return fmt.Errorf("subnetwork must be a network address, got: %q", ni.Subnetwork)
		}
	}
	return nil
}

// MergeVpcAccess merges the access from the YAML settings and the output bundle (though the output
// bundle is not expected to have any VPC access settings at the moment).
func MergeVpcAccess(yamlAccess, envAccess *VpcAccess) *VpcAccess {
	if yamlAccess == nil {
		return envAccess
	}
	if envAccess == nil {
		return yamlAccess
	}

	ret := &VpcAccess{}
	if envAccess.Egress != "" {
		ret.Egress = envAccess.Egress
	} else if yamlAccess.Egress != "" {
		ret.Egress = yamlAccess.Egress
	}

	// Note: connector and network interfaces are mutually exclusive, so we only copy one field from
	// one source.
	if envAccess.Connector != "" {
		ret.Connector = envAccess.Connector
	} else if envAccess.NetworkInterfaces != nil {
		ret.NetworkInterfaces = make([]NetworkInterface, len(envAccess.NetworkInterfaces))
		copy(ret.NetworkInterfaces, envAccess.NetworkInterfaces)
	} else if yamlAccess.Connector != "" {
		ret.Connector = yamlAccess.Connector
	} else if yamlAccess.NetworkInterfaces != nil {
		ret.NetworkInterfaces = make([]NetworkInterface, len(yamlAccess.NetworkInterfaces))
		copy(ret.NetworkInterfaces, yamlAccess.NetworkInterfaces)
	}

	return ret
}

// NormalizeVpcAccess ensures that any connector is a fully qualified resource name.
func NormalizeVpcAccess(vpcAccess *VpcAccess, project, region string) {
	if ConnectorID.MatchString(vpcAccess.Connector) {
		vpcAccess.Connector = fmt.Sprintf("projects/%s/locations/%s/connectors/%s", project, region, vpcAccess.Connector)
	}
}
