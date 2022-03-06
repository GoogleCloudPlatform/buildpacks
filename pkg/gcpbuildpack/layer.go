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

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/buildpacks/libcnb"
)

const (
	layerMode os.FileMode = 0755
)

type layerOption func(ctx *Context, l *libcnb.Layer) error

// BuildLayer specifies a Build layer.
var BuildLayer = func(ctx *Context, l *libcnb.Layer) error {
	l.Build = true
	return nil
}

// CacheLayer specifies a Cache layer.
var CacheLayer = func(ctx *Context, l *libcnb.Layer) error {
	l.Cache = true
	return nil
}

// LaunchLayer specifies a Launch layer.
var LaunchLayer = func(ctx *Context, l *libcnb.Layer) error {
	l.Launch = true
	return nil
}

// LaunchLayerIfDevMode specifies a Launch layer, but only if dev mode is enabled.
var LaunchLayerIfDevMode = func(ctx *Context, l *libcnb.Layer) error {
	devMode, err := env.IsDevMode()
	if err != nil {
		ctx.Warnf("Dev mode not enabled: %v", err)
		return nil
	}
	if devMode {
		l.Launch = true
	}
	return nil
}

// LaunchLayerUnlessSkipRuntimeLaunch specifies a Launch layer unless XGoogleSkipRuntimeLaunch is set to "true".
var LaunchLayerUnlessSkipRuntimeLaunch = func(ctx *Context, l *libcnb.Layer) error {
	skip, err := env.IsPresentAndTrue(env.XGoogleSkipRuntimeLaunch)
	if err != nil {
		return buildererror.Errorf(buildererror.StatusInternal, err.Error())
	}
	if !skip {
		l.Launch = true
	}
	return nil
}

// Layer returns a layer, creating its directory.
func (ctx *Context) Layer(name string, opts ...layerOption) (*libcnb.Layer, error) {
	l, err := ctx.buildContext.Layers.Layer(name)
	if err != nil {
		return nil, buildererror.Errorf(buildererror.StatusInternal, err.Error())
	}
	if err := ctx.MkdirAll(l.Path, layerMode); err != nil {
		return nil, buildererror.Errorf(buildererror.StatusInternal, "creating %s: %v", l.Path, err)
	}
	for _, o := range opts {
		if err := o(ctx, &l); err != nil {
			return nil, err
		}
	}
	if l.Metadata == nil {
		l.Metadata = make(map[string]interface{})
	}
	ctx.buildResult.Layers = append(ctx.buildResult.Layers, layerContributor{&l})
	return &l, nil
}

type layerContributor struct {
	l *libcnb.Layer
}

// Contribute accepts a layer and transforms it, returning a layer.
func (lc layerContributor) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	return *lc.l, nil
}

// Name is the name of the layer.
func (lc layerContributor) Name() string {
	return lc.l.Name
}

// ClearLayer erases the existing layer, and re-creates the directory.
func (ctx *Context) ClearLayer(l *libcnb.Layer) error {
	if err := ctx.RemoveAll(l.Path); err != nil {
		return err
	}
	if err := ctx.MkdirAll(l.Path, layerMode); err != nil {
		return err
	}
	return nil
}

// SetMetadata sets metadata on the layer.
func (ctx *Context) SetMetadata(l *libcnb.Layer, key, value string) {
	l.Metadata[key] = value
}

// GetMetadata gets metadata from the layer.
func (ctx *Context) GetMetadata(l *libcnb.Layer, key string) string {
	v, ok := l.Metadata[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		ctx.Exit(1, buildererror.Errorf(buildererror.StatusInternal, "could not cast metadata %v to string", v))
	}
	return s
}
