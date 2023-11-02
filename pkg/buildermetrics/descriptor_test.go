// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package buildermetrics

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidateDescriptor(t *testing.T) {
	testCases := []struct {
		name       string
		descriptor Descriptor
		wantFail   bool
	}{
		{
			name: "correct descriptor",
			descriptor: newDescriptor(
				ArNpmCredsGenCounterID,
				"npm_artifact_registry_creds_generated",
				"The number of artifact registry credentials generated for NPM",
			),
		},
		{
			name: "no name",
			descriptor: newDescriptor(
				ArNpmCredsGenCounterID,
				"",
				"The number of artifact registry credentials generated for NPM",
			),
			wantFail: true,
		},
		{
			name: "no description",
			descriptor: newDescriptor(
				ArNpmCredsGenCounterID,
				"npm_artifact_registry_creds_generated",
				"",
			),
			wantFail: true,
		},
		{
			name: "no id",
			descriptor: newDescriptor(
				MetricID(""),
				"npm_artifact_registry_creds_generated",
				"The number of artifact registry credentials generated for NPM",
			),
			wantFail: true,
		},
		{
			name: "mismatched ids",
			descriptor: newDescriptor(
				ComposerInstallLatencyID,
				"npm_artifact_registry_creds_generated",
				"The number of artifact registry credentials generated for NPM",
			),
			wantFail: true,
		},
		{
			name: "correct descriptor with label",
			descriptor: newDescriptor(
				ArNpmCredsGenCounterID,
				"npm_artifact_registry_creds_generated",
				"The number of artifact registry credentials generated for NPM",
				Label{Name: "cacheHit", LabelType: Bool},
				Label{Name: "somethingElse", LabelType: Int},
			),
		},
		{
			name: "label named value",
			descriptor: newDescriptor(
				ArNpmCredsGenCounterID,
				"npm_artifact_registry_creds_generated",
				"The number of artifact registry credentials generated for NPM",
				Label{Name: "value", LabelType: Bool},
			),
			wantFail: true,
		},
		{
			name: "two labels same name",
			descriptor: newDescriptor(
				ArNpmCredsGenCounterID,
				"npm_artifact_registry_creds_generated",
				"The number of artifact registry credentials generated for NPM",
				Label{Name: "cacheHit", LabelType: Bool},
				Label{Name: "somethingElse", LabelType: Int},
				Label{Name: "cacheHit", LabelType: String},
			),
			wantFail: true,
		},
		{name: "bad label type",
			descriptor: newDescriptor(
				ArNpmCredsGenCounterID,
				"npm_artifact_registry_creds_generated",
				"The number of artifact registry credentials generated for NPM",
				Label{Name: "cacheHit", LabelType: -1},
			),
			wantFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// We're just testing validateDescriptor here, so we'll always use ArNpmCredsGenCounterID.
			// except for the registered ID test.
			err := validateDescriptor(ArNpmCredsGenCounterID, tc.descriptor)
			if tc.wantFail && err == nil {
				t.Errorf("%v: expected error, none returned", tc.name)
			}
			if !tc.wantFail && err != nil {
				t.Errorf("%v: expected no error, but failed with error: %v", tc.name, err)
			}
		})
	}
}

func TestDescriptorEqual(t *testing.T) {
	id1 := MetricID("1")
	id2 := MetricID("2")
	n1 := "name1"
	n2 := "name2"
	d1 := "description1"
	d2 := "description2"
	l1 := Label{Name: "label1", LabelType: Bool}
	l2 := Label{Name: "label2", LabelType: Int}
	l3NameMismatchA := Label{Name: "label4_A", LabelType: String}
	l3NameMismatchB := Label{Name: "label4_B", LabelType: String}
	l4TypeMismatchA := Label{Name: "label3", LabelType: String}
	l4TypeMismatchB := Label{Name: "label3", LabelType: Int}
	testCases := []struct {
		name string
		d1   Descriptor
		d2   Descriptor
		want bool
	}{
		{
			name: "equal, labels in order",
			d1:   newDescriptor(id1, n1, d1, l1, l2, l3NameMismatchA),
			d2:   newDescriptor(id1, n1, d1, l1, l2, l3NameMismatchA),
			want: true,
		},
		{
			name: "equal, labels in different order, sorter works for label name",
			d1:   newDescriptor(id1, n1, d1, l1, l2, l3NameMismatchA, l3NameMismatchB),
			d2:   newDescriptor(id1, n1, d1, l3NameMismatchB, l3NameMismatchA, l2, l1),
			want: true,
		},
		{
			name: "equal, labels in different order, sorter works for name and type",
			d1:   newDescriptor(id1, n1, d1, l1, l2, l4TypeMismatchA, l4TypeMismatchB),
			d2:   newDescriptor(id1, n1, d1, l4TypeMismatchB, l4TypeMismatchA, l2, l1),
			want: true,
		},
		{
			name: "id not equal",
			d1:   newDescriptor(id1, n1, d1, l1, l2, l4TypeMismatchA),
			d2:   newDescriptor(id2, n1, d1, l1, l2, l4TypeMismatchA),
			want: false,
		},
		{
			name: "name not equal",
			d1:   newDescriptor(id1, n1, d1, l1, l2, l4TypeMismatchA),
			d2:   newDescriptor(id1, n2, d1, l1, l2, l4TypeMismatchA),
			want: false,
		},
		{
			name: "description not equal",
			d1:   newDescriptor(id1, n1, d1, l1, l2, l4TypeMismatchA),
			d2:   newDescriptor(id1, n1, d2, l1, l2, l4TypeMismatchA),
			want: false,
		},
		{
			name: "labels not equal",
			d1:   newDescriptor(id1, n1, d1, l1, l2, l4TypeMismatchA),
			d2:   newDescriptor(id1, n1, d1, l1, l2, l4TypeMismatchB),
			want: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.d1.equal(tc.d2) != tc.want {
				t.Errorf("%v: equal(%v, %v) = %v, want %v", tc.name, tc.d1, tc.d2, tc.want, tc.d1)
			}
		})
	}
}

// validateDescriptor is used to validate the descriptors in the descriptors table.
func validateDescriptor(descriptorsTableID MetricID, d Descriptor) error {
	if d.ID == "" {
		return fmt.Errorf("Descriptor with missing ID: %v", d)
	}
	if d.Name == "" {
		return fmt.Errorf("Descriptor with missing Name name: %v", d)
	}
	if d.Description == "" {
		return fmt.Errorf("Descriptor with missing description: %v", d)
	}

	found := false
	for _, descriptor := range descriptors {
		if d.Name == descriptor.Name {
			if found {
				return fmt.Errorf("the descriptor name %q is registered in the descriptors table for more than one metric", d.Name)
			}
			found = true
		}
	}
	if !found {
		return fmt.Errorf("the descriptor %q is not registered in the descriptors table", d.Name)
	}

	if d.ID != descriptorsTableID {
		return fmt.Errorf("Descriptor %q must have an ID matching the corresponding metric ID in the descriptors table", d.Name)
	}

	for _, l1 := range d.Labels {
		if strings.TrimSpace(strings.ToLower(l1.Name)) == "value" {
			return fmt.Errorf("Descriptor labels may not be called value")
		}

		foundOne := false
		for _, l2 := range d.Labels {
			if l1.Name == l2.Name {
				if foundOne {
					return fmt.Errorf("found multiple labels named %v. Label names must be unique", l1.Name)
				}
				foundOne = true
			}
		}

		foundType := false
		for _, supportedType := range supportedLabelTypes {
			if l1.LabelType == supportedType {
				foundType = true
				break
			}
		}
		if !foundType {
			return fmt.Errorf("Descriptor label %q must be one of the supported types", l1.LabelType)
		}
	}
	return nil
}
