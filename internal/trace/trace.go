package trace

import (
    "net/http"
    "time"
)

type HTTPMessage struct {
    Headers   http.Header `json:"headers"`
    Body      string      `json:"body"`
    Truncated bool        `json:"truncated"`
}

type Entry struct {
    ID            string      `json:"id"`
    StartedAt     time.Time   `json:"startedAt"`
    DurationMs    int64       `json:"durationMs"`
    ClientAddr    string      `json:"clientAddr"`
    Method        string      `json:"method"`
    Path          string      `json:"path"`
    Host          string      `json:"host"`
    StatusCode    int         `json:"statusCode"`
    SOAPAction    string      `json:"soapAction"`
    Req           HTTPMessage `json:"req"`
    Resp          HTTPMessage `json:"resp"`
    Error         string      `json:"error,omitempty"`
    SizeReqBytes  int         `json:"sizeReqBytes"`
    SizeRespBytes int         `json:"sizeRespBytes"`
}
