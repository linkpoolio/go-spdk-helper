package initiator

import (
	"testing"
)

// A completed connect must clear exactly its own subsystem's dead siblings:
// never the freshly-connected good path (even if it still reads "connecting"
// for a moment), never live paths (legitimate multipath siblings), never
// other subsystems (other volumes).
func TestDeadSiblingControllerPaths(t *testing.T) {
	subsystems := []Subsystem{
		{
			NQN: "nqn.2023-01.io.longhorn.spdk:volume-pvc-a",
			Paths: []Path{
				{Name: "nvme3", State: "live", Address: "traddr=10.10.3.21,trsvcid=20116"},
				{Name: "nvme7", State: "connecting", Address: "traddr=10.10.3.21,trsvcid=20053"},
				{Name: "nvme9", State: "resetting", Address: "traddr=10.10.3.21,trsvcid=20012"},
				// The just-connected good path, not yet transitioned to live.
				{Name: "nvme12", State: "connecting", Address: "traddr=10.10.3.21,trsvcid=20200"},
			},
		},
		{
			NQN: "nqn.2023-01.io.longhorn.spdk:volume-pvc-b",
			Paths: []Path{
				{Name: "nvme4", State: "connecting", Address: "traddr=10.10.3.21,trsvcid=20001"},
			},
		},
	}

	dead := deadSiblingControllerPaths(subsystems, "nqn.2023-01.io.longhorn.spdk:volume-pvc-a", "10.10.3.21", "20200")
	if len(dead) != 2 {
		t.Fatalf("expected 2 dead siblings for pvc-a, got %d: %+v", len(dead), dead)
	}
	for _, p := range dead {
		switch p.Name {
		case "nvme3":
			t.Fatal("live path must never be selected")
		case "nvme12":
			t.Fatal("the just-connected good path must never be selected")
		case "nvme7", "nvme9":
			// expected
		default:
			t.Fatalf("unexpected path selected: %+v", p)
		}
	}

	// Case-insensitive live check ("LIVE" from some nvme-cli versions).
	subsystems[0].Paths[1].State = "LIVE"
	if got := deadSiblingControllerPaths(subsystems, "nqn.2023-01.io.longhorn.spdk:volume-pvc-a", "10.10.3.21", "20200"); len(got) != 1 || got[0].Name != "nvme9" {
		t.Fatalf("uppercase LIVE must be treated as live; got %+v", got)
	}

	// Unknown subsystem: nothing to clear.
	if len(deadSiblingControllerPaths(subsystems, "nqn.2023-01.io.longhorn.spdk:volume-pvc-zz", "1.2.3.4", "1")) != 0 {
		t.Fatal("no dead siblings expected for an absent subsystem")
	}

	// Healthy single-path subsystem: nothing to clear.
	healthy := []Subsystem{{
		NQN:   "nqn.2023-01.io.longhorn.spdk:volume-pvc-c",
		Paths: []Path{{Name: "nvme1", State: "live", Address: "traddr=10.10.3.21,trsvcid=20002"}},
	}}
	if len(deadSiblingControllerPaths(healthy, "nqn.2023-01.io.longhorn.spdk:volume-pvc-c", "10.10.3.21", "20002")) != 0 {
		t.Fatal("healthy subsystem must yield no dead siblings")
	}
}

// disconnectDeadSiblingControllers must be a no-op without a configured NQN —
// it can only ever clear the initiator's own subsystem.
func TestDisconnectDeadSiblingsRequiresNQN(t *testing.T) {
	i, err := NewInitiator("test-vol", "", &NVMeTCPInfo{SubsystemNQN: "nqn.test:vol"}, nil)
	if err != nil {
		t.Fatalf("failed to create initiator: %v", err)
	}
	i.NVMeTCPInfo.SubsystemNQN = ""
	// Must return without attempting enumeration (which would fail loudly in
	// a test environment without host namespaces).
	i.disconnectDeadSiblingControllers("10.0.0.1", "4420")
}

// DisconnectDeadSubsystemControllers must be a no-op for an empty NQN — it
// can only ever act on a specific volume's subsystem.
func TestDisconnectDeadSubsystemControllersRequiresNQN(t *testing.T) {
	if n := DisconnectDeadSubsystemControllers("", nil, nil); n != 0 {
		t.Fatalf("expected no disconnects for empty NQN, got %d", n)
	}
}
