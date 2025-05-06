package netchan

import (
	"reflect"

	"hybridpipe.io/core"
)

// Register registers the NetChan protocol with the HybridPipe system.
func Register() {
	core.RegisterRouter(core.NETCHAN, reflect.TypeOf((*Packet)(nil)).Elem())
}

// init registers the NetChan protocol when the package is imported.
func init() {
	Register()
}
