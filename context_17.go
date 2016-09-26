// +build go1.7

package shutdown

import (
	"context"
)

var shutdownCtx context.Context

func initContext() {
	ctx, cancel := context.WithCancel(context.Background())
	shutdownCtx = ctx
	shutdownFuncs = append(shutdownFuncs, cancel)
}

// Context returns a context that is canceled when program shutdown 
// is requested.
func Context() context.Context {
	catchSignals()
	return shutdownCtx
}
