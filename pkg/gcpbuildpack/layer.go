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

package gcpbuildpack

import (
	"os"

	"github.com/buildpack/libbuildpack/layers"
)

const (
	layerMode os.FileMode = 0755
)

// Layer returns a layer, creating its directory.
func (ctx *Context) Layer(name string) *layers.Layer {
	l := ctx.b.Layers.Layer(name)
	ctx.MkdirAll(l.Root, layerMode)
	return &l
}

// ClearLayer erases the existing layer, and re-creates the directory.
func (ctx *Context) ClearLayer(l *layers.Layer) {
	ctx.RemoveAll(l.Root)
	ctx.MkdirAll(l.Root, layerMode)
}
