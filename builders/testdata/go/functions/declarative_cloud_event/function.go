// Copyright 2021 Google LLC
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

// Package fn tests a declarative CloudEvent function.
package fn

import (
	"context"
	"errors"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

func init() {
	functions.CloudEvent("Func", cloudEventFunc)
	functions.CloudEvent("OtherFunc", cloudEventFunc)
}

func cloudEventFunc(ctx context.Context, e cloudevents.Event) error {
	var data string
	if err := e.DataAs(&data); err != nil || len(data) == 0 {
		return errors.New("FAIL")
	}

	return nil
}

// NonDeclarativeFunc is a CloudEvent function that is not declaratively
// registered.
func NonDeclarativeFunc(ctx context.Context, e cloudevents.Event) error {
	return cloudEventFunc(ctx, e)
}

func someOtherFunc(ctx context.Context, e cloudevents.Event) error {
	return errors.New("FAIL")
}
