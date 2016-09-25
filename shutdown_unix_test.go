// +build darwin linux

package shutdown

import (
	"syscall"
	"testing"
	"time"
)

func TestSignal(t *testing.T) {
	TestingReset()

	go func() {
		time.Sleep(time.Millisecond * 50)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()

	select {
	case <-InProgress():
		break
	case <-time.After(time.Millisecond * 100):
		t.Error("shutdown not in progress")
	}
}
