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

// Package gcpbuildpack is a framework for implementing buildpacks (https://buildpacks.io/).
package gcpbuildpack

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	libbuild "github.com/buildpack/libbuildpack/build"
	"github.com/buildpack/libbuildpack/buildpack"
	"github.com/buildpack/libbuildpack/buildpackplan"
	"github.com/buildpack/libbuildpack/buildplan"
	libdetect "github.com/buildpack/libbuildpack/detect"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	debugEnv = "BP_DEBUG"

	// cacheHitMessage is emitted by ctx.CacheHit(). Must match acceptance test value.
	cacheHitMessage = "***** CACHE HIT:"

	// cacheMissMessage is emitted by ctx.CacheMiss(). Must match acceptance test value.
	cacheMissMessage = "***** CACHE MISS:"
)

var (
	logger = log.New(os.Stderr, "", 0)
)

// DetectFn is the callback signature for Detect()
type DetectFn func(*Context) error

// BuildFn is the callback signature for Build()
type BuildFn func(*Context) error

type stats struct {
	spans []*spanInfo
	user  time.Duration
}

// Context provides contextually aware functions for buildpack authors.
type Context struct {
	info            buildpack.Info
	applicationRoot string
	buildpackRoot   string
	exitCode        int
	buildPlan       buildplan.Plan
	buildpackPlans  []buildpackplan.Plan
	debug           bool
	processes       layers.Processes
	d               *libdetect.Detect
	b               *libbuild.Build
	stats           stats
}

// NewContext creates a context.
func NewContext(info buildpack.Info) *Context {
	var err error
	debug := false
	if v := os.Getenv(debugEnv); v != "" {
		if debug, err = strconv.ParseBool(v); err != nil {
			logger.Printf("Warning: failed to parse env var %s: %v", debugEnv, err)
		}
	}

	return &Context{
		debug: debug,
		info:  info,
	}
}

func newDetectContext() *Context {
	d, err := libdetect.DefaultDetect()
	if err != nil {
		logger.Printf("Failed to initialize /bin/detect: %v", err)
		os.Exit(1)
	}
	ctx := NewContext(d.Buildpack.Info)
	ctx.d = &d
	ctx.applicationRoot = ctx.d.Application.Root
	ctx.buildpackRoot = ctx.d.Buildpack.Root
	return ctx
}

func newBuildContext() *Context {
	b, err := libbuild.DefaultBuild()
	if err != nil {
		logger.Printf("Failed to initialize /bin/build: %v", err)
		os.Exit(1)
	}
	ctx := NewContext(b.Buildpack.Info)
	ctx.b = &b
	ctx.applicationRoot = ctx.b.Application.Root
	ctx.buildpackRoot = ctx.b.Buildpack.Root
	return ctx
}

// BuildpackID returns the buildpack id.
func (ctx *Context) BuildpackID() string {
	return ctx.info.ID
}

// BuildpackVersion returns the buildpack version.
func (ctx *Context) BuildpackVersion() string {
	return ctx.info.Version
}

// BuildpackName returns the buildpack name.
func (ctx *Context) BuildpackName() string {
	return ctx.info.Name
}

// ApplicationRoot returns the root folder of the application code.
func (ctx *Context) ApplicationRoot() string {
	return ctx.applicationRoot
}

// BuildpackRoot returns the root folder of the buildpack.
func (ctx *Context) BuildpackRoot() string {
	return ctx.buildpackRoot
}

// Main is the main entrypoint to a buildpack's detect and build functions.
func Main(d DetectFn, b BuildFn) {
	switch filepath.Base(os.Args[0]) {
	case "detect":
		detect(d)
	case "build":
		build(b)
	default:
		logger.Print("Unknown command, expected 'detect' or 'build'.")
		os.Exit(1)
	}
}

// detect implements the /bin/detect phase of the buildpack.
func detect(f DetectFn) {
	ctx := newDetectContext()
	status := StatusInternal
	defer func(now time.Time) {
		ctx.Span(fmt.Sprintf("Buildpack Detect %s", ctx.info.ID), now, status)
	}(time.Now())

	if err := f(ctx); err != nil {
		msg := fmt.Sprintf("Failed to run /bin/detect: %v", err)
		var be *Error
		if errors.As(err, &be) {
			status = be.Status
			ctx.Exit(ctx.d.Error(1), be)
		}
		ctx.Exit(ctx.d.Error(1), Errorf(status, msg))
	}

	_, err := ctx.d.Pass(ctx.buildPlan)
	if err != nil {
		ctx.Exit(ctx.d.Error(1), Errorf(StatusInternal, err.Error()))
	}

	status = StatusOk
}

func build(b BuildFn) {
	start := time.Now()
	ctx := newBuildContext()
	ctx.Logf("======== %s@%s ========", ctx.BuildpackID(), ctx.BuildpackVersion())
	ctx.Logf(ctx.BuildpackName())

	status := StatusInternal
	defer func(now time.Time) {
		ctx.Span(fmt.Sprintf("Buildpack Build %s", ctx.BuildpackID()), now, status)
	}(time.Now())

	if err := b(ctx); err != nil {
		msg := fmt.Sprintf("Failed to run /bin/build: %v", err)
		var be *Error
		if errors.As(err, &be) {
			status = be.Status
			ctx.Exit(ctx.b.Failure(1), be)
		}
		ctx.Exit(ctx.b.Failure(1), Errorf(status, msg))
	}

	// Emit application metadata.
	if len(ctx.processes) > 0 {
		metadata := layers.Metadata{Processes: ctx.processes}
		if err := ctx.b.Layers.WriteApplicationMetadata(metadata); err != nil {
			ctx.Exit(ctx.b.Failure(1), Errorf(StatusInternal, "writing application metadata: %v", err))
		}
	}

	if _, err := ctx.b.Success(ctx.buildpackPlans...); err != nil {
		ctx.Exit(ctx.b.Failure(1), Errorf(StatusInternal, err.Error()))
	}

	status = StatusOk
	ctx.saveSuccessOutput(time.Since(start))
}

// Exit causes the buildpack to exit with the given exit code and message.
func (ctx *Context) Exit(exitCode int, be *Error) {
	if be != nil {
		ctx.Logf("Failure: " + be.Message)
		ctx.saveErrorOutput(be)
	}
	ctx.exitCode = exitCode
	os.Exit(exitCode)
}

// OptOut is used during the detect phase to opt out of the build process.
func (ctx *Context) OptOut(format string, args ...interface{}) {
	ctx.Logf(format, args...)
	os.Exit(libdetect.FailStatusCode)
}

// OptIn is used during the detect phase to opt in to the build process.
func (ctx *Context) OptIn(format string, args ...interface{}) {
	ctx.Logf(format, args...)
	os.Exit(libdetect.PassStatusCode)
}

// Logf emits a structured logging line.
func (ctx *Context) Logf(format string, args ...interface{}) {
	logger.Printf(format, args...)
}

// Debugf emits a structured logging line if the debug flag is set.
func (ctx *Context) Debugf(format string, args ...interface{}) {
	if !ctx.debug {
		return
	}
	ctx.Logf("DEBUG: "+format, args...)
}

// Warnf emits a structured logging line for warnings.
func (ctx *Context) Warnf(format string, args ...interface{}) {
	ctx.Logf("Warning: "+format, args...)
}

// CacheHit records a cache hit debug message. This is used in acceptance test validation.
func (ctx *Context) CacheHit(tag string) {
	ctx.Debugf("%s %q", cacheHitMessage, tag)
}

// CacheMiss records a cache miss debug message. This is used in acceptance test validation.
func (ctx *Context) CacheMiss(tag string) {
	ctx.Debugf("%s %q", cacheMissMessage, tag)
}

// Span emits a structured Stackdriver span.
func (ctx *Context) Span(label string, start time.Time, status Status) {
	now := time.Now()
	ctx.Logf("Timing %v %s `%s`", time.Since(start), status, label)
	attributes := map[string]interface{}{
		"/buildpack_id":      ctx.BuildpackID(),
		"/buildpack_name":    ctx.BuildpackName(),
		"/buildpack_version": ctx.BuildpackVersion(),
	}
	si, err := newSpanInfo(label, start, now, attributes, status)
	if err != nil {
		ctx.Logf("Warning: invalid span dropped: %v", err)
	}
	ctx.stats.spans = append(ctx.stats.spans, si)
}

// AddBuildPlanProvides adds a provided dependency to the build plan.
func (ctx *Context) AddBuildPlanProvides(provided buildplan.Provided) {
	ctx.buildPlan.Provides = append(ctx.buildPlan.Provides, provided)
}

// AddBuildPlanRequires adds a required dependency to the build plan.
func (ctx *Context) AddBuildPlanRequires(required buildplan.Required) {
	ctx.buildPlan.Requires = append(ctx.buildPlan.Requires, required)
}

// AddBuildpackPlan adds a required dependency to the build plan.
func (ctx *Context) AddBuildpackPlan(plan buildpackplan.Plan) {
	ctx.buildpackPlans = append(ctx.buildpackPlans, plan)
}

// AddWebProcess adds the given command as the web start process, overwriting any previous web start process.
func (ctx *Context) AddWebProcess(cmd []string) {
	current := ctx.processes
	ctx.processes = layers.Processes{}
	for _, p := range current {
		if p.Type == "web" {
			ctx.Logf("Warning: overwriting existing web process %q.", p.Command)
			continue // Do not add this item back to the ctx.processes; we are overwriting it.
		}
		ctx.processes = append(ctx.processes, p)
	}
	p := layers.Process{
		Type:    "web",
		Command: cmd[0],
		Direct:  true, // Uses Exec (no shell).
	}
	if len(cmd) > 1 {
		p.Args = cmd[1:]
	}
	ctx.processes = append(ctx.processes, p)
}

// HTTPStatus returns the status code for a url.
func (ctx *Context) HTTPStatus(url string) int {
	result := ctx.Exec([]string{"curl", "--head", "-w", "%{http_code}", "-o", "/dev/null", "--silent", "--location", url})
	cs := strings.TrimSpace(result.Stdout)
	c, err := strconv.Atoi(cs)
	if err != nil {
		ctx.Exit(1, UserErrorf("Unexpected response code %q from %s.", cs, url))
	}
	return c
}
