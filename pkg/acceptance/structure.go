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

// StructureTest describes verifications on a container image.
type StructureTest struct {
	SchemaVersion      string              `yaml:"schemaVersion"`
	MetadataTest       metadataTest        `yaml:"metadataTest"`
	FileExistenceTests []fileExistenceTest `yaml:"fileExistenceTests"`
}

// metadataTest verifies the image's metadata.
type metadataTest struct {
	Env          []envVar `yaml:"env"`
	ExposedPorts []string `yaml:"exposedPorts"`
	Entrypoint   []string `yaml:"entrypoint"`
	Cmd          []string `yaml:"cmd"`
	Workdir      string   `yaml:"workdir"`
}

// fileExistenceTest verifies the existence of a file.
type fileExistenceTest struct {
	Name        string `yaml:"name"`
	Path        string `yaml:"path"`
	ShouldExist bool   `yaml:"shouldExist"`
	UID         int    `yaml:"uid"`
	GID         int    `yaml:"gid"`
}

// envVar tests for the existence of an environment variable.
type envVar struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// NewStructureTest creates a new StructureTest. It returns nil if there's nothing to check.
func NewStructureTest(filesMustExist, filesMustNotExist []string) *StructureTest {
	if len(filesMustExist) == 0 && len(filesMustNotExist) == 0 {
		return nil
	}

	var fts []fileExistenceTest
	for _, file := range filesMustExist {
		fts = append(fts, fileExistenceTest{
			Name:        file,
			Path:        file,
			ShouldExist: true,
			UID:         1000,
			GID:         1000,
		})
	}
	for _, file := range filesMustNotExist {
		fts = append(fts, fileExistenceTest{
			Name:        file,
			Path:        file,
			ShouldExist: false,
			UID:         -1, // -1 means "ignore"
			GID:         -1, // -1 means "ignore"
		})
	}

	return &StructureTest{
		SchemaVersion:      "2.0.0",
		FileExistenceTests: fts,
	}
}
