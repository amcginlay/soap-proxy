package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/antchfx/xmlquery"
	"soap-proxy/internal/config"
)

// ActionHook describes the optional SOAPAction/XPath bridge.
type ActionHook struct {
	SOAPAction string
	XPath      string
	Endpoint   string
	client     *http.Client
	timeout    time.Duration
}

// newActionHooks builds ActionHooks from config entries.
func newActionHooks(cfgs []config.HookConfig) ([]*ActionHook, error) {
	hooks := make([]*ActionHook, 0, len(cfgs))
	for _, c := range cfgs {
		timeout := time.Duration(c.TimeoutSeconds) * time.Second
		hooks = append(hooks, &ActionHook{
			SOAPAction: c.SOAPAction,
			XPath:      c.XPath,
			Endpoint:   c.Endpoint,
			timeout:    timeout,
			client: &http.Client{
				Timeout: timeout,
			},
		})
	}
	return hooks, nil
}

// MaybeHandle triggers the hook asynchronously when the SOAPAction matches.
func (h *ActionHook) MaybeHandle(action string, respBody []byte) {
	if h == nil || action != h.SOAPAction {
		return
	}
	bodyCopy := append([]byte(nil), respBody...)
	go func() {
		if err := h.handle(bodyCopy); err != nil {
			log.Printf("action hook failed: %v", err)
		}
	}()
}

func (h *ActionHook) handle(respBody []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	doc, err := xmlquery.Parse(bytes.NewReader(respBody))
	if err != nil {
		return fmt.Errorf("parse XML: %w", err)
	}

	node := xmlquery.FindOne(doc, h.XPath)
	if node == nil {
		return fmt.Errorf("xpath %q not found in response", h.XPath)
	}

	value := strings.TrimSpace(node.InnerText())
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("build payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build hook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("send hook: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("hook endpoint returned status %d", resp.StatusCode)
	}

	return nil
}
