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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFieldListsMatch(t *testing.T) {
	l1 := Label{Name: "l1", LabelType: String}
	l2 := Label{Name: "l2", LabelType: Bool}
	l3 := Label{Name: "l3", LabelType: Int}
	l4 := Label{Name: "l4", LabelType: String}
	l5 := Label{Name: "l5", LabelType: Bool}
	l6 := Label{Name: "l6", LabelType: Int}
	f1 := Field{Label: l1, Value: "s1"}
	f2 := Field{Label: l2, Value: true}
	f3 := Field{Label: l3, Value: 42}
	f4 := Field{Label: l4, Value: "s2"}
	f5 := Field{Label: l5, Value: false}
	f6 := Field{Label: l6, Value: 43}
	testCases := []struct {
		name string
		fl1  []Field
		fl2  []Field
		want bool
	}{
		{
			name: "all matching",
			fl1:  []Field{f1, f2, f3, f4, f5, f6},
			fl2:  []Field{f1, f2, f3, f4, f5, f6},
			want: true,
		},
		{
			name: "all matching out of order",
			fl1:  []Field{f1, f2, f3, f4, f5, f6},
			fl2:  []Field{f6, f5, f4, f3, f2, f1},
			want: true,
		},
		{
			name: "missing a field",
			fl1:  []Field{f1, f2, f3, f4, f5, f6},
			fl2:  []Field{f6, f5, f4, f2, f1},
			want: false,
		},
		{
			name: "extra field",
			fl1:  []Field{f1, f3, f4, f5, f6},
			fl2:  []Field{f6, f5, f4, f3, f2, f1},
			want: false,
		},
		{
			name: "left empty",
			fl1:  nil,
			fl2:  []Field{f1, f2, f3, f4, f5, f6},
			want: false,
		},
		{
			name: "right empty",
			fl1:  []Field{f1, f2, f3, f4, f5, f6},
			fl2:  nil,
			want: false,
		},
		{
			name: "both empty",
			fl1:  nil,
			fl2:  nil,
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := fieldListsMatch(tc.fl1, tc.fl2)
			if got != tc.want {
				diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(BuilderMetrics{}, Counter{}, FloatDP{}))
				t.Errorf("got: %v, want: %v, diff:\n%s", got, tc.want, diff)
			}
		})
	}
}
