// Package shutdown coordinates graceful shutdown for a process.
package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/jjeffery/structlog"
)

// Timeout is the maximum amount of time that the program shutdown
// should take. Once shutdown has been requested, if the program is
// still running after this amount of time it will be terminated.
var Timeout = 5 * time.Second

// Signals are the OS signals that will be interpreted as a request
// to gracefully shutdown the process. Default values depend on the
// operating system.
//
// To override the defaults, set this variable before calling any
// functions in this package. To disable signal handling, set to nil.
var Signals []os.Signal

// Logger is used to log messages.
var Logger = structlog.DefaultLogger

var (
	mutex             sync.Mutex
	shutdownFuncs     []func()
	shutdownRequested bool
	shutdownC         <-chan struct{}
	catchSignals      func()
	shutdownCtx       context.Context
)

func init() {
	initShutdownC()
	initCatchSignals()
}

func initShutdownC() {
	ctx, cancel := context.WithCancel(context.Background())
	shutdownC = ctx.Done()
	shutdownCtx = ctx
	shutdownFuncs = append(shutdownFuncs, cancel)
}

func initCatchSignals() {
	var once sync.Once
	catchSignals = func() {
		once.Do(func() {
			if len(Signals) > 0 {
				ch := make(chan os.Signal)
				signal.Notify(ch, Signals...)

				go func() {
					for sig := range ch {
						Logger.Log("msg", "signal caught", "signal", sig.String())
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

// Context returns a background context that is canceled when a
// graceful shutdown is requested.
func Context() context.Context {
	catchSignals()
	return shutdownCtx
}

// RegisterCallback appends function f to the list of functions that will be
// called when a shutdown is requested. Function f must be safe to call from
// any goroutine.
func RegisterCallback(f func()) {
	if f == nil {
		return
	}
	catchSignals()
	mutex.Lock()
	shutdownFuncs = append(shutdownFuncs, f)
	mutex.Unlock()
}

// TestingReset clears all shutdown functions. Should
// only be used during testing.
func TestingReset() {
	mutex.Lock()
	shutdownFuncs = nil
	shutdownRequested = false
	initShutdownC()
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
		Logger.Log("msg", "shutdown already requested", "level", "warn")
		return
	}

	Logger.Log("msg", "shutdown requested")
	for _, f := range funcs {
		f()
	}

	go func() {
		time.Sleep(Timeout)
		Logger.Log("msg", "process terminated", "level", "fatal")
		os.Exit(1)
	}()
}
