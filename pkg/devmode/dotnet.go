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

package devmode

// DotNetSyncRules is the list of SyncRules to be configured in Dev Mode for .NET.
func DotNetSyncRules(dest string) []SyncRule {
	return []SyncRule{
		{Src: "**/*.cs", Dest: dest},
		{Src: "*.csproj", Dest: dest},
		{Src: "**/*.fs", Dest: dest},
		{Src: "*.fsproj", Dest: dest},
		{Src: "**/*.vb", Dest: dest},
		{Src: "*.vbproj", Dest: dest},
		{Src: "**/*.resx", Dest: dest},
	}
}
