package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBdevNvmeSetOptionsTransportTosJSON(t *testing.T) {
	untagged, err := json.Marshal(BdevNvmeSetOptionsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(untagged), `"transport_tos"`) {
		t.Fatalf("transport_tos 0 must be omitted (SPDK default), got %s", untagged)
	}

	tagged, err := json.Marshal(BdevNvmeSetOptionsRequest{TransportTos: 96})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tagged), `"transport_tos":96`) {
		t.Fatalf("transport_tos 96 must be on the wire, got %s", tagged)
	}
}
