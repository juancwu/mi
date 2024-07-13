package main

import (
	"fmt"
	"os"

	"github.com/juancwu/konbini-cli/cmd"
	"github.com/juancwu/konbini-cli/text"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Printf("%s", text.Foreground(text.RED, err.Error()))
		os.Exit(1)
	}
}
