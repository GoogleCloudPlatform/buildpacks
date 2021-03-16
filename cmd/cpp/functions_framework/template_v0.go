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

package main

const mainTextTemplateV0 = `// Generated file. DO NOT EDIT.
// This file runs the the Functions Framework for C++ with a HTTP function.

#include <google/cloud/functions/framework.h>

{{if .Namespace}}namespace {{.Namespace}} { {{end}}
{{.Signature.ReturnType}} {{.ShortName}} ( {{.Signature.ArgumentType }} );
{{if .Namespace}} }  // namespace {{.Namespace}} {{end}}

int main(int argc, char* argv[]) {
  return google::cloud::functions::Run(
	    argc, argv, {{.Signature.WrapperType}} ( {{.Target}} ) );
}
`
