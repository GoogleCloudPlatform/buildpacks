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

package cloudfunctions

import "github.com/GoogleCloudPlatform/buildpacks/pkg/env"

// SkipFrameworkInjection is used to allow opting out of Functions Framework auto-injection
// when it hasn't been explicitly declared as a dependency.
const SkipFrameworkInjection = "GOOGLE_SKIP_FRAMEWORK_INJECTION"

// IsSkipFrameworkInjectionEnabled returns true if skipping Functions Framework injection is enabled.
func IsSkipFrameworkInjectionEnabled() (bool, error) {
	return env.IsPresentAndTrue(SkipFrameworkInjection)
}
