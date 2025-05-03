package qpid

import (
	"reflect"

	"hybridpipe.io/core"
)

// Register registers the Qpid protocol with the HybridPipe system.
func Register() {
	core.RegisterRouter(core.QPID, reflect.TypeOf((*Packet)(nil)).Elem())
}

// init registers the Qpid protocol when the package is imported.
func init() {
	Register()
}
