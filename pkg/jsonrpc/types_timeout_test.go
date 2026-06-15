package jsonrpc

import "testing"

func TestIsJSONRPCRespErrorConnectionTimeout(t *testing.T) {
	mk := func(code RespErrorCode) error {
		return JSONClientError{ErrorDetail: &ResponseError{Code: code, Message: "x"}}
	}
	if !IsJSONRPCRespErrorConnectionTimeout(mk(RespErrorCodeConnectionTimeout)) {
		t.Fatal("expected -110 to match")
	}
	if IsJSONRPCRespErrorConnectionTimeout(mk(RespErrorCodeNoSuchDevice)) {
		t.Fatal("-19 must not match")
	}
	if IsJSONRPCRespErrorConnectionTimeout(nil) {
		t.Fatal("nil must not match")
	}
}
