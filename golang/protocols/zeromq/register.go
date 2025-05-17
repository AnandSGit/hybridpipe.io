package zeromq

import (
	"reflect"

	"hybridpipe.io/core"
)

// Register registers the ZeroMQ protocol with the HybridPipe system.
func Register() {
	core.RegisterRouter(core.ZEROMQ, reflect.TypeOf((*Packet)(nil)).Elem())
}

// init registers the ZeroMQ protocol when the package is imported.
func init() {
	Register()
}
