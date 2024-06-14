package main

import (
	"log"

	"github.com/juancwu/konbini-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
