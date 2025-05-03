package amqp

import (
	"reflect"

	"hybridpipe.io/core"
)

// Register registers the AMQP protocol with the HybridPipe system.
func Register() {
	core.RegisterRouter(core.AMQP1, reflect.TypeOf((*Packet)(nil)).Elem())
}

// init registers the AMQP protocol when the package is imported.
func init() {
	Register()
}
