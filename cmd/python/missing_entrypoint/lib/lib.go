// Copyright 2025 Google LLC
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
package lib

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	gunicorn        = "gunicorn"
	uvicorn         = "uvicorn"
	gradio          = "gradio"
	streamlit       = "streamlit"
	fastapiStandard = "fastapi[standard]"
	requirements    = "requirements.txt"
	pyprojectToml   = "pyproject.toml"
	googleAdk       = "google-adk"
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
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

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	hasMain, err := ctx.FileExists("main.py")
	if err != nil {
		return fmt.Errorf("finding main.py file: %w", err)
	}
	hasApp, err := ctx.FileExists("app.py")
	if err != nil {
		return fmt.Errorf("finding app.py file: %w", err)
	}

	pyprojectExists, err := ctx.FileExists(pyprojectToml)
	if err != nil {
		return fmt.Errorf("error checking for pyproject.toml: %w", err)
	}

	// We will use the smart default entrypoint if the runtime version supports it (>=3.13)
	supportsSmartDefault, err := python.SupportsSmartDefaultEntrypoint(ctx)
	if err != nil {
		return err
	}

	adkPresent := false
	if supportsSmartDefault {
		adkPresent, err = isAdkPresent(ctx)
		if err != nil {
			return err
		}
	}

	if !hasMain && !hasApp && !pyprojectExists && !adkPresent {
		return gcp.UserErrorf("for Python, provide a main.py or app.py file or set an entrypoint with %q env var or by creating a %q file", env.Entrypoint, "Procfile")
	}

	// If both main.py and app.py are present, we will use main.py as the python module.
	pyModule := "app:app"
	pyFile := "app.py"
	if hasMain {
		pyModule = "main:app"
		pyFile = "main.py"
	}
	cmd := []string{"gunicorn", "-b", ":8080", pyModule}

	// We will use the smart default entrypoint if the runtime version supports it (>=3.13)
	if supportsSmartDefault {
		cmd, err = smartDefaultEntrypoint(ctx, pyModule, pyFile)
		if err != nil {
			return fmt.Errorf("error detecting smart default entrypoint: %w", err)
		}
	}

	// Script command from pyproject.toml takes precedence over the smart default entrypoint.
	if python.IsPyprojectEnabled(ctx) {
		scriptCmd, err := python.GetScriptCommand(ctx)
		if err != nil {
			return fmt.Errorf("getting script command from pyproject.toml: %w", err)
		}

		isPoetry, _, err := python.IsPoetryProject(ctx)
		if err != nil {
			return fmt.Errorf("error detecting poetry project: %w", err)
		}
		isUV, _, err := python.IsUVPyproject(ctx)
		if err != nil {
			return fmt.Errorf("error detecting uv project: %w", err)
		}

		if scriptCmd != nil {
			if isPoetry {
				cmd = append([]string{"poetry", "run"}, scriptCmd...)
			} else if isUV {
				cmd = append([]string{"uv", "run"}, scriptCmd...)
			} else {
				cmd = scriptCmd
			}
		} else if !hasMain && !hasApp && !adkPresent {
			return gcp.UserErrorf("for Python with pyproject.toml, provide a main.py or app.py file or a script command in pyproject.toml or set an entrypoint with %q env var or by creating a %q file", env.Entrypoint, "Procfile")
		}
	}

	ctx.Warnf("Setting default entrypoint: %q", strings.Join(cmd, " "))
	ctx.AddProcess(gcp.WebProcess, cmd, gcp.AsDefaultProcess())

	return nil
}

func smartDefaultEntrypoint(ctx *gcp.Context, pyModule string, pyFile string) ([]string, error) {
	// To be compatible with the old builder, we will use below priority order:
	// 1. gunicorn 2. uvicorn 3. gradio 4. streamlit

	// If gunicorn is present in requirements.txt or pyproject.toml, we will use gunicorn as the entrypoint.
	gPresent, err := python.PackagePresent(ctx, gunicorn)
	if err != nil {
		return nil, fmt.Errorf("error detecting gunicorn: %w", err)
	}
	if gPresent {
		return []string{"gunicorn", "-b", ":8080", pyModule}, nil
	}
	// If uvicorn is present in requirements.txt or pyproject.toml, we will use uvicorn as the entrypoint.
	uPresent, err := python.PackagePresent(ctx, uvicorn)
	if err != nil {
		return nil, fmt.Errorf("error detecting uvicorn: %w", err)
	}
	if uPresent {
		return []string{"uvicorn", pyModule, "--port", "8080", "--host", "0.0.0.0"}, nil
	}
	// If fastapi[standard] is present in requirements.txt or pyproject.toml, we will use uvicorn as the entrypoint.
	fastapiStandardPresent, err := python.PackagePresent(ctx, fastapiStandard)
	if err != nil {
		return nil, fmt.Errorf("error detecting fastapi: %w", err)
	}
	if fastapiStandardPresent {
		return []string{"uvicorn", pyModule, "--port", "8080", "--host", "0.0.0.0"}, nil
	}
	// If gradio is present in requirements.txt or pyproject.toml, we will use gradio as the entrypoint.
	gradioPresent, err := python.PackagePresent(ctx, gradio)
	if err != nil {
		return nil, fmt.Errorf("error detecting gradio: %w", err)
	}
	if gradioPresent {
		if err := addGradioEnvVarLayer(ctx); err != nil {
			return nil, fmt.Errorf("error adding gradio env var layer: %w", err)
		}
		return []string{"python", pyFile}, nil
	}
	// If streamlit is present in requirements.txt or pyproject.toml, we will use streamlit as the entrypoint.
	sPresent, err := python.PackagePresent(ctx, streamlit)
	if err != nil {
		return nil, fmt.Errorf("error detecting streamlit: %w", err)
	}
	if sPresent {
		return []string{"streamlit", "run", pyFile, "--server.address", "0.0.0.0", "--server.port", "8080"}, nil
	}
	// If google-adk is present in requirements.txt or pyproject.toml, we will use it as the entrypoint.
	adkPresent, err := isAdkPresent(ctx)
	if err != nil {
		return nil, err
	}
	if adkPresent {
		return []string{"adk", "api_server", "--port", "8080", "--host", "0.0.0.0"}, nil
	}

	return []string{"gunicorn", "-b", ":8080", pyModule}, nil
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

func isAdkPresent(ctx *gcp.Context) (bool, error) {
	// If google-adk is present in requirements.txt, we will use it as the entrypoint.
	adkPresent, err := python.PackagePresent(ctx, googleAdk)
	if err != nil {
		return false, fmt.Errorf("error detecting google-adk: %w", err)
	}
	return adkPresent, nil
}
