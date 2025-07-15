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

// Implements python/missing-entrypoint buildpack.
// This buildpack's goal is to display a clear error message when
// no entrypoint is defined on a Python application.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	gunicorn  = "gunicorn"
	uvicorn   = "uvicorn"
	gradio    = "gradio"
	streamlit = "streamlit"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride("python"); result != nil {
		return result, nil
	}

	atLeastOne, err := ctx.HasAtLeastOne("*.py")
	if err != nil {
		return nil, fmt.Errorf("finding *.py files: %w", err)
	}
	if !atLeastOne {
		return gcp.OptOut("no .py files found"), nil
	}
	return gcp.OptIn("found .py files"), nil
}

func buildFn(ctx *gcp.Context) error {
	hasMain, err := ctx.HasAtLeastOne("main.py")
	if err != nil {
		return fmt.Errorf("finding main.py files: %w", err)
	}
	if !hasMain {
		return fmt.Errorf("for Python, provide a main.py file or set an entrypoint with %q env var or by creating a %q file", env.Entrypoint, "Procfile")
	}
	cmd := []string{"gunicorn", "-b", ":8080", "main:app"}

	// We will use the smart default entrypoint if the runtime version supports it (>=3.13)
	if supports, err := python.SupportsSmartDefaultEntrypoint(ctx); err == nil && supports {
		// We will eventually remove the FastAPISmartDefaults flag and use smartdefaultEntrypoint for
		// all use-cases.
		if os.Getenv(env.FastAPISmartDefaults) == "true" {
			cmd, err = uvicornEntrypoint(ctx)
			if err != nil {
				return fmt.Errorf("error detecting smart default entrypoint: %w", err)
			}
		}
		if os.Getenv(env.PythonSmartDefaults) == "true" {
			cmd, err = smartDefaultEntrypoint(ctx)
			if err != nil {
				return fmt.Errorf("error detecting smart default entrypoint: %w", err)
			}
		}
	}
	ctx.Warnf("Setting default entrypoint: %q", strings.Join(cmd, " "))
	ctx.AddProcess(gcp.WebProcess, cmd, gcp.AsDefaultProcess())

	return nil
}

func uvicornEntrypoint(ctx *gcp.Context) ([]string, error) {
	// To be compatible with the old builder, we will use below priority order:
	// 1. gunicorn 2. uvicorn
	// If gunicorn is present in requirements.txt, we will use gunicorn as the entrypoint.
	gPresent, err := python.PackagePresent(ctx, "requirements.txt", gunicorn)
	if err != nil {
		return nil, fmt.Errorf("error detecting gunicorn: %w", err)
	}
	if gPresent {
		return []string{"gunicorn", "-b", ":8080", "main:app"}, nil
	}
	// If uvicorn is present in requirements.txt, we will use uvicorn as the entrypoint.
	uPresent, err := python.PackagePresent(ctx, "requirements.txt", uvicorn)
	if err != nil {
		return nil, fmt.Errorf("error detecting uvicorn: %w", err)
	}
	if uPresent {
		return []string{"uvicorn", "main:app", "--port", "8080", "--host", "0.0.0.0"}, nil
	}
	return []string{"gunicorn", "-b", ":8080", "main:app"}, nil
}

func smartDefaultEntrypoint(ctx *gcp.Context) ([]string, error) {
	// To be compatible with the old builder, we will use below priority order:
	// 1. gunicorn 2. uvicorn 3. gradio 4. streamlit
	// If gunicorn is present in requirements.txt, we will use gunicorn as the entrypoint.
	gPresent, err := python.PackagePresent(ctx, "requirements.txt", gunicorn)
	if err != nil {
		return nil, fmt.Errorf("error detecting gunicorn: %w", err)
	}
	if gPresent {
		return []string{"gunicorn", "-b", ":8080", "main:app"}, nil
	}
	// If uvicorn is present in requirements.txt, we will use uvicorn as the entrypoint.
	uPresent, err := python.PackagePresent(ctx, "requirements.txt", uvicorn)
	if err != nil {
		return nil, fmt.Errorf("error detecting uvicorn: %w", err)
	}
	if uPresent {
		return []string{"uvicorn", "main:app", "--port", "8080", "--host", "0.0.0.0"}, nil
	}
	// If gradio is present in requirements.txt, we will use gradio as the entrypoint.
	gradioPresent, err := python.PackagePresent(ctx, "requirements.txt", gradio)
	if err != nil {
		return nil, fmt.Errorf("error detecting gradio: %w", err)
	}
	if gradioPresent {
		if err := addGradioEnvVarLayer(ctx); err != nil {
			return nil, fmt.Errorf("error adding gradio env var layer: %w", err)
		}
		return []string{"python", "main.py"}, nil
	}
	// If streamlit is present in requirements.txt, we will use streamlit as the entrypoint.
	sPresent, err := python.PackagePresent(ctx, "requirements.txt", streamlit)
	if err != nil {
		return nil, fmt.Errorf("error detecting streamlit: %w", err)
	}
	if sPresent {
		return []string{"streamlit", "run", "main.py", "--server.address", "0.0.0.0", "--server.port", "8080"}, nil
	}

	return []string{"gunicorn", "-b", ":8080", "main:app"}, nil
}

func addGradioEnvVarLayer(ctx *gcp.Context) error {
	layer, err := ctx.Layer("gradio-env-var", gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating gradio-env-var layer: %w", err)
	}
	layer.LaunchEnvironment.Default("GRADIO_SERVER_NAME", "0.0.0.0")
	layer.LaunchEnvironment.Default("GRADIO_SERVER_PORT", "8080")
	return nil
}
