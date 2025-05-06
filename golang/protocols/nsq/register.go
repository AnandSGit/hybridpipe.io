package nsq

import (
	"reflect"

	"hybridpipe.io/core"
)

// Register registers the NSQ protocol with the HybridPipe system.
func Register() {
	core.RegisterRouter(core.NSQ, reflect.TypeOf((*Packet)(nil)).Elem())
}

// init registers the NSQ protocol when the package is imported.
func init() {
	Register()
}
