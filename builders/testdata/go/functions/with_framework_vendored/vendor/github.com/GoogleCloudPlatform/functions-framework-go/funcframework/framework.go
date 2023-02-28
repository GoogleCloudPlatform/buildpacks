// Package funcframework stubs interfaces declared by github.com/GoogleCloudPlatform/functions-framework-go/funcframework
package funcframework

import (
	"fmt"
	"net/http"
)

// Start launches a stub HTTP server on the specified port.
func Start(port string) error {
	http.ListenAndServe(fmt.Sprintf(":%s", port), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "PASS")
	}))

	return nil
}
