package tcp

import (
	"reflect"

	"hybridpipe.io/core"
)

// Register registers the TCP protocol with the HybridPipe system.
func Register() {
	core.RegisterRouter(core.TCP, reflect.TypeOf((*Packet)(nil)).Elem())
}

// init registers the TCP protocol when the package is imported.
func init() {
	Register()
}
