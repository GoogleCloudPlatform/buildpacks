// Copyright 2025 Google LLC
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

package lib

import (
	"testing"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		envs  []string
		files map[string]string
		want  int
	}{
		{
			name: "with spring boot dependency in pom",
			files: map[string]string{
				"pom.xml": `
		<project>
		  <dependencies>
		    <dependency>
		      <groupId>org.springframework.boot</groupId>
		      <artifactId>spring-boot-starter-web</artifactId>
		    </dependency>
		  </dependencies>
		</project>`,
			},
			envs: []string{"GOOGLE_RUNTIME_VERSION=25.0.0_36.0.LTS"},
			want: 0, // OptIn
		},
		{
			name: "with spring boot plugin in pom",
			files: map[string]string{
				"pom.xml": `
		<project>
		  <build>
		    <plugins>
		      <plugin>
		        <groupId>org.springframework.boot</groupId>
		        <artifactId>spring-boot-maven-plugin</artifactId>
		      </plugin>
		    </plugins>
		  </build>
		</project>`,
			},
			envs: []string{"GOOGLE_RUNTIME_VERSION=25.0.0_36.0.LTS"},
			want: 0, // OptIn
		},
		{
			name: "with spring boot plugin in pom but java version",
			files: map[string]string{
				"pom.xml": `
		<project>
		  <build>
		    <plugins>
		      <plugin>
		        <groupId>org.springframework.boot</group>
		        <artifactId>spring-boot-maven-plugin</artifactId>
		      </plugin>
		    </plugins>
		  </build>
		</project>`,
			},
			envs: []string{"GOOGLE_RUNTIME_VERSION=21.0.1"},
			want: 100, // OptOut
		},
		{
			name: "without spring boot dependency in pom",
			files: map[string]string{
				"pom.xml": "<project></project>",
			},
			envs: []string{"GOOGLE_RUNTIME_VERSION=25"},
			want: 100, // OptOut
		},
		{
			name:  "no pom",
			files: map[string]string{},
			envs:  []string{"GOOGLE_RUNTIME_VERSION=25"},
			want:  100, // OptOut
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, tc.envs, tc.want)
		})
	}
}
