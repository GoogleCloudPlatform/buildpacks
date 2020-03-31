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

	"github.com/GoogleCloudPlatform/buildpacks/pkg/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptance(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name: "cs no deps",
			App:  "cs_no_deps",
		},
		{
			Name: "cs AssemblyName set",
			App:  "cs_assemblyname",
		},
		{
			Name: "cs with dep",
			App:  "cs_with_dep",
		},
		{
			Name: "cs custom project file",
			App:  "cs_nested_proj",
			Env:  []string{"GOOGLE_BUILDABLE=app/app.csproj"},
		},
		{
			Name: "cs custom project file from GAE",
			App:  "cs_nested_proj",
			Env:  []string{"GAE_YAML_MAIN=app/app.csproj"},
		},
		{
			Name: "cs custom project file precedence",
			App:  "cs_nested_proj",
			Env:  []string{"GAE_YAML_MAIN=nonexistent/app.csproj", "GOOGLE_BUILDABLE=app/app.csproj"},
		},
		{
			Name: "cs custom project with proj in name",
			App:  "cs_nested_proj",
			Env:  []string{"GOOGLE_BUILDABLE=myproj"},
		},
		{
			Name: "cs custom entrypoint",
			App:  "cs_custom_entrypoint",
			Env:  []string{"GOOGLE_ENTRYPOINT=bin/app --flag=myflag"},
		},
		{
			Name: "prebuilt app",
			App:  "prebuilt",
			Env:  []string{"GOOGLE_ENTRYPOINT=./cs_no_deps"},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env, "GOOGLE_RUNTIME=dotnet3")

			acceptance.TestApp(t, builder, tc)
		})
	}
}
