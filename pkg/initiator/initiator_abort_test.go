package initiator

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// A failing abort check must stop the connect retry loop on the spot: no
// discovery, no connect exec, no further attempts. Otherwise a dial known to
// be doomed (engine recreated on another port, frontend evicted) burns the
// full retry budget while the caller holds its locks.
func TestDiscoverAndConnectAbortCheckStopsRetries(t *testing.T) {
	i, err := NewInitiator("test-vol", "", &NVMeTCPInfo{SubsystemNQN: "nqn.test:vol"}, nil)
	if err != nil {
		t.Fatalf("failed to create initiator: %v", err)
	}

	calls := 0
	i.SetAbortCheck(func() error {
		calls++
		return fmt.Errorf("engine target moved")
	})

	start := time.Now()
	_, _, err = i.discoverAndConnectNVMeTCPTarget("127.0.0.1", "4420", 15, time.Second)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected connect to abort with an error")
	}
	if !strings.Contains(err.Error(), "aborting NVMe/TCP target connect") ||
		!strings.Contains(err.Error(), "engine target moved") {
		t.Fatalf("expected abort error, got: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected a single abort-check call (no retries), got %d", calls)
	}
	// 15 attempts at 1s fixed delay would take ~14s; an abort must not wait.
	if elapsed > 2*time.Second {
		t.Fatalf("abort took %v, retry loop was not cut short", elapsed)
	}
}

// A passing abort check must not change behavior; the loop proceeds into the
// normal discovery/connect path.
func TestDiscoverAndConnectAbortCheckNilAndPassing(t *testing.T) {
	i, err := NewInitiator("test-vol", "", &NVMeTCPInfo{SubsystemNQN: "nqn.test:vol"}, nil)
	if err != nil {
		t.Fatalf("failed to create initiator: %v", err)
	}

	// Pre-set NQN skips discovery; connect against a dead local port fails
	// fast with "connection refused" from nvme-cli (or exec failure in a
	// containerized test env). Either way the error must be the connect's,
	// not an abort, and the check must have run once per attempt.
	calls := 0
	i.SetAbortCheck(func() error {
		calls++
		return nil
	})

	_, _, err = i.discoverAndConnectNVMeTCPTarget("127.0.0.1", "1", 2, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected connect to a dead port to fail")
	}
	if strings.Contains(err.Error(), "aborting NVMe/TCP target connect") {
		t.Fatalf("passing abort check must not abort, got: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected abort check on each of 2 attempts, got %d", calls)
	}
}
