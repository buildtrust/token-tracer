package main

import (
	"log"

	"github.com/buildtrust/token-tracer/config"
)

func main() {
	if err := config.Init(); err != nil {
		log.Fatalf("init config error: %v\n", err)
	}
}
