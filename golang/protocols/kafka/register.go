package kafka

import (
	"reflect"

	"hybridpipe.io/core"
)

// Register registers the Kafka protocol with the HybridPipe system.
func Register() {
	core.RegisterRouter(core.KAFKA, reflect.TypeOf((*Packet)(nil)).Elem())
}

// init registers the Kafka protocol when the package is imported.
func init() {
	Register()
}
