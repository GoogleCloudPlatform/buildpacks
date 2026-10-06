// Package functions stubs interfaces declared by github.com/GoogleCloudPlatform/functions-framework-go/functions
package functions

import "net/http"

// HTTP is a no-op stub.
func HTTP(name string, fn func(http.ResponseWriter, *http.Request)) {
	// Noop for testing.
}
