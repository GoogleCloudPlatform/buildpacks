// Package tpc provides utilities for Trusted Partner Cloud environments.
package tpc

import (
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
)

var (
	universeToProject = map[string]string{
		"prp": "tpczero-system/serverless-runtimes-tpc",
		"tsp": "eu0-system/serverless-runtimes-tpc",
		"tsq": "tpcone-system/serverless-runtimes-tpc",
		"thp": "s3ns-system/serverless-runtimes-tpc",
		"thq": "q-s3ns-system/serverless-runtimes-tpc",
	}

	arRegionToHostname = map[string]string{
		"u-us-prp1":             "docker.pkg-tpczero.goog",
		"u-germany-northeast1q": "docker.pkg-tpcone.goog",
		"u-germany-northeast1":  "docker.pkg-berlin-build0.goog",
		"u-france-east1":        "docker.s3nsregistry.fr",
		"u-france-east1q":       "docker.qs3nsregistry.fr",
	}
)

// IsTPC returns true if the build universe is set and is not GDU.
func IsTPC() bool {
	_, present := GetTarballProject()
	return present
}

// ARRegionToHostname returns the Artifact Registry hostname for a given region in TPC.
func ARRegionToHostname(region string) (string, bool) {
	hostname, present := arRegionToHostname[region]
	return hostname, present
}

// UniverseToProject returns the Artifact Registry project for a given universe.
func UniverseToProject(universe string) (string, bool) {
	project, present := universeToProject[universe]
	return project, present
}

// GetTarballProject returns the Artifact Registry project for the TPC tarball.
func GetTarballProject() (string, bool) {
	project := os.Getenv(env.TPCTarballProject)
	if project != "" {
		return project, true
	}
	return "", false
}

// GetHostname returns the hostname for the TPC build.
func GetHostname() (string, bool) {
	hostname := os.Getenv(env.TPCHostname)
	if hostname != "" {
		return hostname, true
	}
	return "", false
}
