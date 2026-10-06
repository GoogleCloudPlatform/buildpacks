package cloudfunctions

import (
	"encoding/json"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

var (
	// FrameworkVersionLabel label key that stores the functions framework version.
	FrameworkVersionLabel = "functions-framework-version"
)

// FrameworkVersionInfo struct contains metadata about framework version in the container.
type FrameworkVersionInfo struct {
	// Runtime is the name of the runtime.
	Runtime string `json:"runtime"`
	// Version is the version of the functions framework dependency in the image.
	Version string `json:"version"`
	// Injected is true if the version in the image is the default added by the buildpack.
	Injected bool `json:"injected"`
}

func (fvi *FrameworkVersionInfo) String() string {
	b, _ := json.Marshal(fvi)
	return string(b)
}

// AddFrameworkVersionLabel sets the google.functions-framework-version label on the image.
func AddFrameworkVersionLabel(ctx *gcp.Context, version *FrameworkVersionInfo) {
	ctx.AddLabel(FrameworkVersionLabel, version.String())
}
