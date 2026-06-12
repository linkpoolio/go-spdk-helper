package initiator

import (
	"fmt"
	"testing"
)

// TestCollectDevicesSkipsDevicesWithoutSubsystem reproduces the 2026-06-12
// production cascade: while one volume detaches, its block device briefly has
// no subsystem in sysfs. Enumerating devices for an unrelated healthy volume
// must skip the mid-teardown device, not fail the whole lookup.
func TestCollectDevicesSkipsDevicesWithoutSubsystem(t *testing.T) {
	healthyNQN := "nqn.2023-01.io.longhorn.spdk:volume-pvc-healthy"
	nvmeDevices := []CliDevice{
		{DevicePath: "/dev/nvme13n1", NameSpace: 1},
		// Sibling volume mid-teardown: subsystem already gone.
		{DevicePath: "/dev/nvme19n2", NameSpace: 2},
		// Device whose subsystem lookup errors transiently.
		{DevicePath: "/dev/nvme26n1", NameSpace: 1},
		// Device reporting ambiguous subsystems.
		{DevicePath: "/dev/nvme31n1", NameSpace: 1},
	}
	listSubsys := func(devicePath string) ([]Subsystem, error) {
		switch devicePath {
		case "/dev/nvme13n1":
			return []Subsystem{{
				Name: "nvme-subsys13",
				NQN:  healthyNQN,
				Paths: []Path{{
					Name:      "nvme31",
					Transport: "tcp",
					Address:   "traddr=10.205.11.217,trsvcid=20055",
					State:     "live",
				}},
			}}, nil
		case "/dev/nvme19n2":
			return []Subsystem{}, nil
		case "/dev/nvme26n1":
			return nil, fmt.Errorf("transient sysfs read failure")
		case "/dev/nvme31n1":
			return []Subsystem{{Name: "a"}, {Name: "b"}}, nil
		}
		t.Fatalf("unexpected device path %s", devicePath)
		return nil, nil
	}

	devices := collectDevices(nvmeDevices, listSubsys)

	if len(devices) != 1 {
		t.Fatalf("expected exactly 1 device, got %d: %+v", len(devices), devices)
	}
	d := devices[0]
	if d.SubsystemNQN != healthyNQN {
		t.Errorf("expected NQN %s, got %s", healthyNQN, d.SubsystemNQN)
	}
	if len(d.Controllers) != 1 || d.Controllers[0].Controller != "nvme31" {
		t.Errorf("unexpected controllers: %+v", d.Controllers)
	}
	if len(d.Namespaces) != 1 || d.Namespaces[0].NameSpace != "nvme13n1" {
		t.Errorf("unexpected namespaces: %+v", d.Namespaces)
	}
}
