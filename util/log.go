package util

import (
	"fmt"

	"github.com/juancwu/mi/text"
)

// LogApiResponseErrs ensures a standard way to log errors from api responses.
func LogApiResponseErrs(errs []string) {
	for _, e := range errs {
		fmt.Printf("%s %s\n", text.Foreground(text.RED, "Error:"), e)
	}
}
