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

func TestIsJSONRPCRespErrorAlreadyExists(t *testing.T) {
	mk := func(code RespErrorCode) error {
		return JSONClientError{ErrorDetail: &ResponseError{Code: code, Message: "A controller named x already exists and multipath is disabled"}}
	}
	if !IsJSONRPCRespErrorAlreadyExists(mk(RespErrorCodeAlreadyExists)) {
		t.Fatal("expected -114 to match")
	}
	if IsJSONRPCRespErrorAlreadyExists(mk(RespErrorCodeNoSuchDevice)) {
		t.Fatal("-19 must not match")
	}
	if IsJSONRPCRespErrorAlreadyExists(mk(RespErrorCodeConnectionTimeout)) {
		t.Fatal("-110 must not match")
	}
	if IsJSONRPCRespErrorAlreadyExists(nil) {
		t.Fatal("nil must not match")
	}
}
