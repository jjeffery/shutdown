// +build windows

package shutdown

import "os"

func init() {
	Signals = []os.Signal{os.Interrupt}
}
