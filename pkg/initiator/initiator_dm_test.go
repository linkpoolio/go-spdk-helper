package initiator

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
)

// The exact production failure: a crash leaves a stale /dev/mapper node with
// no backing dm device; dmsetup remove then fails with ENXIO and the wrapped
// executor error must be classified as already-gone, not fatal (it wedged
// every engine-frontend restart on the volume until cleaned manually).
func TestDmsetupRemoveErrIsAlreadyGone(t *testing.T) {
	prodErr := errors.Wrapf(
		fmt.Errorf("failed to execute: /usr/bin/nsenter [nsenter --mount=/proc/1/ns/mnt dmsetup remove pvc-x], output , stderr device-mapper: remove ioctl on pvc-x  failed: No such device or address\nCommand failed.: exit status 1"),
		"failed to remove linear dm device")
	if !dmsetupRemoveErrIsAlreadyGone(prodErr) {
		t.Errorf("production ENXIO error must classify as already-gone")
	}

	if dmsetupRemoveErrIsAlreadyGone(fmt.Errorf("device or resource busy")) {
		t.Errorf("EBUSY must stay fatal")
	}
	if dmsetupRemoveErrIsAlreadyGone(nil) {
		t.Errorf("nil is not already-gone")
	}
}
