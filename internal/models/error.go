package models

import "encoding/json"

type ErrorResponse struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Detail  json.RawMessage `json:"detail,omitempty"`
	TraceID string          `json:"traceId,omitempty"`
}
