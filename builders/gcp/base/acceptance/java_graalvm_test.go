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
package acceptance

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptanceFFJavaGraalVM(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:           "Standard GCF application",
			App:            "java/graalvm/functions_framework",
			Env:            []string{"GOOGLE_JAVA_USE_NATIVE_IMAGE=true", "GOOGLE_FUNCTION_TARGET=functions.HelloWorld"},
			MustUse:        []string{javaGraalVM, javaMaven, javaFF, javaNativeImage},
			FilesMustExist: []string{"/layers/google.java.native-image/native-image/bin/native-app"}},
	}
	for _, tc := range testCases {
		tc := tc

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, builder, tc)
		})
	}
}
