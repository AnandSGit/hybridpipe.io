package redis

import (
	"reflect"

	"hybridpipe.io/core"
)

// Register registers the Redis protocol with the HybridPipe system.
func Register() {
	core.RegisterRouter(core.REDIS, reflect.TypeOf((*Packet)(nil)).Elem())
}

// init registers the Redis protocol when the package is imported.
func init() {
	Register()
}
