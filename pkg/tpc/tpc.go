// Package tpc provides utilities for Trusted Partner Cloud environments.
package tpc

import (
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
)

// IsTPC returns true if the build universe is set and is not GDU.
func IsTPC() bool {
	_, present := GetTarballProject()
	return present
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
