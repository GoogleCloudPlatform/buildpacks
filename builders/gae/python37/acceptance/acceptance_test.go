// Copyright 2020 Google LLC
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
package acceptance

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptance(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			App: "no_requirements_txt",
		},
		{
			App: "requirements_txt",
		},
		{
			App: "requirements_bin_conflict",
		},
		{
			App: "pip_dependency",
		},
		{
			App: "pip_check",
			// The warning message is cut short because it's not deterministic.
			MustOutput: []string{`WARNING: Found incompatible dependencies: "sub-dependency-`},
		},
		{
			App: "gunicorn_present",
		},
		{
			App: "gunicorn_outdated",
		},
		{
			App: "custom_entrypoint",
			Env: []string{"GOOGLE_ENTRYPOINT=uwsgi --http :$PORT --wsgi-file custom.py --callable app"},
		},
		{
			Name: "custom gunicorn entrypoint",
			App:  "gunicorn_present",
			Env:  []string{"GOOGLE_ENTRYPOINT=gunicorn main:app"},
		},
	}
	for _, tc := range testCases {
		tc := tc
		if tc.Name == "" {
			tc.Name = tc.App
		}
		tc.Env = append(tc.Env, "GOOGLE_RUNTIME=python37")

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, builder, tc)
		})
	}
}
