package apphostingschema

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestValidateVpcAccess(t *testing.T) {
	tests := []struct {
		desc      string
		vpcAccess *VpcAccess
		wantErr   bool
	}{
		{
			desc: "valid connector name and egress",
			vpcAccess: &VpcAccess{
				Connector: "projects/project-id/locations/us-central1/connectors/my-connector",
				Egress:    "ALL_TRAFFIC",
			},
		},
		{
			desc: "valid connector id",
			vpcAccess: &VpcAccess{
				Connector: "my-connector",
			},
		},
		{
			desc: "valid network interface",
			vpcAccess: &VpcAccess{
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "projects/project-id/global/networks/test-network",
						Subnetwork: "projects/project-id/regions/us-central1/subnetworks/test-subnetwork",
						Tags:       []string{"test-tag"},
					},
				},
			},
		},
		{
			desc: "invalid egress",
			vpcAccess: &VpcAccess{
				Connector: "my-connector",
				Egress:    "INVALID_EGRESS",
			},
			wantErr: true,
		},
		{
			desc: "invalid connector name",
			vpcAccess: &VpcAccess{
				Connector: "locations/us-central1/connectors/my-connector/foo",
			},
			wantErr: true,
		},
		{
			desc: "invalid network interface",
			vpcAccess: &VpcAccess{
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "invalid/network",
						Subnetwork: "projects/project-id/regions/us-central1/subnetworks/test-subnetwork",
					},
				},
			},
			wantErr: true,
		},
		{
			desc: "invalid subnetwork interface",
			vpcAccess: &VpcAccess{
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "projects/project-id/global/networks/test-network",
						Subnetwork: "invalid/subnetwork",
					},
				},
			},
			wantErr: true,
		},
		{
			desc: "connector and network interfaces cannot be set at the same time",
			vpcAccess: &VpcAccess{
				Connector: "my-connector",
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "network",
						Subnetwork: "subnetwork",
					},
				},
			},
			wantErr: true,
		},
		{
			desc: "at least network or subnetwork is required",
			vpcAccess: &VpcAccess{
				NetworkInterfaces: []NetworkInterface{
					{
						Tags: []string{"tag1"},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := ValidateVpcAccess(tc.vpcAccess)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateVpcAccess(%v) returned %v, wantErr %v", tc.vpcAccess, err, tc.wantErr)
			}
		})
	}
}

func TestMergeVpcAccess(t *testing.T) {
	tests := []struct {
		desc          string
		yamlAccess    *VpcAccess
		envAccess     *VpcAccess
		wantVpcAccess *VpcAccess
	}{
		{
			desc: "yaml access is nil",
			envAccess: &VpcAccess{
				Egress: "ALL_TRAFFIC",
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "network",
						Subnetwork: "subnetwork",
					},
				},
			},
			wantVpcAccess: &VpcAccess{
				Egress: "ALL_TRAFFIC",
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "network",
						Subnetwork: "subnetwork",
					},
				},
			},
		},
		{
			desc: "env access is nil",
			yamlAccess: &VpcAccess{
				Egress: "ALL_TRAFFIC",
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "network",
						Subnetwork: "subnetwork",
					},
				},
			},
			wantVpcAccess: &VpcAccess{
				Egress: "ALL_TRAFFIC",
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "network",
						Subnetwork: "subnetwork",
					},
				},
			},
		},
		{
			desc:          "both access are nil",
			wantVpcAccess: nil,
		},
		{
			desc: "env overrides egress settings",
			yamlAccess: &VpcAccess{
				Connector: "my-connector",
				Egress:    "ALL_TRAFFIC",
			},
			envAccess: &VpcAccess{
				Connector: "my-connector",
				Egress:    "PRIVATE_RANGES_ONLY",
			},
			wantVpcAccess: &VpcAccess{
				Connector: "my-connector",
				Egress:    "PRIVATE_RANGES_ONLY",
			},
		},
		{
			desc: "env overrides network interfaces",
			yamlAccess: &VpcAccess{
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "network",
						Subnetwork: "subnetwork",
						Tags:       []string{"tag1"},
					},
				},
			},
			envAccess: &VpcAccess{
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "network2",
						Subnetwork: "subnetwork2",
						Tags:       []string{"tag2"},
					},
				},
			},
			wantVpcAccess: &VpcAccess{
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "network2",
						Subnetwork: "subnetwork2",
						Tags:       []string{"tag2"},
					},
				},
			},
		},
		{
			desc: "env overrides connector",
			yamlAccess: &VpcAccess{
				Connector: "my-connector",
			},
			envAccess: &VpcAccess{
				Connector: "my-connector-2",
			},
			wantVpcAccess: &VpcAccess{
				Connector: "my-connector-2",
			},
		},
		{
			desc: "env overrides connector vs network interfaces",
			yamlAccess: &VpcAccess{
				Connector: "my-connector",
			},
			envAccess: &VpcAccess{
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "network",
						Subnetwork: "subnetwork",
						Tags:       []string{"tag2"},
					},
				},
			},
			wantVpcAccess: &VpcAccess{
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "network",
						Subnetwork: "subnetwork",
						Tags:       []string{"tag2"},
					},
				},
			},
		},
		{
			desc: "env overrides netowrk interface vs connector",
			yamlAccess: &VpcAccess{
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "network",
						Subnetwork: "subnetwork",
						Tags:       []string{"tag1"},
					},
				},
			},
			envAccess: &VpcAccess{
				Connector: "my-connector",
			},
			wantVpcAccess: &VpcAccess{
				Connector: "my-connector",
			},
		},
		{
			desc: "merge case",
			yamlAccess: &VpcAccess{
				Connector: "my-connector",
				Egress:    "ALL_TRAFFIC",
			},
			envAccess: &VpcAccess{
				Connector: "my-test-connector",
			},
			wantVpcAccess: &VpcAccess{
				Connector: "my-test-connector",
				Egress:    "ALL_TRAFFIC",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			gotVpcAccess := MergeVpcAccess(tc.yamlAccess, tc.envAccess)
			if diff := cmp.Diff(tc.wantVpcAccess, gotVpcAccess); diff != "" {
				t.Errorf("MergeVpcAccess(%v, %v) returned unexpected diff (-want +got):\n%s", tc.yamlAccess, tc.envAccess, diff)
			}
		})
	}
}

func TestNormalizeVpcAccess(t *testing.T) {
	tests := []struct {
		desc          string
		vpcAccess     *VpcAccess
		project       string
		region        string
		wantVpcAccess *VpcAccess
	}{
		{
			desc: "connector id is normalized",
			vpcAccess: &VpcAccess{
				Connector: "my-connector",
			},
			project: "project-id",
			region:  "us-central1",
			wantVpcAccess: &VpcAccess{
				Connector: "projects/project-id/locations/us-central1/connectors/my-connector",
			},
		},
		{
			desc: "connector name is not normalized",
			vpcAccess: &VpcAccess{
				Connector: "projects/project-id/locations/us-central1/connectors/my-connector",
			},
			project: "project-id",
			region:  "us-central1",
			wantVpcAccess: &VpcAccess{
				Connector: "projects/project-id/locations/us-central1/connectors/my-connector",
			},
		},
		{
			desc: "network interface id is normalized",
			vpcAccess: &VpcAccess{
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "test-network",
						Subnetwork: "test-subnetwork",
					},
				},
			},
			project: "project-id",
			region:  "us-central1",
			wantVpcAccess: &VpcAccess{
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "projects/project-id/global/networks/test-network",
						Subnetwork: "projects/project-id/regions/us-central1/subnetworks/test-subnetwork",
					},
				},
			},
		},
		{
			desc: "network interface is not normalized",
			vpcAccess: &VpcAccess{
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "projects/project-id/global/networks/test-network",
						Subnetwork: "projects/project-id/regions/us-central1/subnetworks/test-subnetwork",
					},
				},
			},
			project: "project-id",
			region:  "us-central1",
			wantVpcAccess: &VpcAccess{
				NetworkInterfaces: []NetworkInterface{
					{
						Network:    "projects/project-id/global/networks/test-network",
						Subnetwork: "projects/project-id/regions/us-central1/subnetworks/test-subnetwork",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			NormalizeVpcAccess(tc.vpcAccess, tc.project, tc.region)
			if diff := cmp.Diff(tc.wantVpcAccess, tc.vpcAccess); diff != "" {
				t.Errorf("NormalizeVpcAccess(%v) returned unexpected diff (-want +got):\n%s", tc.vpcAccess, diff)
			}
		})
	}
}
