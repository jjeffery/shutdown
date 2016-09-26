// Package shutdown coordinates an orderly shutdown for a process.
package shutdown

import (
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
// perform additional processing before terminating. Terminate should be
// callable from any goroutine.
var Terminate = func() { os.Exit(1) }

var (
	mutex             sync.Mutex
	once              sync.Once
	shutdownFuncs     []func()
	shutdownRequested bool
	testingResetC     chan struct{}
	shutdownC         <-chan struct{}
)

func init() {
	initGlobals()
}

// initGlobals is called during initialization, and also from
// TestingReset.
func initGlobals() {
	shutdownFuncs = nil
	shutdownRequested = false
	testingResetC = make(chan struct{})
	// initContext is different for < go1.7 and >=go1.7
	initContext()
	shutdownC = shutdownCtx.Done()
}

// catchSignals initiates catching termination signals.
// It initializes once only, when one of the other public functions
// is called. This way the calling program can initialize the Signals
// variable before calling any functions in this package.
func catchSignals() {
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

// InProgress returns a channel that is closed when a graceful shutdown
// is requested.
func InProgress() <-chan struct{} {
	catchSignals()
	return shutdownC
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

// RegisterCallback appends function f to the list of functions that will be
// called when a shutdown is requested. Function f must be safe to call from
// any goroutine.
func RegisterCallback(f func()) {
	catchSignals()
	if f == nil {
		return
	}
	mutex.Lock()
	shutdownFuncs = append(shutdownFuncs, f)
	mutex.Unlock()
}

// TestingReset clears all shutdown functions. Should
// only be used during testing.
func TestingReset() {
	close(testingResetC)
	mutex.Lock()
	initGlobals()
	mutex.Unlock()
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

	mutex.Lock()
	funcs = shutdownFuncs
	shutdownFuncs = nil
	alreadyRequested = shutdownRequested
	shutdownRequested = true
	mutex.Unlock()

	if alreadyRequested {
		return
	}

	go func(testingResetC <-chan struct{}) {
		select {
		case <-time.After(Timeout):
			Terminate()
		case <-testingResetC:
			// indicates TestingReset has been called
			break
		}
	}(testingResetC)

	for _, f := range funcs {
		f()
	}
}
