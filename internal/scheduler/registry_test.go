package scheduler

import "testing"

func TestConnRegistryPushesToMultipleConnections(t *testing.T) {
	reg := NewConnRegistry()
	ch1 := reg.Register("thread-1")
	ch2 := reg.Register("thread-1")

	if ok := reg.Push("thread-1", ReminderEvent{ID: "rem-1", ThreadID: "thread-1", Message: "hi"}); !ok {
		t.Fatal("Push returned false, want true")
	}

	for name, ch := range map[string]chan ReminderEvent{"ch1": ch1, "ch2": ch2} {
		select {
		case got := <-ch:
			if got.ID != "rem-1" {
				t.Fatalf("%s got ID=%q, want rem-1", name, got.ID)
			}
		default:
			t.Fatalf("%s did not receive event", name)
		}
	}
}

func TestConnRegistryUnregisterRemovesOnlyOneConnection(t *testing.T) {
	reg := NewConnRegistry()
	ch1 := reg.Register("thread-1")
	ch2 := reg.Register("thread-1")

	reg.Unregister("thread-1", ch1)
	if _, ok := <-ch1; ok {
		t.Fatal("ch1 still open after unregister")
	}

	if ok := reg.Push("thread-1", ReminderEvent{ID: "rem-2", ThreadID: "thread-1"}); !ok {
		t.Fatal("Push returned false, want true")
	}
	select {
	case got := <-ch2:
		if got.ID != "rem-2" {
			t.Fatalf("ch2 got ID=%q, want rem-2", got.ID)
		}
	default:
		t.Fatal("ch2 did not receive event")
	}
}
