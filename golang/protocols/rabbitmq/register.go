package rabbitmq

import (
	"reflect"

	"hybridpipe.io/core"
)

// Register registers the RabbitMQ protocol with the HybridPipe system.
func Register() {
	core.RegisterRouter(core.RABBITMQ, reflect.TypeOf((*Packet)(nil)).Elem())
}

// init registers the RabbitMQ protocol when the package is imported.
func init() {
	Register()
}
