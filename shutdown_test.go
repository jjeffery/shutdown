package shutdown

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestRequested(t *testing.T) {
	defer TestingReset()
	if got, want := Requested(), false; got != want {
		t.Fatalf("want %v, got %v", want, got)
	}
	RequestShutdown()
	if got, want := Requested(), true; got != want {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestRegisterCallback(t *testing.T) {
	defer TestingReset()
	var count struct {
		last      int64
		callback1 int64
		callback2 int64
		callback3 int64
	}

	RegisterCallback(func() {
		count.callback1 = atomic.AddInt64(&count.last, 1)
	})
	RegisterCallback(func() {
		count.callback2 = atomic.AddInt64(&count.last, 1)
	})
	RegisterCallback(func() {
		count.callback3 = atomic.AddInt64(&count.last, 1)
	})
	RegisterCallback(nil)
	RequestShutdown()
	RequestShutdown() // second call will be ignored
	if got, want := count.callback1, int64(1); got != want {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := count.callback2, int64(2); got != want {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := count.callback3, int64(3); got != want {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestInProgress(t *testing.T) {
	defer TestingReset()

	select {
	case <-InProgress():
		t.Fatalf("InProgress indicates shutdown requested")
	default:
		break
	}

	RequestShutdown()

	select {
	case <-InProgress():
		break
	case <-time.After(time.Millisecond * 50):
		t.Errorf("InProgress not closed after shutdown requested")
	}
}

func TestContext(t *testing.T) {
	defer TestingReset()

	select {
	case <-Context().Done():
		t.Fatalf("Context indicates shutdown requested")
	default:
		break
	}

	RequestShutdown()

	select {
	case <-Context().Done():
		break
	case <-time.After(time.Millisecond * 50):
		t.Errorf("Context not closed after shutdown requested")
	}
}

func TestTerminate(t *testing.T) {
	defer TestingReset()
	var terminated bool
	Terminate = func() { terminated = true }
	Timeout = time.Millisecond * 50
	RequestShutdown()
	time.Sleep(Timeout * 2)
	if got, want := terminated, true; got != want {
		t.Errorf("want %v, got %v", want, got)
	}
}
