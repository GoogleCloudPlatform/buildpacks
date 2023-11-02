// Copyright 2023 Google LLC
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

// Label defines the name and type of an additional field to be added to metrics.
type Label struct {
	Name      string    `json:"n,omitempty"`
	LabelType labelType `json:"l,omitempty"`
}

// labelType is used to indicate which type a Label carries.
type labelType int

const (
	// Bool indicates the type of a Label is a bool.
	Bool labelType = iota
	// Int indicates the type of a Label is an int.
	Int
	// String indicates the type of a Label is a string.
	String
)

var (
	supportedLabelTypes = []labelType{Bool, Int, String}
)

// Field defines an additional field to be attached to a metric.
type Field struct {
	Label Label `json:"l,omitempty"`
	Value any   `json:"v,omitempty"`
}

func typesMatch(label Label, field Field) bool {
	switch field.Value.(type) {
	case bool:
		return label.LabelType == Bool
	case int64:
		return label.LabelType == Int
	case string:
		return label.LabelType == String
	default:
		return false
	}
}

func fieldListsMatch(fl1 []Field, fl2 []Field) bool {
	if len(fl1) != len(fl2) {
		return false
	}

	seen1 := make(map[Field]int)
	seen2 := make(map[Field]int)
	for _, f1 := range fl1 {
		seen1[f1]++
	}
	for _, f2 := range fl2 {
		seen2[f2]++
	}

	if len(seen1) != len(seen2) {
		return false
	}

	for s1k, s1v := range seen1 {
		s2v, found := seen2[s1k]
		if !found || s1v != s2v {
			return false
		}
	}
	return true
}
