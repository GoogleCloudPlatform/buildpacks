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

// Implements ruby/bundle buildpack.
// The bundle buildpack installs dependencies using bundle.
package lib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/fetch"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/fileutil"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/ruby"
	"github.com/buildpacks/libcnb/v2"
)

// source: https://rubygems.org/pages/download
var (
	rubygemsURLTemplate = "https://rubygems.org/rubygems/rubygems-%s.tgz"
	bundler1Version     = "1.17.3"
)

const (
	layerName         = "rubygems"
	dependencyHashKey = "dependency_hash"
	rubyVersionKey    = "ruby_version"
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	gemfileExists, err := ctx.FileExists("Gemfile")
	if err != nil {
		return nil, err
	}
	if gemfileExists {
		return gcp.OptInFileFound("Gemfile"), nil
	}
	gemsRbExists, err := ctx.FileExists("gems.rb")
	if err != nil {
		return nil, err
	}
	if gemsRbExists {
		return gcp.OptInFileFound("gems.rb"), nil
	}
	return gcp.OptOut("no Gemfile or gems.rb found"), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	layer, err := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayerUnlessSkipRuntimeLaunch)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}

	if err = installRubygems(ctx, layer); err != nil {
		return err
	}

	// Install bundler1 for older Ruby runtimes if required. Ruby 3.2+ does not support it.
	supportsBundler1, err := ruby.SupportsBundler1(ctx)
	if err != nil {
		return err
	}
	if supportsBundler1 {
		usingBundler1, err := isUsingBundler1(ctx)
		if err != nil {
			return err
		}
		if usingBundler1 {
			if err = installBundler1(ctx, layer); err != nil {
				return err
			}
		}
	}

	// this makes ruby use the gem and bundler from the layer, instead of the default location
	layer.SharedEnvironment.Default("RUBYLIB", filepath.Join(layer.Path, "lib"))
	// this makes gem aware of bundler in the layer
	layer.SharedEnvironment.Default("GEM_PATH", fmt.Sprintf("%s:$GEM_PATH", layer.Path))
	// this ensures gem, bundle, and bundler commands are used from the <layer>/bin
	layer.SharedEnvironment.Prepend("PATH", string(os.PathListSeparator), filepath.Join(layer.Path, "bin"))
	// stop bundler from using load to launch exec. This loads the system installed bundler otherwise
	layer.SharedEnvironment.Prepend("BUNDLE_DISABLE_EXEC_LOAD", string(os.PathListSeparator), "1")

	return nil
}

func isUsingBundler1(ctx *gcp.Context) (bool, error) {
	lockFile := ""
	exists, err := ctx.FileExists("Gemfile.lock")
	if err != nil {
		return false, err
	}
	if exists {
		lockFile = filepath.Join(ctx.ApplicationRoot(), "Gemfile.lock")
	} else {

		exists, err = ctx.FileExists("gems.locked")
		if err != nil {
			return false, err
		}
		if exists {
			lockFile = filepath.Join(ctx.ApplicationRoot(), "gems.locked")
		} else {
			return false, nil
		}
	}

	version, err := ruby.ParseBundlerVersion(lockFile)
	if err != nil {
		return false, err
	}

	return strings.HasPrefix(version, "1."), nil
}

// installBundler1 installs bundler {bundler1Version} inside the rubygems layer
func installBundler1(ctx *gcp.Context, layer *libcnb.Layer) error {
	ctx.Logf("Installing bundler %s since the Gemfile.lock BUNDLED WITH uses 1.x.x", bundler1Version)
	_, err := ctx.Exec([]string{"gem", "install", fmt.Sprintf("bundler:%s", bundler1Version), "--no-document"},
		gcp.WithEnv(fmt.Sprintf("GEM_PATH=%s", layer.Path),
			fmt.Sprintf("GEM_HOME=%s", layer.Path)),
		gcp.WithUserAttribution,
	)
	if err != nil {
		return fmt.Errorf("installing bundler %s, err: %v", bundler1Version, err)
	}

	// bundler 1.17.3 won't work if we don't remove the newer bundler that comes with rubygems
	if err := os.RemoveAll(filepath.Join(layer.Path, "lib", "bundler")); err != nil &&
		!errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.Remove(filepath.Join(layer.Path, "lib", "bundler.rb")); err != nil &&
		!errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}

// installRubygems installs a newer version of rubygems and bundler
func installRubygems(ctx *gcp.Context, layer *libcnb.Layer) error {
	tempDir, err := os.MkdirTemp(layer.Path, "rubygems")
	if err != nil {
		return fmt.Errorf("creating a temp directory, err: %q", err)
	}
	defer os.RemoveAll(tempDir)

	rubygemsVersion, bundler2Version := rubyGemsAndBundlerVersion(ctx)
	rubygemsURL := fmt.Sprintf(rubygemsURLTemplate, rubygemsVersion)
	if err = fetch.Tarball(rubygemsURL, tempDir, 1); err != nil {
		return fmt.Errorf("fetching rubygems tarball from %s, err: %q", rubygemsURL, err)
	}

	// this allows us to ship rubygems and bundler separately from the ruby runtime
	if _, err = ctx.Exec([]string{"ruby", "setup.rb", "-E", "--no-document", "--destdir", layer.Path, "--prefix", "/"},
		gcp.WithWorkDir(tempDir),
		gcp.WithUserAttribution,
	); err != nil {
		return err
	}

	// this is used to run bundler/setup
	// https://github.com/rubygems/rubygems/blob/v3.3.15/bundler/lib/bundler/shared_helpers.rb#L277
	destExe := filepath.Join(layer.Path, "exe")
	os.MkdirAll(destExe, 0755)
	if err = fileutil.MaybeCopyPathContents(
		destExe,
		filepath.Join(layer.Path, "gems", fmt.Sprintf("bundler-%s", bundler2Version), "exe"),
		fileutil.AllPaths,
	); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}

// rubyGemsAndBundlerVersion returns the rubygems and bundler2 versions to use based on the Ruby version.
func rubyGemsAndBundlerVersion(ctx *gcp.Context) (string, string) {
	rubygemsVersion := "3.3.15"
	bundler2Version := "2.3.15"

	if ruby.IsRuby25(ctx) {
		rubygemsVersion = "3.2.26"
		bundler2Version = "2.2.26"
	}

	if ruby.IsRuby4(ctx) {
		rubygemsVersion = "4.0.3"
		bundler2Version = "4.0.3"
	}

	return rubygemsVersion, bundler2Version
}
