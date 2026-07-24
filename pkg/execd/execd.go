// Copyright 2026 Google LLC
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

// Package execd provides an interface for installing exec.d scripts.
package execd

import (
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

// InstallerCapability is the key used to inject the ExecDInstaller capability.
const InstallerCapability = "execd.Installer"

// Installer defines the interface for installing exec.d scripts.
// This allows abstracting the installation logic so that it can be swapped out
// for the "maker" use case, avoiding file access to missing buildpack directories.
type Installer interface {
	Install(ctx *gcp.Context, layerName string, scriptRelPath string) error
}

// MakerInstaller implements the Installer interface for the maker tool.
// It performs a no-op instead of trying to copy scripts from the buildpack root,
// which is not available in the maker environment.
type MakerInstaller struct{}

// Install does nothing, avoiding the "file not found" error in maker mode.
func (i MakerInstaller) Install(ctx *gcp.Context, layerName string, scriptRelPath string) error {
	// No-op for maker.
	return nil
}
