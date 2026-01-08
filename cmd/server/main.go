// Package main is the entry point for the OIDC service.
package main

import (
	"log"
	"os"

	"github.com/alkem-io/oidc-service/internal/app"
	"github.com/alkem-io/oidc-service/internal/config"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("load configuration: %v", err)
		return 1
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Printf("initialize application: %v", err)
		return 1
	}
	defer func() {
		if err := application.Close(); err != nil {
			log.Printf("close application: %v", err)
		}
	}()

	if err := application.Run(); err != nil {
		log.Printf("run application: %v", err)
		return 1
	}

	return 0
}
