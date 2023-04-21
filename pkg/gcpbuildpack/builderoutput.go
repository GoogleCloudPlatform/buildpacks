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
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/builderoutput"
)

const (
	builderOutputEnv         = "BUILDER_OUTPUT"
	builderOutputFilename    = "output"
	expectedBuilderOutputEnv = "EXPECTED_BUILDER_OUTPUT"
)

var (
	maxMessageBytes = 3000
	// InternalErrorf constructs an Error with status StatusInternal (Google-attributed SLO).
	InternalErrorf = buildererror.InternalErrorf
	// UserErrorf constructs an Error with status StatusUnknown (user-attributed SLO).
	UserErrorf = buildererror.UserErrorf
)

// MessageProducer is a function that produces a useful message from the result.
type MessageProducer func(result *ExecResult) string

// KeepCombinedTail returns the tail of the combined stdout/stderr from the result.
var KeepCombinedTail = func(result *ExecResult) string { return keepTail(result.Combined) }

// KeepCombinedHead returns the head of the combined stdout/stderr from the result.
var KeepCombinedHead = func(result *ExecResult) string { return keepHead(result.Combined) }

// KeepStderrTail returns the tail of stderr from the result.
var KeepStderrTail = func(result *ExecResult) string { return keepTail(result.Stderr) }

// KeepStderrHead returns the head of stderr from the result.
var KeepStderrHead = func(result *ExecResult) string { return keepHead(result.Stderr) }

// KeepStdoutTail returns the tail of stdout from the result.
var KeepStdoutTail = func(result *ExecResult) string { return keepTail(result.Stdout) }

// KeepStdoutHead returns the head of stdout from the result.
var KeepStdoutHead = func(result *ExecResult) string { return keepHead(result.Stdout) }

// saveErrorOutput saves to the builder output file, if appropriate.
func (ctx *Context) saveErrorOutput(be *buildererror.Error) {
	outputDir := os.Getenv(builderOutputEnv)
	if outputDir == "" {
		return
	}

	if len(be.Message) > maxMessageBytes {
		be.Message = keepTail(be.Message)
	}

	be.BuildpackID, be.BuildpackVersion = ctx.BuildpackID(), ctx.BuildpackVersion()
	bo := builderoutput.BuilderOutput{Error: *be}
	bm := buildermetrics.GlobalBuilderMetrics()
	bo.Metrics = *bm
	data, err := bo.JSON()
	if err != nil {
		ctx.Warnf("Failed to marshal, skipping structured error output: %v", err)
		return
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		ctx.Warnf("Failed to create dir %s, skipping structured error output: %v", outputDir, err)
		return
	}

	// /bin/detect steps run in parallel, so they might compete over the output file. To eliminate
	// this competition, write to temp file, then `mv -f` to final location (last one in wins).
	tname := filepath.Join(outputDir, fmt.Sprintf("%s-%d", builderOutputFilename, rand.Int()))
	if err := ioutil.WriteFile(tname, data, 0644); err != nil {
		ctx.Warnf("Failed to write %s, skipping structured error output: %v", tname, err)
		return
	}
	fname := filepath.Join(outputDir, builderOutputFilename)
	if _, err := ctx.Exec([]string{"mv", "-f", tname, fname}); err != nil {
		ctx.Warnf("Failed to move %s to %s, skipping structured error output: %v", tname, fname, err)
		return
	}
	if expected := os.Getenv(expectedBuilderOutputEnv); expected != "" {
		// This logic is for acceptance tests. Ideally they would examine $BUILDER_OUTPUT themselves, but as
		// currently constructed that is difficult. So instead they delegate the task of checking whether
		// $BUILDER_OUTPUT contains a certain expected error-message pattern to this code.
		r, err := regexp.Compile(expected)
		if err == nil {
			ctx.Logf("Expected pattern included in error output: %t", r.MatchString(be.Message))
		} else {
			ctx.Warnf("Bad regexp %q: %v", expectedBuilderOutputEnv, err)
		}
	}
	return
}

func keepTail(message string) string {
	message = strings.TrimSpace(message)

	if len(message) <= maxMessageBytes {
		return message
	}

	return "..." + message[len(message)-maxMessageBytes+3:]
}

func keepHead(message string) string {
	message = strings.TrimSpace(message)

	if len(message) <= maxMessageBytes {
		return message
	}

	return message[:maxMessageBytes-3] + "..."
}

// saveSuccessOutput saves information from the context into BUILDER_OUTPUT.
func (ctx *Context) saveSuccessOutput(duration time.Duration) {
	outputDir := os.Getenv(builderOutputEnv)
	if outputDir == "" {
		return
	}

	bo := builderoutput.New()
	fname := filepath.Join(outputDir, builderOutputFilename)

	fnameExists, err := ctx.FileExists(fname)
	if err != nil {
		ctx.Warnf("Failed to determine if %s exists, skipping statistics: %v", fname, err)
		return
	}
	// Previous buildpacks may have already written to the builder output file.
	if fnameExists {
		content, err := ioutil.ReadFile(fname)
		if err != nil {
			ctx.Warnf("Failed to read %s, skipping statistics: %v", fname, err)
			return
		}
		bofj, err := builderoutput.FromJSON(content)
		bo = &bofj
		if err != nil {
			ctx.Warnf("Failed to unmarshal %s, skipping statistics: %v", fname, err)
			return
		}
	}

	if len(ctx.InstalledRuntimeVersions()) > 0 {
		bo.InstalledRuntimeVersions = append(bo.InstalledRuntimeVersions, ctx.InstalledRuntimeVersions()...)
	}

	bo.Stats = append(bo.Stats, builderoutput.BuilderStat{
		BuildpackID:      ctx.BuildpackID(),
		BuildpackVersion: ctx.BuildpackVersion(),
		DurationMs:       duration.Milliseconds(),
		UserDurationMs:   ctx.stats.user.Milliseconds(),
	})
	bo.Warnings = append(bo.Warnings, ctx.warnings...)

	bm := buildermetrics.GlobalBuilderMetrics()
	bm.ForEachCounter(func(id buildermetrics.CounterID, c *buildermetrics.Counter) {
		count := bo.Metrics.GetCounter(id)
		count.Increment(c.Value())
	})

	var content []byte
	// Make sure the message is smaller than the maximum allowed size.
	for {
		var err error
		content, err = bo.JSON()
		if err != nil {
			ctx.Warnf("Failed to marshal stats, skipping statistics: %v", err)
			return
		}
		if len(content) <= maxMessageBytes {
			break
		}
		// This is a defensive check; if there are no warnings, the message should be small enough.
		// In either case, skip this stat.
		if len(bo.Warnings) == 0 {
			ctx.Warnf("The builder output is too large and there are no warnings, skipping statistics")
			return
		}
		diff := len(content) - maxMessageBytes
		last := len(bo.Warnings) - 1
		// If the last warning is too long, only trim it. Otherwise, drop it.
		// Also drop the last warning if it is shorter than three characters.
		if len(bo.Warnings[last]) > diff+3 {
			bo.Warnings[last] = bo.Warnings[last][:len(bo.Warnings[last])-diff-3] + "..."
		} else {
			bo.Warnings = bo.Warnings[:last]
		}
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		ctx.Warnf("Failed to create dir %s, skipping statistics: %v", outputDir, err)
		return
	}
	if err := ioutil.WriteFile(fname, content, 0644); err != nil {
		ctx.Warnf("Failed to write %s, skipping statistics: %v", fname, err)
		return
	}
}
