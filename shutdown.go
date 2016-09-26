// Package shutdown coordinates an orderly shutdown for a process.
package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"
)

// Timeout is the maximum amount of time that the program shutdown
// should take. Once shutdown has been requested, if the program is
// still running after this amount of time it will be terminated.
var Timeout = 5 * time.Second

// Signals are the OS signals that will be interpreted as a request
// to shutdown the process. Default values depend on the operating system.
//
// To override the defaults, set this variable before calling any
// functions in this package. To disable signal handling, set to nil.
var Signals []os.Signal

// Terminate is called when the program should be terminated. The default
// value calls os.Exit(1), but the calling program can override this to
// perform additional processing before terminating.
var Terminate = func() { os.Exit(1) }

var global struct {
	mutex             sync.Mutex
	shutdownFuncs     []func()
	shutdownRequested bool
	shutdownC         <-chan struct{}
	testingResetC     chan struct{}
	shutdownCtx       context.Context
	catchSignals      func()
}

func init() {
	initContext()
	initCatchSignals()
}

func initContext() {
	ctx, cancel := context.WithCancel(context.Background())
	global.shutdownC = ctx.Done()
	global.shutdownCtx = ctx
	global.shutdownFuncs = append(global.shutdownFuncs, cancel)
	global.testingResetC = make(chan struct{})
}

func initCatchSignals() {
	var once sync.Once
	global.catchSignals = func() {
		once.Do(func() {
			if len(Signals) > 0 {
				ch := make(chan os.Signal)
				signal.Notify(ch, Signals...)

				go func() {
					for _ = range ch {
						RequestShutdown()
						return
					}
				}()
			}
		})
	}
}

// InProgress returns a channel that is closed when a graceful shutdown
// is requested.
func InProgress() <-chan struct{} {
	global.catchSignals()
	return global.shutdownC
}

// Requested returns true if shutdown has been requested.
func Requested() bool {
	select {
	case <-InProgress():
		return true
	default:
		return false
	}
}

// Context returns a background context that is canceled when a
// graceful shutdown is requested.
func Context() context.Context {
	global.catchSignals()
	return global.shutdownCtx
}

// RegisterCallback appends function f to the list of functions that will be
// called when a shutdown is requested. Function f must be safe to call from
// any goroutine.
func RegisterCallback(f func()) {
	if f == nil {
		return
	}
	global.catchSignals()
	global.mutex.Lock()
	global.shutdownFuncs = append(global.shutdownFuncs, f)
	global.mutex.Unlock()
}

// TestingReset clears all shutdown functions. Should
// only be used during testing.
func TestingReset() {
	close(global.testingResetC)
	global.mutex.Lock()
	global.shutdownFuncs = nil
	global.shutdownRequested = false
	initContext()
	global.mutex.Unlock()
}

// RequestShutdown will initiate a graceful shutdown. The InProgress
// channel will be closed, the context canceled, and any functions
// specified using RegisterCallback will be called in the order that they
// were registered.
//
// Calling this function starts a timer that times out after
// Timeout. If the program is still running after this time
// it is terminated.
func RequestShutdown() {
	var funcs []func()
	var alreadyRequested bool

	global.mutex.Lock()
	funcs = global.shutdownFuncs
	global.shutdownFuncs = nil
	alreadyRequested = global.shutdownRequested
	global.shutdownRequested = true
	global.mutex.Unlock()

	if alreadyRequested {
		return
	}

	for _, f := range funcs {
		f()
	}

	go func(resetC <-chan struct{}) {
		select {
		case <-time.After(Timeout):
			Terminate()
		case <-resetC:
			// this option is just for testing,
			// the goroutine will be terminated if TestingReset
			// is called.
			break
		}
	}(global.testingResetC)
}
