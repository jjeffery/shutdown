// +build darwin linux

// TODO: add other targets, currently only UNIX variants used are
// darwin and linux.

package shutdown

import (
	"os"
	"syscall"
)

func init() {
	Signals = []os.Signal{os.Interrupt, syscall.SIGTERM}
}
