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
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"cloud.google.com/go/logging"
	"github.com/buildpacks/libcnb/v2"
	"google.golang.org/api/option"
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
	labelKeyRegexp = regexp.MustCompile(labelKeyRegexpStr)
)

// DetectFn is the callback signature for Detect()
type DetectFn func(*Context) (DetectResult, error)

// BuildFn is the callback signature for Build()
type BuildFn func(*Context) error

// BuildpackFuncs contains the Detect and Build functions for a buildpack.
type BuildpackFuncs struct {
	Detect DetectFn
	Build  BuildFn
}

type stats struct {
	spans []*spanInfo
	user  time.Duration
}

// Context provides contextually aware functions for buildpack authors.
type Context struct {
	info                     libcnb.BuildpackInfo
	applicationRoot          string
	buildpackRoot            string
	debug                    bool
	logger                   *log.Logger
	cloudLogger              *logging.Logger
	installedRuntimeVersions []string
	stats                    stats
	exiter                   Exiter
	warnings                 []string

	// detect items
	detectContext libcnb.DetectContext

	// build items
	buildContext      libcnb.BuildContext
	buildResult       libcnb.BuildResult
	layerContributors []layerContributor

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

// WithLogger override the logger implementation, this is useful for unit tests
// which want to verify logging output.
func WithLogger(logger *log.Logger) ContextOption {
	return func(ctx *Context) {
		ctx.logger = logger
	}
}

// WithStackID sets the StackID in Context.
func WithStackID(stackID string) ContextOption {
	return func(ctx *Context) {
		ctx.buildContext.StackID = stackID
	}
}

// NewContext creates a context.
func NewContext(opts ...ContextOption) *Context {
	debug, err := env.IsDebugMode()
	defaultLogger := log.New(os.Stderr, "", 0)
	if err != nil {
		defaultLogger.Printf("Failed to parse debug mode: %v", err)
		os.Exit(1)
	}
	ctx := &Context{
		debug:   debug,
		execCmd: exec.Command,
		logger:  defaultLogger,
	}

	// Set up cloud logging if the GOOGLE_ENABLE_STRUCTURED_LOGGING env var is set by RCS.
	if os.Getenv("GOOGLE_ENABLE_STRUCTURED_LOGGING") == "true" {
		client, err := logging.NewClient(context.Background(), "", option.WithoutAuthentication())
		if err != nil {
			defaultLogger.Printf("WARNING: Failed to create Cloud Logging client for structured logs: %v", err)
		} else {
			// This cloud build logger will write JSON-formatted logs to os.Stderr.
			// RedirectAsJSON avoids sending the logs to the Cloud Logging API (which would fail with the no-auth client),
			// and serializes the Entry object into a JSON string and writes it to os.Stderr.
			ctx.cloudLogger = client.Logger("structured-logs", logging.RedirectAsJSON(os.Stderr))
		}
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
	ctx.applicationRoot = ctx.detectContext.ApplicationPath
	ctx.buildpackRoot = ctx.detectContext.Buildpack.Path
	return ctx
}

func newBuildContext(buildContext libcnb.BuildContext) *Context {
	ctx := NewContext(WithBuildpackInfo(buildContext.Buildpack.Info))
	ctx.buildContext = buildContext
	ctx.applicationRoot = ctx.buildContext.ApplicationPath
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

// StackID returns the stack id.
func (ctx *Context) StackID() string {
	return ctx.buildContext.StackID
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
		log.New(os.Stderr, "", 0).Print("Unknown command, expected 'detect' or 'build'.")
		os.Exit(1)
	}
}

type gcpdetector struct {
	detectFn DetectFn
}

// detectFnWrapper creates a DetectFunc that wraps the given detectFn.
func detectFnWrapper(detectFn DetectFn) libcnb.DetectFunc {
	return func(ldctx libcnb.DetectContext) (libcnb.DetectResult, error) {

		ctx := newDetectContext(ldctx)
		status := buildererror.StatusInternal
		defer func(now time.Time) {
			ctx.Span(fmt.Sprintf("Buildpack Detect %q", ctx.info.ID), now, status)
		}(time.Now())

		result, err := detectFn(ctx)
		if err != nil {
			msg := fmt.Sprintf("failed to run /bin/detect: %v", err)
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
}

// detect implements the /bin/detect phase of the buildpack.
func detect(detectFn DetectFn, opts ...libcnb.Option) {
	config := libcnb.NewConfig(opts...)
	wrappedDetectFn := detectFnWrapper(detectFn)
	libcnb.Detect(wrappedDetectFn, config)
}

type gcpbuilder struct {
	buildFn BuildFn
}

// buildFnWrapper creates a libcnb.BuildFunc that wraps the given buildFn.
func buildFnWrapper(buildFn BuildFn) libcnb.BuildFunc {
	return func(lbctx libcnb.BuildContext) (libcnb.BuildResult, error) {
		start := time.Now()
		ctx := newBuildContext(lbctx)
		ctx.Logf("=== %s (%s@%s) ===", ctx.BuildpackName(), ctx.BuildpackID(), ctx.BuildpackVersion())

		status := buildererror.StatusInternal
		defer func(now time.Time) {
			ctx.Span(fmt.Sprintf("Buildpack Build %q", ctx.BuildpackID()), now, status)
		}(time.Now())

		if err := buildFn(ctx); err != nil {
			var be *buildererror.Error
			if errors.As(err, &be) {
				status = be.Status
			}
			err := fmt.Errorf("failed to build: %w", err)
			ctx.Exit(1, err)
		}

		for i := 0; i < len(ctx.buildResult.Layers); i++ {
			creator := ctx.layerContributors[i]
			name := creator.Name()
			layer, _ := ctx.buildContext.Layers.Layer(name)
			layer, _ = creator.Contribute(layer)
			ctx.buildResult.Layers[i] = layer
		}
		status = buildererror.StatusOk
		ctx.saveSuccessOutput(time.Since(start))
		return ctx.buildResult, nil
	}
}

// build implements the /bin/build phase of the buildpack.
func build(buildFn BuildFn) {
	options := []libcnb.Option{
		// Without this flag the build SBOM is NOT written to the image's "io.buildpacks.build.metadata" label.
		// The acceptence tests rely on this being present.
		// libcnb.WithBOMLabel(true),
	}
	config := libcnb.NewConfig(options...)
	wrappedBuildFn := buildFnWrapper(buildFn)
	libcnb.Build(wrappedBuildFn, config)
}

// Exit causes the buildpack to exit with the given exit code and message.
func (ctx *Context) Exit(exitCode int, err error) {
	ctx.exiter.Exit(exitCode, err)
}

// Logf emits a structured logging line.
func (ctx *Context) Logf(format string, args ...interface{}) {
	ctx.logger.Printf(format, args...)
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

// StructuredLogf logs a structured JSON payload to Cloud Logging.
func (ctx *Context) StructuredLogf(severity logging.Severity, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if message == "" {
		return
	}
	// Fallback to the simple logger if the structured logger isn't initialized.
	if ctx.cloudLogger == nil {
		ctx.logger.Print(message)
		return
	}

	ctx.cloudLogger.Log(logging.Entry{
		Severity: severity,
		Payload:  message,
	})
}

// CacheHit records a cache hit debug message. This is used in acceptance test validation.
func (ctx *Context) CacheHit(tag string) {
	ctx.Logf("%s %q", cacheHitMessage, tag)
}

// CacheMiss records a cache miss debug message. This is used in acceptance test validation.
func (ctx *Context) CacheMiss(tag string) {
	ctx.Logf("%s %q", cacheMissMessage, tag)
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

// InstalledRuntimeVersions returns the list of runtime versions installed during build time.
func (ctx *Context) InstalledRuntimeVersions() []string {
	return ctx.installedRuntimeVersions
}

// AddInstalledRuntimeVersion adds a runtime version to the list of installed runtimes. Used
// for versionless runtimes to provide feedback on the runtime version selected at build time.
func (ctx *Context) AddInstalledRuntimeVersion(version string) {
	ctx.installedRuntimeVersions = append(ctx.installedRuntimeVersions, version)
}

// AddWebProcess adds the given command as the web start process, overwriting any previous web start process.
func (ctx *Context) AddWebProcess(cmd []string) {
	ctx.AddProcess(WebProcess, cmd, AsDirectProcess(), AsDefaultProcess())
}

// processOption configures the AddProcess function.
type processOption func(o *libcnb.Process)

// AsDirectProcess causes the process to be executed directly, i.e. without a shell.
func AsDirectProcess() processOption {
	return func(o *libcnb.Process) {
		o.Command = o.Command[2:]
	}
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
			ctx.Logf("Overwriting existing %s process %q.", name, p.Command)
			continue // Do not add this item back to the ctx.processes; we are overwriting it.
		}
		ctx.buildResult.Processes = append(ctx.buildResult.Processes, p)
	}

	cmdWithDirectAsFalse := append([]string{"bash", "-c"}, cmd...)
	p := libcnb.Process{
		Type:    name,
		Command: cmdWithDirectAsFalse,
	}
	for _, opt := range opts {
		opt(&p)
	}
	if len(p.Command) > 0 && p.Command[0] == "bash" {
		p.Command = []string{"bash", "-c", strings.Join(p.Command[2:], " ")}
	}
	ctx.buildResult.Processes = append(ctx.buildResult.Processes, p)

}

// HTTPStatus returns the status code for a url.
func (ctx *Context) HTTPStatus(url string) (int, error) {
	res, err := http.Head(url)
	if err != nil {
		return 0, InternalErrorf("getting status code for %s: %v", url, err)
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

// MainRunner is the main entrypoint for runners.
func MainRunner(buildpacks map[string]BuildpackFuncs, buildpackID *string, phase *string) {
	ctx := NewContext()
	if *buildpackID == "" {
		err := buildererror.Errorf(buildererror.StatusInternal, "Usage: runner -buildpack <id> [args...]")
		ctx.Exit(failStatusCode, err)
	}

	bp, ok := buildpacks[*buildpackID]
	if !ok {
		var ids []string
		for id := range buildpacks {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		err := buildererror.Errorf(buildererror.StatusInternal, "Unknown buildpack ID: %s\nRegistered buildpacks are:\n  %s", *buildpackID, strings.Join(ids, "\n  "))
		ctx.Exit(failStatusCode, err)
	}

	switch *phase {
	case "detect":
		detect(bp.Detect)
	case "build":
		build(bp.Build)
	default:
		err := buildererror.Errorf(buildererror.StatusInternal, "Invalid phase %q, expected 'detect' or 'build'", *phase)
		ctx.Exit(failStatusCode, err)
	}
}
