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

// Implements /bin/build for java/maven buildpack.
package main

import (
	"time"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	dateFormat = time.RFC3339Nano
	// repoExpiration is an arbitrary amount of time of 10 weeks to refresh the cache layer.
	// TODO: Investigate proper cache-clearing strategy
	repoExpiration = time.Duration(time.Hour * 24 * 7 * 10)
)

type repoMetadata struct {
	ExpiryTimestamp string `toml:"expiry_timestamp"`
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if !ctx.FileExists("pom.xml") {
		ctx.OptOut("pom.xml not found.")
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	var repoMeta repoMetadata
	m2CachedRepo := ctx.Layer("m2")
	ctx.ReadMetadata(m2CachedRepo, &repoMeta)
	checkCacheExpiration(ctx, &repoMeta, m2CachedRepo)

	command := "mvn"
	if ctx.FileExists("mvnw") {
		command = "./mvnw"
	}
	ctx.ExecUser([]string{command, "clean", "package", "--batch-mode", "-DskipTests", "-Dmaven.repo.local=" + m2CachedRepo.Root})

	ctx.WriteMetadata(m2CachedRepo, &repoMeta, layers.Cache)

	return nil
}

// checkCacheExpiration clears the m2 layer and sets a new expiry timestamp when the cache is past expiration.
func checkCacheExpiration(ctx *gcp.Context, repoMeta *repoMetadata, m2CachedRepo *layers.Layer) {
	future := time.Now().Add(repoExpiration).Format(dateFormat)

	if repoMeta.ExpiryTimestamp == "" {
		ctx.ClearLayer(m2CachedRepo)
		repoMeta.ExpiryTimestamp = future
		return
	}

	t, err := time.Parse(dateFormat, repoMeta.ExpiryTimestamp)
	if err != nil {
		ctx.Debugf("Could not parse date %q, resetting expiration: %v", repoMeta.ExpiryTimestamp, err)
		ctx.ClearLayer(m2CachedRepo)
		repoMeta.ExpiryTimestamp = future
		return
	}

	if t.Before(time.Now()) {
		// Clear the local maven repo after some fixed amount of time so that it doesn't continually grow.
		ctx.ClearLayer(m2CachedRepo)
		repoMeta.ExpiryTimestamp = future
		return
	}
	return
}
