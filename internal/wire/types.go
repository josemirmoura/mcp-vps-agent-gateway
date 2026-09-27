package wire

import "encoding/json"

type Request struct {
	ID           string          `json:"id"`
	Subject      string          `json:"subject"`
	Tool         string          `json:"tool"`
	Resource     string          `json:"resource,omitempty"`
	Action       string          `json:"action,omitempty"`
	InvocationID string          `json:"invocation_id,omitempty"`
	GrantID      string          `json:"grant_id,omitempty"`
	AdminToken   string          `json:"admin_token,omitempty"`
	Args         json.RawMessage `json:"args,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Response struct {
	ID     string          `json:"id"`
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

func ErrorResponse(id, code, message string) Response {
	return Response{ID: id, OK: false, Error: &Error{Code: code, Message: message}}
}
