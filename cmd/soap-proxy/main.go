package main

import (
    "log"

    "soap-proxy/internal/proxy"
)

func main() {
    if err := proxy.Run(); err != nil {
        log.Fatalf("exited with error: %v", err)
    }
}
