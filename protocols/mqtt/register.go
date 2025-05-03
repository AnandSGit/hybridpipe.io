package mqtt

import (
	"reflect"

	"hybridpipe.io/core"
)

// Register registers the MQTT protocol with the HybridPipe system.
func Register() {
	core.RegisterRouter(core.MQTT, reflect.TypeOf((*Packet)(nil)).Elem())
}

// init registers the MQTT protocol when the package is imported.
func init() {
	Register()
}
