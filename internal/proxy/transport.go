package proxy

import (
    "bytes"
    "crypto/tls"
    "crypto/x509"
    "encoding/xml"
    "io"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/google/uuid"
    "soap-proxy/internal/storage"
    "soap-proxy/internal/trace"
)

const maxBodySize = 1 << 20 // 1MB

// LoggingTransport wraps a RoundTripper to capture requests and responses.
type LoggingTransport struct {
    Base  http.RoundTripper
    Store *storage.FileTraceStore
}

// NewMTLSTransport creates an http.RoundTripper using mTLS to the upstream.
func NewMTLSTransport(certFile, keyFile, caFile string) (http.RoundTripper, error) {
    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return nil, err
    }
    caBytes, err := os.ReadFile(caFile)
    if err != nil {
        return nil, err
    }
    pool := x509.NewCertPool()
    if !pool.AppendCertsFromPEM(caBytes) {
        return nil, err
    }

    cfg := &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      pool,
        MinVersion:   tls.VersionTLS12,
    }
    return &http.Transport{TLSClientConfig: cfg}, nil
}

// NewLoggingTransport constructs a LoggingTransport.
func NewLoggingTransport(base http.RoundTripper, store *storage.FileTraceStore) *LoggingTransport {
    return &LoggingTransport{Base: base, Store: store}
}

// extractSOAPAction tries, in order:
// 1) The SOAPAction HTTP header (trimmed of quotes)
// 2) The first element name inside <...:Body> in the XML request body.
func extractSOAPAction(headers http.Header, body []byte) string {
    if headers != nil {
        if h := headers.Get("SOAPAction"); h != "" {
            return strings.Trim(h, "\""")
        }
    }
    if len(body) == 0 {
        return ""
    }

    dec := xml.NewDecoder(bytes.NewReader(body))
    foundBody := false

    for {
        tok, err := dec.Token()
        if err != nil {
            return ""
        }
        switch se := tok.(type) {
        case xml.StartElement:
            if !foundBody {
                if strings.EqualFold(se.Name.Local, "Body") {
                    foundBody = true
                }
            } else {
                return se.Name.Local
            }
        }
    }
}

// RoundTrip implements http.RoundTripper and logs the request/response.
func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    id := uuid.NewString()
    start := time.Now()
    clientAddr := req.RemoteAddr

    // capture request body
    var reqBuf bytes.Buffer
    if req.Body != nil {
        _, _ = io.Copy(&reqBuf, io.LimitReader(req.Body, maxBodySize+1))
        _ = req.Body.Close()
    }
    reqBytes := reqBuf.Bytes()
    truncatedReq := len(reqBytes) > maxBodySize
    if truncatedReq {
        reqBytes = reqBytes[:maxBodySize]
    }
    req.Body = io.NopCloser(bytes.NewReader(reqBytes))

    soapAction := extractSOAPAction(req.Header, reqBytes)

    entry := trace.Entry{
        ID:         id,
        StartedAt:  start,
        ClientAddr: clientAddr,
        Method:     req.Method,
        Path:       req.URL.Path,
        Host:       req.Host,
        SOAPAction: soapAction,
        Req: trace.HTTPMessage{
            Headers:   req.Header.Clone(),
            Body:      string(reqBytes),
            Truncated: truncatedReq,
        },
        SizeReqBytes: len(reqBytes),
    }

    resp, err := t.Base.RoundTrip(req)
    entry.DurationMs = time.Since(start).Milliseconds()

    if err != nil {
        entry.Error = err.Error()
        _ = t.Store.Add(entry)
        return nil, err
    }

    var respBuf bytes.Buffer
    if resp.Body != nil {
        _, _ = io.Copy(&respBuf, io.LimitReader(resp.Body, maxBodySize+1))
        _ = resp.Body.Close()
    }
    respBytes := respBuf.Bytes()
    truncatedResp := len(respBytes) > maxBodySize
    if truncatedResp {
        respBytes = respBytes[:maxBodySize]
    }
    resp.Body = io.NopCloser(bytes.NewReader(respBytes))

    entry.StatusCode = resp.StatusCode
    entry.Resp = trace.HTTPMessage{
        Headers:   resp.Header.Clone(),
        Body:      string(respBytes),
        Truncated: truncatedResp,
    }
    entry.SizeRespBytes = len(respBytes)

    _ = t.Store.Add(entry)
    return resp, nil
}
