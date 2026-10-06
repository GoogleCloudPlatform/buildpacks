package apphostingschema

import (
	"fmt"
	"regexp"
	"slices"
)

var (
	// fullyQualifiedConnector is the regular expression for a full VPC Connector name.
	fullyQualifiedConnector = regexp.MustCompile(`^projects/[^/]+/locations/[^/]+/connectors/[^/]+$`)

	// resourceID is the regular expression for a resource ID.
	resourceID = regexp.MustCompile(`^[^/]+$`)

	// fullyQualifiedNetwork is the regular expression for a fully qualified network name.
	fullyQualifiedNetwork = regexp.MustCompile(`^projects/[^/]+/global/networks/[^/]+$`)

	// fullyQualifiedSubnetwork is the regular expression for a fully qualified subnetwork name.
	fullyQualifiedSubnetwork = regexp.MustCompile(`^projects/[^/]+/regions/[^/]+/subnetworks/[^/]+$`)

	// validEgress is the list of valid egress settings.
	validEgress = []string{"ALL_TRAFFIC", "PRIVATE_RANGES_ONLY"}
)

// ValidateVpcAccess validates the form of a vpcAccess struct.
func ValidateVpcAccess(vpcAccess *VpcAccess) error {
	if vpcAccess == nil {
		return nil
	}
	if vpcAccess.Egress != "" && !slices.Contains(validEgress, vpcAccess.Egress) {
		return fmt.Errorf("egress must be one of %v, got: %q", validEgress, vpcAccess.Egress)
	}
	if vpcAccess.Connector != "" && !fullyQualifiedConnector.MatchString(vpcAccess.Connector) && !resourceID.MatchString(vpcAccess.Connector) {
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
		if ni.Network != "" && !fullyQualifiedNetwork.MatchString(ni.Network) && !resourceID.MatchString(ni.Network) {
			return fmt.Errorf("network must be fully qualified or an ID, got: %q", ni.Network)
		}
		if ni.Subnetwork != "" && !fullyQualifiedSubnetwork.MatchString(ni.Subnetwork) && !resourceID.MatchString(ni.Subnetwork) {
			return fmt.Errorf("subnetwork must be fully qualified or an ID, got: %q", ni.Subnetwork)
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
	if vpcAccess == nil {
		return
	}

	if resourceID.MatchString(vpcAccess.Connector) {
		vpcAccess.Connector = fmt.Sprintf("projects/%s/locations/%s/connectors/%s", project, region, vpcAccess.Connector)
	}
	// N.B. range returns copies, so editing the value directly would not affect the original.
	ni := vpcAccess.NetworkInterfaces
	for x := range ni {
		if resourceID.MatchString(ni[x].Network) {
			ni[x].Network = fmt.Sprintf("projects/%s/global/networks/%s", project, ni[x].Network)
		}
		if resourceID.MatchString(ni[x].Subnetwork) {
			ni[x].Subnetwork = fmt.Sprintf("projects/%s/regions/%s/subnetworks/%s", project, region, ni[x].Subnetwork)
		}
	}
}
