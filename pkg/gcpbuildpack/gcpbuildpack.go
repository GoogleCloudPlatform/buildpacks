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
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
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

	// WebProcess is the name of the default web process.
	WebProcess = "web"
)

var (
	logger         = log.New(os.Stderr, "", 0)
	labelKeyRegexp = regexp.MustCompile(labelKeyRegexpStr)
)

// DetectFn is the callback signature for Detect()
type DetectFn func(*Context) (DetectResult, error)

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

	// build items
	buildContext libcnb.BuildContext
	buildResult  libcnb.BuildResult

	execCmd func(name string, arg ...string) *exec.Cmd
}

// ContextOption configures NewContext functions.
type ContextOption func(ctx *Context)

// WithApplicationRoot sets the application root in Context.
func WithApplicationRoot(root string) ContextOption {
	return func(ctx *Context) {
		ctx.applicationRoot = root
	}
}

// WithBuildpackRoot sets the buildpack root in Context.
func WithBuildpackRoot(root string) ContextOption {
	return func(ctx *Context) {
		ctx.buildpackRoot = root
	}
}

// WithBuildpackInfo sets the buildpack info in Context.
func WithBuildpackInfo(info libcnb.BuildpackInfo) ContextOption {
	return func(ctx *Context) {
		ctx.info = info
	}
}

// WithBuildContext sets the buildContext in Context.
func WithBuildContext(buildCtx libcnb.BuildContext) ContextOption {
	return func(ctx *Context) {
		ctx.buildContext = buildCtx
	}
}

// WithExecCmd overrides the exec.Cmd instance used for executing commands,
// primarily useful for testing.
func WithExecCmd(execCmd func(name string, args ...string) *exec.Cmd) ContextOption {
	return func(ctx *Context) {
		ctx.execCmd = execCmd
	}
}

// NewContext creates a context.
func NewContext(opts ...ContextOption) *Context {
	debug, err := env.IsDebugMode()
	if err != nil {
		logger.Printf("Failed to parse debug mode: %v", err)
		os.Exit(1)
	}
	ctx := &Context{
		debug:   debug,
		execCmd: exec.Command,
	}
	ctx.exiter = defaultExiter{ctx: ctx}
	for _, o := range opts {
		o(ctx)
	}

	return ctx
}

func newDetectContext(detectContext libcnb.DetectContext) *Context {
	ctx := NewContext(WithBuildpackInfo(detectContext.Buildpack.Info))
	ctx.detectContext = detectContext
	ctx.applicationRoot = ctx.detectContext.Application.Path
	ctx.buildpackRoot = ctx.detectContext.Buildpack.Path
	return ctx
}

func newBuildContext(buildContext libcnb.BuildContext) *Context {
	ctx := NewContext(WithBuildpackInfo(buildContext.Buildpack.Info))
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

// Processes returns the list of processes added by buildpacks.
func (ctx *Context) Processes() []libcnb.Process {
	return ctx.buildResult.Processes
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
	status := buildererror.StatusInternal
	defer func(now time.Time) {
		ctx.Span(fmt.Sprintf("Buildpack Detect %s", ctx.info.ID), now, status)
	}(time.Now())

	result, err := gcpd.detectFn(ctx)
	if err != nil {
		msg := fmt.Sprintf("Failed to run /bin/detect: %v", err)
		var be *buildererror.Error
		if errors.As(err, &be) {
			status = be.Status
			return libcnb.DetectResult{}, be
		}
		return libcnb.DetectResult{}, buildererror.Errorf(status, msg)
	}
	// detectFn has an interface return type so result may be nil.
	if result == nil {
		return libcnb.DetectResult{}, InternalErrorf("detect did not return a result or an error")
	}

	status = buildererror.StatusOk
	ctx.Logf(result.Reason())
	return result.Result(), nil
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

	status := buildererror.StatusInternal
	defer func(now time.Time) {
		ctx.Span(fmt.Sprintf("Buildpack Build %s", ctx.BuildpackID()), now, status)
	}(time.Now())

	if err := gcpb.buildFn(ctx); err != nil {
		msg := fmt.Sprintf("Failed to run /bin/build: %v", err)
		var be *buildererror.Error
		if errors.As(err, &be) {
			status = be.Status
			ctx.Exit(1, be)
		}
		ctx.Exit(1, buildererror.Errorf(status, msg))
	}

	status = buildererror.StatusOk
	ctx.saveSuccessOutput(time.Since(start))
	return ctx.buildResult, nil
}

func build(buildFn BuildFn) {
	options := []libcnb.Option{
		// Without this flag the build SBOM is NOT written to the image's "io.buildpacks.build.metadata" label.
		// The acceptence tests rely on this being present.
		libcnb.WithBOMLabel(true),
	}
	gcpb := gcpbuilder{buildFn: buildFn}
	libcnb.Build(gcpb, options...)
}

// Exit causes the buildpack to exit with the given exit code and message.
func (ctx *Context) Exit(exitCode int, be *buildererror.Error) {
	ctx.exiter.Exit(exitCode, be)
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
	if env.IsGCP() {
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
func (ctx *Context) Span(label string, start time.Time, status buildererror.Status) {
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

// AddBOMEntry adds an entry to the bill of materials.
func (ctx *Context) AddBOMEntry(entry libcnb.BOMEntry) {
	if ctx.buildResult.BOM == nil {
		ctx.buildResult.BOM = &libcnb.BOM{}
	}
	ctx.buildResult.BOM.Entries = append(ctx.buildResult.BOM.Entries, entry)
}

// AddWebProcess adds the given command as the web start process, overwriting any previous web start process.
func (ctx *Context) AddWebProcess(cmd []string) {
	ctx.AddProcess(WebProcess, cmd, AsDirectProcess(), AsDefaultProcess())
}

// processOption configures the AddProcess function.
type processOption func(o *libcnb.Process)

// AsDirectProcess causes the process to be executed directly, i.e. without a shell.
func AsDirectProcess() processOption {
	return func(o *libcnb.Process) { o.Direct = true }
}

// AsDefaultProcess marks the process as the default one for when launcher is invoked without arguments.
func AsDefaultProcess() processOption {
	return func(o *libcnb.Process) { o.Default = true }
}

// AddProcess adds the given command as named process, overwriting any previous process with the same name.
func (ctx *Context) AddProcess(name string, cmd []string, opts ...processOption) {
	current := ctx.buildResult.Processes
	ctx.buildResult.Processes = []libcnb.Process{}
	for _, p := range current {
		if p.Type == name {
			ctx.Debugf("Overwriting existing %s process %q.", name, p.Command)
			continue // Do not add this item back to the ctx.processes; we are overwriting it.
		}
		ctx.buildResult.Processes = append(ctx.buildResult.Processes, p)
	}
	p := libcnb.Process{
		Type:    name,
		Command: cmd[0],
	}
	if len(cmd) > 1 {
		p.Arguments = cmd[1:]
	}
	for _, opt := range opts {
		opt(&p)
	}
	ctx.buildResult.Processes = append(ctx.buildResult.Processes, p)
}

// HTTPStatus returns the status code for a url.
func (ctx *Context) HTTPStatus(url string) (int, error) {
	res, err := http.Head(url)
	if err != nil {
		return 0, UserErrorf("getting status code for %s: %v", url, err)
	}
	return res.StatusCode, nil
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
	ctx.Logf("Adding image label %s: %s", key, value)
	ctx.buildResult.Labels = append(ctx.buildResult.Labels, libcnb.Label{Key: key, Value: value})
}
