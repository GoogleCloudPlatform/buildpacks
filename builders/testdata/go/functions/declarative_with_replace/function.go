// Package hello contains a function with go.mod using replace directive.
package hello

import (
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"rsc.io/quote"
)

func init() {
	functions.HTTP("Func", myFunc)
}

func myFunc(w http.ResponseWriter, r *http.Request) {
	if quote.Hello() == "Forked!" {
		fmt.Fprintln(w, "PASS")
		return
	}
	fmt.Fprintln(w, "FAIL")
}
