package main

import (
	"fmt"
	"os"

	"github.com/juancwu/mi/cmd"
	"github.com/juancwu/mi/common"
	"github.com/juancwu/mi/text"
)

var version string = "dev"

func main() {
	// make it accessible globally
	common.Version = version
	if err := cmd.Execute(version); err != nil {
		fmt.Printf("%s\n", text.Foreground(text.RED, err.Error()))
		os.Exit(1)
	}
}
