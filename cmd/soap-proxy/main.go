package main

import (
	"flag"
	"log"

	"soap-proxy/internal/config"
	"soap-proxy/internal/proxy"
)

func main() {
	cfgPath := flag.String("config", "config/config.yaml", "path to YAML config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := proxy.Run(cfg); err != nil {
		log.Fatalf("exited with error: %v", err)
	}
}
