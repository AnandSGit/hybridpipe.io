package nats

import (
	"reflect"

	"hybridpipe.io/core"
)

// Register registers the NATS protocol with the HybridPipe system.
func Register() {
	core.RegisterRouter(core.NATS, reflect.TypeOf((*Packet)(nil)).Elem())
}

// init registers the NATS protocol when the package is imported.
func init() {
	Register()
}
