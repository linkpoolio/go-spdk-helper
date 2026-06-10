package client

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"testing"

	"github.com/longhorn/go-spdk-helper/pkg/jsonrpc"

	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
)

// TestEnsureNvmfTransportIdempotentWithUppercaseResponse verifies that
// ensureNvmfTransport recognises an existing transport even though SPDK
// reports trtype uppercase ("TCP") while this package's constants are
// lowercase ("tcp"), and therefore does not issue a redundant
// nvmf_create_transport call.
func TestEnsureNvmfTransportIdempotentWithUppercaseResponse(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer func() {
		_ = serverConn.Close()
	}()
	defer func() {
		_ = clientConn.Close()
	}()

	var mu sync.Mutex
	var methods []string

	go func() {
		decoder := json.NewDecoder(serverConn)
		encoder := json.NewEncoder(serverConn)
		for {
			var msg jsonrpc.Message
			if err := decoder.Decode(&msg); err != nil {
				return
			}

			mu.Lock()
			methods = append(methods, msg.Method)
			mu.Unlock()

			var result interface{}
			switch msg.Method {
			case "nvmf_get_transports":
				result = []map[string]interface{}{{"trtype": "TCP"}}
			default:
				result = true
			}
			if err := encoder.Encode(&jsonrpc.Response{ID: msg.ID, Version: "2.0", Result: result}); err != nil {
				return
			}
		}
	}()

	cli := &Client{
		conn:    clientConn,
		jsonCli: jsonrpc.NewClient(context.Background(), clientConn),
	}

	if err := cli.ensureNvmfTransport(spdktypes.NvmeTransportTypeTCP); err != nil {
		t.Fatalf("ensureNvmfTransport failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(methods) != 1 || methods[0] != "nvmf_get_transports" {
		t.Fatalf("expected a single nvmf_get_transports call, got %v", methods)
	}
}

func TestTransportTypesEqual(t *testing.T) {
	testCases := []struct {
		name     string
		a, b     spdktypes.NvmeTransportType
		expected bool
	}{
		// SPDK reports trtype uppercase in nvmf_get_transports while this
		// package's constants are lowercase; the match must be
		// case-insensitive or ensureNvmfTransport never recognises an
		// existing transport.
		{"SPDK uppercase TCP vs lowercase constant", spdktypes.NvmeTransportType("TCP"), spdktypes.NvmeTransportTypeTCP, true},
		{"SPDK uppercase RDMA vs lowercase constant", spdktypes.NvmeTransportType("RDMA"), spdktypes.NvmeTransportTypeRDMA, true},
		{"identical lowercase", spdktypes.NvmeTransportTypeTCP, spdktypes.NvmeTransportTypeTCP, true},
		{"mixed case", spdktypes.NvmeTransportType("Tcp"), spdktypes.NvmeTransportTypeTCP, true},
		{"different transports", spdktypes.NvmeTransportType("TCP"), spdktypes.NvmeTransportTypeRDMA, false},
		{"empty vs tcp", spdktypes.NvmeTransportType(""), spdktypes.NvmeTransportTypeTCP, false},
		{"both empty", spdktypes.NvmeTransportType(""), spdktypes.NvmeTransportType(""), true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := transportTypesEqual(tc.a, tc.b); got != tc.expected {
				t.Errorf("transportTypesEqual(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.expected)
			}
		})
	}
}

func TestDetectAddressFamily(t *testing.T) {
	testCases := []struct {
		name     string
		ip       string
		expected spdktypes.NvmeAddressFamily
	}{
		{"IPv4", "192.168.1.1", spdktypes.NvmeAddressFamilyIPv4},
		{"IPv6", "fd00::1", spdktypes.NvmeAddressFamilyIPv6},
		{"bracketed IPv6", "[fd00::1]", spdktypes.NvmeAddressFamilyIPv6},
		{"IPv6 loopback", "::1", spdktypes.NvmeAddressFamilyIPv6},
		{"empty", "", spdktypes.NvmeAddressFamilyIPv4},
		{"malformed", "not-an-ip", spdktypes.NvmeAddressFamilyIPv4},
		{"IPv4-mapped v6", "::ffff:10.0.0.1", spdktypes.NvmeAddressFamilyIPv4},
		{"bracketed IPv4-mapped v6", "[::ffff:10.0.0.1]", spdktypes.NvmeAddressFamilyIPv4},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := DetectAddressFamily(tc.ip)
			if got != tc.expected {
				t.Errorf("DetectAddressFamily(%q) = %q, want %q", tc.ip, got, tc.expected)
			}
		})
	}
}
