package httpapi

import (
	"testing"
)

func TestAcquireInvokeSlot(t *testing.T) {
	s := &Server{InvokeSlots: NewInvokeSlots(1)}

	release, ok := s.acquireInvokeSlot()
	if !ok {
		t.Fatal("first slot should be acquired")
	}
	defer release()

	_, ok = s.acquireInvokeSlot()
	if ok {
		t.Fatal("second slot should be denied while first is held")
	}

	release()
	_, ok = s.acquireInvokeSlot()
	if !ok {
		t.Fatal("slot should be available after release")
	}
}

func TestAcquireInvokeSlotUnlimited(t *testing.T) {
	s := &Server{}
	for i := 0; i < 100; i++ {
		release, ok := s.acquireInvokeSlot()
		if !ok {
			t.Fatalf("unlimited slots should always succeed, failed at %d", i)
		}
		release()
	}
}
