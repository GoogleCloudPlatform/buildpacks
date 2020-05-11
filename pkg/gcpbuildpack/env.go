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

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/buildpack/libbuildpack/layers"
)

// SetFunctionsEnvVars sets launch-time functions environment variables.
func (ctx *Context) SetFunctionsEnvVars(l *layers.Layer) {
	if target := os.Getenv(env.FunctionTarget); target != "" {
		ctx.DefaultLaunchEnv(l, env.FunctionTargetLaunch, target)
	} else {
		ctx.Exit(1, UserErrorf("required env var %s not found", env.FunctionTarget))
	}

	if signature, ok := os.LookupEnv(env.FunctionSignatureType); ok {
		ctx.DefaultLaunchEnv(l, env.FunctionSignatureTypeLaunch, signature)
	}

	if source, ok := os.LookupEnv(env.FunctionSource); ok {
		ctx.DefaultLaunchEnv(l, env.FunctionSourceLaunch, source)
	}
}

// AppendBuildEnv appends the value of this environment variable to any previous declarations of the value without any
// delimitation.  If delimitation is important during concatenation, callers are required to add it.
func (ctx *Context) AppendBuildEnv(l *layers.Layer, name string, format string, args ...interface{}) {
	if err := l.AppendBuildEnv(name, format, args...); err != nil {
		ctx.Exit(1, InternalErrorf("appending build env var %s: %v", name, err))
	}
}

// AppendLaunchEnv appends the value of this environment variable to any previous declarations of the value without any
// delimitation.  If delimitation is important during concatenation, callers are required to add it.
func (ctx *Context) AppendLaunchEnv(l *layers.Layer, name string, format string, args ...interface{}) {
	if err := l.AppendLaunchEnv(name, format, args...); err != nil {
		ctx.Exit(1, InternalErrorf("appending launch env var %s: %v", name, err))
	}
}

// AppendSharedEnv appends the value of this environment variable to any previous declarations of the value without any
// delimitation.  If delimitation is important during concatenation, callers are required to add it.
func (ctx *Context) AppendSharedEnv(l *layers.Layer, name string, format string, args ...interface{}) {
	if err := l.AppendSharedEnv(name, format, args...); err != nil {
		ctx.Exit(1, InternalErrorf("appending shared env var %s: %v", name, err))
	}
}

// DefaultBuildEnv sets a default for an environment variable with this value.
func (ctx *Context) DefaultBuildEnv(l *layers.Layer, name string, format string, args ...interface{}) {
	if err := l.DefaultBuildEnv(name, format, args...); err != nil {
		ctx.Exit(1, InternalErrorf("setting default build env var %s: %v", name, err))
	}
}

// DefaultLaunchEnv sets a default for an environment variable with this value.
func (ctx *Context) DefaultLaunchEnv(l *layers.Layer, name string, format string, args ...interface{}) {
	if err := l.DefaultLaunchEnv(name, format, args...); err != nil {
		ctx.Exit(1, InternalErrorf("setting default launch env var %s: %v", name, err))
	}
}

// DefaultSharedEnv sets a default for an environment variable with this value.
func (ctx *Context) DefaultSharedEnv(l *layers.Layer, name string, format string, args ...interface{}) {
	if err := l.DefaultSharedEnv(name, format, args...); err != nil {
		ctx.Exit(1, InternalErrorf("setting default shared env var %s: %v", name, err))
	}
}

// DelimiterBuildEnv sets a delimiter for an environment variable with this value.
func (ctx *Context) DelimiterBuildEnv(l *layers.Layer, name string, delimiter string) {
	if err := l.DelimiterBuildEnv(name, delimiter); err != nil {
		ctx.Exit(1, InternalErrorf("setting build env var delimiter %s: %v", name, err))
	}
}

// DelimiterLaunchEnv sets a delimiter for an environment variable with this value.
func (ctx *Context) DelimiterLaunchEnv(l *layers.Layer, name string, delimiter string) {
	if err := l.DelimiterLaunchEnv(name, delimiter); err != nil {
		ctx.Exit(1, InternalErrorf("setting launch env var delimiter %s: %v", name, err))
	}
}

// DelimiterSharedEnv sets a delimiter for an environment variable with this value.
func (ctx *Context) DelimiterSharedEnv(l *layers.Layer, name string, delimiter string) {
	if err := l.DelimiterSharedEnv(name, delimiter); err != nil {
		ctx.Exit(1, InternalErrorf("setting shared env var delimiter %s: %v", name, err))
	}
}

// OverrideBuildEnv overrides any existing value for an environment variable with this value.
func (ctx *Context) OverrideBuildEnv(l *layers.Layer, name string, format string, args ...interface{}) {
	if err := l.OverrideBuildEnv(name, format, args...); err != nil {
		ctx.Exit(1, InternalErrorf("overriding build env var %s: %v", name, err))
	}
}

// OverrideLaunchEnv overrides any existing value for an environment variable with this value.
func (ctx *Context) OverrideLaunchEnv(l *layers.Layer, name string, format string, args ...interface{}) {
	if err := l.OverrideLaunchEnv(name, format, args...); err != nil {
		ctx.Exit(1, InternalErrorf("overriding launch env var %s: %v", name, err))
	}
}

// OverrideSharedEnv overrides any existing value for an environment variable with this value.
func (ctx *Context) OverrideSharedEnv(l *layers.Layer, name string, format string, args ...interface{}) {
	if err := l.OverrideSharedEnv(name, format, args...); err != nil {
		ctx.Exit(1, InternalErrorf("overriding shared env var %s: %v", name, err))
	}
}

// PrependBuildEnv prepends the value of this environment variable to any previous declarations of the value without any
// delimitation.  If delimitation is important during concatenation, callers are required to add it.
func (ctx *Context) PrependBuildEnv(l *layers.Layer, name string, format string, args ...interface{}) {
	if err := l.PrependBuildEnv(name, format, args...); err != nil {
		ctx.Exit(1, InternalErrorf("prepending build env var %s: %v", name, err))
	}
}

// PrependLaunchEnv prepends the value of this environment variable to any previous declarations of the value without
// any delimitation.  If delimitation is important during concatenation, callers are required to add it.
func (ctx *Context) PrependLaunchEnv(l *layers.Layer, name string, format string, args ...interface{}) {
	if err := l.PrependLaunchEnv(name, format, args...); err != nil {
		ctx.Exit(1, InternalErrorf("prepending launch env var %s: %v", name, err))
	}
}

// PrependSharedEnv prepends the value of this environment variable to any previous declarations of the value without
// any delimitation.  If delimitation is important during concatenation, callers are required to add it.
func (ctx *Context) PrependSharedEnv(l *layers.Layer, name string, format string, args ...interface{}) {
	if err := l.PrependSharedEnv(name, format, args...); err != nil {
		ctx.Exit(1, InternalErrorf("prepending shared env var %s: %v", name, err))
	}
}

// PrependPathBuildEnv prepends the value of this environment variable to any previous declarations of the value using
// the OS path delimiter.
func (ctx *Context) PrependPathBuildEnv(l *layers.Layer, name string, format string, args ...interface{}) {
	if err := l.PrependPathBuildEnv(name, format, args...); err != nil {
		ctx.Exit(1, InternalErrorf("prepending build path env var %s: %v", name, err))
	}
}

// PrependPathLaunchEnv prepends the value of this environment variable to any previous declarations of the value using
// the OS path delimiter.
func (ctx *Context) PrependPathLaunchEnv(l *layers.Layer, name string, format string, args ...interface{}) {
	if err := l.PrependPathLaunchEnv(name, format, args...); err != nil {
		ctx.Exit(1, InternalErrorf("prepending launch path env var %s: %v", name, err))
	}
}

// PrependPathSharedEnv prepends the value of this environment variable to any previous declarations of the value using
// the OS path delimiter.
func (ctx *Context) PrependPathSharedEnv(l *layers.Layer, name string, format string, args ...interface{}) {
	if err := l.PrependPathSharedEnv(name, format, args...); err != nil {
		ctx.Exit(1, InternalErrorf("prepending shared path env var %s: %v", name, err))
	}
}

// ReadMetadata reads arbitrary layer metadata from the filesystem.
func (ctx *Context) ReadMetadata(l *layers.Layer, metadata interface{}) {
	if err := l.ReadMetadata(metadata); err != nil {
		ctx.Exit(1, InternalErrorf("reading metadata: %v", err))
	}
}

// RemoveMetadata remove layer metadata from the filesystem.
func (ctx *Context) RemoveMetadata(l *layers.Layer) {
	if err := l.RemoveMetadata(); err != nil {
		ctx.Exit(1, InternalErrorf("removing metadata: %v", err))
	}
}

// WriteMetadata writes arbitrary layer metadata to the filesystem.
func (ctx *Context) WriteMetadata(l *layers.Layer, metadata interface{}, flags ...layers.Flag) {
	if err := l.WriteMetadata(metadata, flags...); err != nil {
		ctx.Exit(1, InternalErrorf("writing metadata: %v", err))
	}
}
