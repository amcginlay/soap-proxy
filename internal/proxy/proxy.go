package proxy

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"

	"soap-proxy/internal/config"
	"soap-proxy/internal/storage"
	"soap-proxy/internal/ui"
)

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

// Run starts the proxy and UI servers.
func Run(cfg *config.Config) error {
	log.Printf("config Foo=%s", cfg.Foo)

	upstreamStr := getenv("UPSTREAM_URL", "https://downstream.example.com/soap")
	proxyListen := getenv("PROXY_LISTEN", ":8080")
	uiListen := getenv("UI_LISTEN", ":8081")
	maxTraces, _ := strconv.Atoi(getenv("MAX_TRACES", "10000"))
	traceFile := getenv("TRACE_FILE", "/data/traces.jsonl")

	certFile := getenv("MTLS_CERT_FILE", "/certs/tls.crt")
	keyFile := getenv("MTLS_KEY_FILE", "/certs/tls.key")
	caFile := getenv("MTLS_CA_FILE", "/certs/ca.crt")

	upstreamURL, err := url.Parse(upstreamStr)
	if err != nil {
		return err
	}

	store, err := storage.NewFileTraceStore(traceFile, maxTraces)
	if err != nil {
		log.Printf("warning: failed to init file store (%v), traces will not persist", err)
		return err
	}
	defer store.Close()

	baseTransport, err := NewMTLSTransport(certFile, keyFile, caFile)
	if err != nil {
		return err
	}

	loggingTransport := NewLoggingTransport(baseTransport, store)

	rp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = upstreamURL.Scheme
			req.URL.Host = upstreamURL.Host
			if upstreamURL.Path != "" && upstreamURL.Path != "/" {
				req.URL.Path = singleJoiningSlash(upstreamURL.Path, req.URL.Path)
			}
			req.Host = upstreamURL.Host
		},
		Transport: loggingTransport,
	}

	// Proxy server (SOAP traffic + health)
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/", rp)
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		log.Printf("Proxy listening on %s, forwarding to %s", proxyListen, upstreamStr)
		if err := http.ListenAndServe(proxyListen, mux); err != nil {
			log.Fatalf("proxy failed: %v", err)
		}
	}()

	// UI / API server
	muxUI := http.NewServeMux()
	muxUI.HandleFunc("/api/traces", func(w http.ResponseWriter, r *http.Request) {
		traces := store.List()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(traces)
	})
	muxUI.HandleFunc("/api/traces/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/api/traces/"):]
		tr, ok := store.Get(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tr)
	})
	muxUI.HandleFunc("/", ui.Handler)

	log.Printf("UI listening on %s", uiListen)
	return http.ListenAndServe(uiListen, muxUI)
}

func singleJoiningSlash(a, b string) string {
	aslash := len(a) > 0 && a[len(a)-1] == '/'
	bslash := len(b) > 0 && b[0] == '/'
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
