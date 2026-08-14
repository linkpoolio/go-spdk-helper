package client

import (
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/longhorn/go-spdk-helper/pkg/jsonrpc"

	spdktypes "github.com/longhorn/go-spdk-helper/pkg/spdk/types"
)

func runJSONRPCRequestTest(t *testing.T, fn func(*Client) error, verify func(t *testing.T, method string, params map[string]interface{}), result interface{}) {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	defer func() {
		_ = serverConn.Close()
	}()
	defer func() {
		_ = clientConn.Close()
	}()

	serverErrCh := make(chan error, 1)
	go func() {
		defer close(serverErrCh)

		decoder := json.NewDecoder(serverConn)
		encoder := json.NewEncoder(serverConn)

		var msg jsonrpc.Message
		if err := decoder.Decode(&msg); err != nil {
			serverErrCh <- err
			return
		}

		params, ok := msg.Params.(map[string]interface{})
		if !ok {
			serverErrCh <- nil
			t.Errorf("unexpected params type %T", msg.Params)
			return
		}

		verify(t, msg.Method, params)

		if err := encoder.Encode(&jsonrpc.Response{ID: msg.ID, Version: "2.0", Result: result}); err != nil {
			serverErrCh <- err
			return
		}

		serverErrCh <- nil
	}()

	cli := &Client{
		conn:    clientConn,
		jsonCli: jsonrpc.NewClient(context.Background(), clientConn),
	}

	if err := fn(cli); err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if err := <-serverErrCh; err != nil {
		t.Fatalf("server verification failed: %v", err)
	}
}

func TestBdevAioCreateNoWaitTriState(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	testCases := []struct {
		name string
		// nowait passed to BdevAioCreate
		nowait *bool
		// expectPresent indicates whether the "nowait" key must be on the wire
		expectPresent bool
		expectValue   bool
	}{
		{"nil omits nowait so SPDK default applies", nil, false, false},
		{"explicit true is sent", boolPtr(true), true, true},
		{"explicit false is sent", boolPtr(false), true, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runJSONRPCRequestTest(t,
				func(cli *Client) error {
					_, err := cli.BdevAioCreate("/dev/test", "aio0", 4096, tc.nowait)
					return err
				},
				func(t *testing.T, method string, params map[string]interface{}) {
					t.Helper()
					if method != "bdev_aio_create" {
						t.Fatalf("unexpected method %s", method)
					}
					value, present := params["nowait"]
					if present != tc.expectPresent {
						t.Fatalf("expected nowait present=%v, got present=%v (value %#v)", tc.expectPresent, present, value)
					}
					if present {
						boolValue, ok := value.(bool)
						if !ok {
							t.Fatalf("expected nowait to be a bool, got %T", value)
						}
						if boolValue != tc.expectValue {
							t.Fatalf("expected nowait=%v, got %v", tc.expectValue, boolValue)
						}
					}
				},
				"aio0",
			)
		})
	}
}

func TestIobufSetOptionsSendsBufsizesInBytes(t *testing.T) {
	runJSONRPCRequestTest(t,
		func(cli *Client) error {
			_, err := cli.IobufSetOptions(16384, 2048, 8192, 135168)
			return err
		},
		func(t *testing.T, method string, params map[string]interface{}) {
			t.Helper()
			if method != "iobuf_set_options" {
				t.Fatalf("unexpected method %s", method)
			}
			if params["small_pool_count"] != float64(16384) {
				t.Fatalf("expected small_pool_count 16384, got %#v", params["small_pool_count"])
			}
			if params["large_pool_count"] != float64(2048) {
				t.Fatalf("expected large_pool_count 2048, got %#v", params["large_pool_count"])
			}
			// Bufsizes are bytes and must be passed through verbatim, not
			// multiplied by 1024.
			if params["small_bufsize"] != float64(8192) {
				t.Fatalf("expected small_bufsize 8192, got %#v", params["small_bufsize"])
			}
			if params["large_bufsize"] != float64(135168) {
				t.Fatalf("expected large_bufsize 135168, got %#v", params["large_bufsize"])
			}
		},
		true,
	)
}

func TestNvmfSubsystemAddNsUsesDefaultANAGroup(t *testing.T) {
	runJSONRPCRequestTest(t,
		func(cli *Client) error {
			_, err := cli.NvmfSubsystemAddNsWithUUID("nqn.test", "bdev0", "nguid0", "")
			return err
		},
		func(t *testing.T, method string, params map[string]interface{}) {
			t.Helper()
			if method != "nvmf_subsystem_add_ns" {
				t.Fatalf("unexpected method %s", method)
			}

			namespace, ok := params["namespace"].(map[string]interface{})
			if !ok {
				t.Fatalf("unexpected namespace payload %T", params["namespace"])
			}
			if _, exists := namespace["anagrpid"]; exists {
				t.Fatalf("expected add_ns to omit anagrpid, got %#v", namespace["anagrpid"])
			}
		},
		float64(1),
	)
}

func TestNvmfSubsystemListenerSetANAStateDefaultsANAGroup(t *testing.T) {
	runJSONRPCRequestTest(t,
		func(cli *Client) error {
			_, err := cli.NvmfSubsystemListenerSetANAState(
				"nqn.test",
				"10.0.0.1",
				"20006",
				spdktypes.NvmeTransportTypeTCP,
				spdktypes.NvmeAddressFamilyIPv4,
				spdktypes.NvmfSubsystemListenerAnaStateOptimized,
				0,
			)
			return err
		},
		func(t *testing.T, method string, params map[string]interface{}) {
			t.Helper()
			if method != "nvmf_subsystem_listener_set_ana_state" {
				t.Fatalf("unexpected method %s", method)
			}
			if params["anagrpid"] != float64(spdktypes.DefaultNvmfANAGroupID) {
				t.Fatalf("expected anagrpid %d, got %#v", spdktypes.DefaultNvmfANAGroupID, params["anagrpid"])
			}
		},
		true,
	)
}

func TestBdevLvolCreateLvstoreRPCRequests(t *testing.T) {
	t.Run("omits ratio when zero so SPDK keeps its default of 100", func(t *testing.T) {
		runJSONRPCRequestTest(t,
			func(cli *Client) error {
				_, err := cli.BdevLvolCreateLvstore("aio0", "lvs0", 33554432, 0)
				return err
			},
			func(t *testing.T, method string, params map[string]interface{}) {
				t.Helper()
				if method != "bdev_lvol_create_lvstore" {
					t.Fatalf("unexpected method %s", method)
				}
				if params["bdev_name"] != "aio0" || params["lvs_name"] != "lvs0" {
					t.Fatalf("unexpected identity params %#v", params)
				}
				if params["cluster_sz"] != float64(33554432) {
					t.Fatalf("expected cluster_sz 33554432, got %#v", params["cluster_sz"])
				}
				if _, ok := params["num_md_pages_per_cluster_ratio"]; ok {
					t.Fatalf("ratio should be omitted at 0, got %#v", params["num_md_pages_per_cluster_ratio"])
				}
			},
			"uuid-0",
		)
	})

	t.Run("sends explicit ratio", func(t *testing.T) {
		runJSONRPCRequestTest(t,
			func(cli *Client) error {
				_, err := cli.BdevLvolCreateLvstore("aio0", "lvs0", 33554432, 400)
				return err
			},
			func(t *testing.T, method string, params map[string]interface{}) {
				t.Helper()
				if method != "bdev_lvol_create_lvstore" {
					t.Fatalf("unexpected method %s", method)
				}
				if params["num_md_pages_per_cluster_ratio"] != float64(400) {
					t.Fatalf("expected ratio 400, got %#v", params["num_md_pages_per_cluster_ratio"])
				}
			},
			"uuid-0",
		)
	})
}
