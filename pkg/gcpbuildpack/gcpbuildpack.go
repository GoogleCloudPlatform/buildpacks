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
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/buildpacks/libcnb"
)

const (
	// cacheHitMessage is emitted by ctx.CacheHit(). Must match acceptance test value.
	cacheHitMessage = "***** CACHE HIT:"

	// cacheMissMessage is emitted by ctx.CacheMiss(). Must match acceptance test value.
	cacheMissMessage = "***** CACHE MISS:"

	passStatusCode = 0
	failStatusCode = 100

	// labelKeyRegexpStr are valid characters for input to a label name. The label
	// name itself undergoes some transformation _after_ this regexp. See
	// AddLabel() for specifcs.
	// See https://docs.docker.com/config/labels-custom-metadata/#key-format-recommendations
	// for the formal label specification, though this regexp is also sensitive to strings
	// allowable as env vars - for example, it does not allow "." even though the label
	// specification does.
	labelKeyRegexpStr = `\A[A-Za-z][A-Za-z0-9-_]*\z`
)

var (
	logger         = log.New(os.Stderr, "", 0)
	labelKeyRegexp = regexp.MustCompile(labelKeyRegexpStr)
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
	info            libcnb.BuildpackInfo
	applicationRoot string
	buildpackRoot   string
	debug           bool
	stats           stats
	exiter          Exiter
	warnings        []string

	// detect items
	detectContext libcnb.DetectContext
	detectResult  libcnb.DetectResult

	// build items
	buildContext libcnb.BuildContext
	buildResult  libcnb.BuildResult
}

// NewContext creates a context.
func NewContext(info libcnb.BuildpackInfo) *Context {
	debug, err := env.IsDebugMode()
	if err != nil {
		logger.Printf("Failed to parse debug mode: %v", err)
		os.Exit(1)
	}
	ctx := &Context{
		debug: debug,
		info:  info,
	}
	ctx.exiter = defaultExiter{ctx: ctx}
	return ctx
}

// NewContextForTests creates a context to be used for tests.
func NewContextForTests(info libcnb.BuildpackInfo, root string) *Context {
	ctx := NewContext(info)
	ctx.applicationRoot = root
	return ctx
}

func newDetectContext(detectContext libcnb.DetectContext) *Context {
	ctx := NewContext(detectContext.Buildpack.Info)
	ctx.detectContext = detectContext
	ctx.applicationRoot = ctx.detectContext.Application.Path
	ctx.buildpackRoot = ctx.detectContext.Buildpack.Path
	return ctx
}

func newBuildContext(buildContext libcnb.BuildContext) *Context {
	ctx := NewContext(buildContext.Buildpack.Info)
	ctx.buildContext = buildContext
	ctx.applicationRoot = ctx.buildContext.Application.Path
	ctx.buildpackRoot = ctx.buildContext.Buildpack.Path
	ctx.buildResult = libcnb.NewBuildResult()
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

// Debug returns whether debug mode is enabled.
func (ctx *Context) Debug() bool {
	return ctx.debug
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

type gcpdetector struct {
	detectFn DetectFn
}

func (gcpd gcpdetector) Detect(ldctx libcnb.DetectContext) (libcnb.DetectResult, error) {
	ctx := newDetectContext(ldctx)
	status := StatusInternal
	defer func(now time.Time) {
		ctx.Span(fmt.Sprintf("Buildpack Detect %s", ctx.info.ID), now, status)
	}(time.Now())

	if err := gcpd.detectFn(ctx); err != nil {
		msg := fmt.Sprintf("Failed to run /bin/detect: %v", err)
		var be *Error
		if errors.As(err, &be) {
			status = be.Status
			return ctx.detectResult, be
		}
		return ctx.detectResult, Errorf(status, msg)
	}
	ctx.detectResult.Pass = true

	status = StatusOk
	return ctx.detectResult, nil
}

// detect implements the /bin/detect phase of the buildpack.
func detect(detectFn DetectFn, opts ...libcnb.Option) {
	gcpd := gcpdetector{detectFn: detectFn}
	libcnb.Detect(gcpd, opts...)
}

type gcpbuilder struct {
	buildFn BuildFn
}

func (gcpb gcpbuilder) Build(lbctx libcnb.BuildContext) (libcnb.BuildResult, error) {
	start := time.Now()
	ctx := newBuildContext(lbctx)
	ctx.Logf("=== %s (%s@%s) ===", ctx.BuildpackName(), ctx.BuildpackID(), ctx.BuildpackVersion())

	status := StatusInternal
	defer func(now time.Time) {
		ctx.Span(fmt.Sprintf("Buildpack Build %s", ctx.BuildpackID()), now, status)
	}(time.Now())

	if err := gcpb.buildFn(ctx); err != nil {
		msg := fmt.Sprintf("Failed to run /bin/build: %v", err)
		var be *Error
		if errors.As(err, &be) {
			status = be.Status
			ctx.Exit(1, be)
		}
		ctx.Exit(1, Errorf(status, msg))
	}

	status = StatusOk
	ctx.saveSuccessOutput(time.Since(start))
	return ctx.buildResult, nil
}

func build(buildFn BuildFn) {
	gcpb := gcpbuilder{buildFn: buildFn}
	libcnb.Build(gcpb)
}

// Exit causes the buildpack to exit with the given exit code and message.
func (ctx *Context) Exit(exitCode int, be *Error) {
	ctx.exiter.Exit(exitCode, be)
}

// OptOut is used during the detect phase to opt out of the build process.
func (ctx *Context) OptOut(format string, args ...interface{}) {
	ctx.Logf(format, args...)
	os.Exit(failStatusCode)
}

// OptIn is used during the detect phase to opt in to the build process.
func (ctx *Context) OptIn(format string, args ...interface{}) {
	ctx.Logf(format, args...)
	os.Exit(passStatusCode)
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
	ctx.warnings = append(ctx.warnings, fmt.Sprintf(format, args...))
	ctx.Logf("WARNING: "+format, args...)
}

// Tipf emits a structured logging line for usage tips.
func (ctx *Context) Tipf(format string, args ...interface{}) {
	// Tips are only displayed for the gcp/base builder, not in GAE/GCF environments.
	if os.Getenv("CNB_STACK_ID") == "google" {
		ctx.Logf(format, args...)
	}
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
	attributes := map[string]interface{}{
		"/buildpack_id":      ctx.BuildpackID(),
		"/buildpack_name":    ctx.BuildpackName(),
		"/buildpack_version": ctx.BuildpackVersion(),
	}
	si, err := newSpanInfo(label, start, now, attributes, status)
	if err != nil {
		ctx.Warnf("Invalid span dropped: %v", err)
	}
	ctx.stats.spans = append(ctx.stats.spans, si)
}

// AddBuildPlanProvides adds a provided dependency to the build plan.
func (ctx *Context) AddBuildPlanProvides(provides libcnb.BuildPlanProvide) {
	if len(ctx.detectResult.Plans) == 0 {
		ctx.detectResult.Plans = []libcnb.BuildPlan{libcnb.BuildPlan{}}
	}
	// TODO: Figure when/why there would be multiple plans and how that will affect this interface.
	plan := ctx.detectResult.Plans[0]
	plan.Provides = append(plan.Provides, provides)
}

// AddBuildPlanRequires adds a required dependency to the build plan.
func (ctx *Context) AddBuildPlanRequires(requires libcnb.BuildPlanRequire) {
	if len(ctx.detectResult.Plans) == 0 {
		ctx.detectResult.Plans = []libcnb.BuildPlan{libcnb.BuildPlan{}}
	}
	plan := ctx.detectResult.Plans[0]
	plan.Requires = append(plan.Requires, requires)
}

// AddBuildpackPlanEntry adds an entry to the build plan.
func (ctx *Context) AddBuildpackPlanEntry(entry libcnb.BuildpackPlanEntry) {
	if ctx.buildResult.Plan == nil {
		ctx.buildResult.Plan = &libcnb.BuildpackPlan{}
	}
	ctx.buildResult.Plan.Entries = append(ctx.buildResult.Plan.Entries, entry)
}

// AddWebProcess adds the given command as the web start process, overwriting any previous web start process.
func (ctx *Context) AddWebProcess(cmd []string) {
	current := ctx.buildResult.Processes
	ctx.buildResult.Processes = []libcnb.Process{}
	for _, p := range current {
		if p.Type == "web" {
			ctx.Debugf("Overwriting existing web process %q.", p.Command)
			continue // Do not add this item back to the ctx.processes; we are overwriting it.
		}
		ctx.buildResult.Processes = append(ctx.buildResult.Processes, p)
	}
	p := libcnb.Process{
		Type:    "web",
		Command: cmd[0],
		Direct:  true, // Uses Exec (no shell).
	}
	if len(cmd) > 1 {
		p.Arguments = cmd[1:]
	}
	ctx.buildResult.Processes = append(ctx.buildResult.Processes, p)
}

// HTTPStatus returns the status code for a url.
func (ctx *Context) HTTPStatus(url string) int {
	res, err := http.Head(url)
	if err != nil {
		ctx.Exit(1, UserErrorf("making a request to %s", url))
	}
	return res.StatusCode
}

// AddLabel adds a label to the user's application container.
func (ctx *Context) AddLabel(key, value string) {
	if !labelKeyRegexp.MatchString(key) {
		ctx.Warnf("Label %q does not match %s, skipping.", key, labelKeyRegexpStr)
		return
	}
	if strings.Contains(key, "__") {
		ctx.Warnf("Label %q must not contain consecutive underscores, skipping.", key)
		return
	}
	key = "google." + strings.ToLower(strings.ReplaceAll(key, "_", "-"))
	ctx.buildResult.Labels = append(ctx.buildResult.Labels, libcnb.Label{Key: key, Value: value})
}
