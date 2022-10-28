// Copyright 2022 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package acceptance

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptance(t *testing.T) {
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	// Tests of the entrypoints provided in
	// https://cloud.google.com/appengine/docs/flexible/python/runtime#application_startup
	testCases := []acceptance.Test{
		{
			Name: "gunicorn with flask entrypoint",
			App:  "gunicorn_flask_entrypoint",
			Env:  []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
		},

		{
			Name: "python script entrypoint",
			App:  "python_script",
			Env:  []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
		},

		{
			Name: "gunicorn with django entrypoint",
			App:  "gunicorn_django_entrypoint",
			Env:  []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
		},

		{
			Name: "uwsgi with flask",
			App:  "uwsgi_flask",
			Env:  []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
		},
	}

	for _, tc := range testCases {

		// returns a copy of the struct
		// need this for parallelization otherwise it will build the same name
		tc := applyStaticTestOptions(tc)

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, builderImage, runImage, tc)
		})
	}
}

func applyStaticTestOptions(tc acceptance.Test) acceptance.Test {
	if tc.Name == "" {
		tc.Name = tc.App
	}
	return tc
}
