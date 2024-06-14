package main

import "log"

func main() {
	if err := executeRootCmd(); err != nil {
		log.Fatal(err)
	}
}
